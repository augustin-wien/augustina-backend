// Package jobs holds background maintenance tasks that run for the lifetime of the process.
package jobs

import (
	"errors"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/paymentprovider"
	"github.com/augustin-wien/augustina-backend/utils"
)

var log = utils.GetLogger()

// paymentTimeout matches the PaymentTimeout (seconds) VivaWallet is given in
// CreatePaymentOrder: an order younger than this is still a normal in-flight
// checkout, not yet a reconciliation candidate.
const paymentTimeout = 5 * time.Minute

// invalidateTimeout is how long an order can sit unverified before it is
// judged permanently abandoned (see invalidateIfAbandoned). Deliberately
// longer than paymentTimeout: an order gets a full 20 minutes of retries
// before GetUnverifiedOrders (and the backoffice screen backed by it) stops
// showing it.
const invalidateTimeout = 20 * time.Minute

// verifyDelay is how old a paid-but-unverified order has to be before the job verifies it
// itself. VivaWallet can deliver its webhook several minutes after the payment.
const verifyDelay = 20 * time.Minute

// placeholderTransactionIDPrefix marks the transaction id CreateOrder stores until
// VivaWallet reports back (see database/queries.orders.go). An order still carrying it
// never had a real transaction id, so there is nothing to look up at VivaWallet.
const placeholderTransactionIDPrefix = "manual-"

// StartOrderReconciliation runs a background job that periodically checks orders still
// marked unverified against VivaWallet directly. This catches the case that produces no
// error log on its own: VivaWallet's webhook never arrives (or is lost, or failed) even
// though the customer did pay, so the order sits verified=false indefinitely. Such an
// order is verified with the same checks the webhook runs; if that fails, the failure is
// stored on the order and reported once.
func StartOrderReconciliation() {
	interval := config.Config.OrderReconcileIntervalMinutes
	if interval <= 0 {
		log.Info("StartOrderReconciliation: disabled (ORDER_RECONCILE_INTERVAL_MINUTES <= 0)")
		return
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	go func() {
		for range ticker.C {
			reconcileUnverifiedOrders()
		}
	}()
}

func reconcileUnverifiedOrders() {
	orders, err := database.Db.GetUnverifiedOrders()
	if err != nil {
		log.Errorw("reconcileUnverifiedOrders: failed to load unverified orders", "error", err)
		return
	}

	for _, order := range orders {
		if order.TransactionID == "" || strings.HasPrefix(order.TransactionID, placeholderTransactionIDPrefix) {
			// Never even got as far as VivaWallet redirecting back with a transaction id,
			// so there is nothing to check with VivaWallet - abandoned checkouts like this
			// make up the bulk of this table's history, purely on age.
			invalidateIfAbandoned(order.ID, order.Timestamp)
			continue
		}
		if time.Since(order.Timestamp) < paymentTimeout {
			// Still a normal in-flight checkout window, not a reconciliation candidate yet.
			continue
		}

		viva, err := paymentprovider.VerifyTransactionID(order.TransactionID, false)
		if err == nil {
			// VivaWallet confirms the transaction succeeded, but the order is still
			// unverified: the webhook never arrived, was lost or failed. Never
			// invalidated, however old it gets.
			if time.Since(order.Timestamp) >= verifyDelay {
				verifyPaidOrder(order, viva)
			}
			continue
		}

		if errors.Is(err, paymentprovider.ErrTransactionNotSuccessful) {
			// VivaWallet positively confirms this transaction did not succeed - unlike a
			// network/auth/unexpected-status error, which could just mean VivaWallet was
			// unreachable this cycle and tells us nothing about the payment itself, this
			// is safe to treat as genuinely abandoned.
			invalidateIfAbandoned(order.ID, order.Timestamp)
		}
	}
}

// verifyPaidOrder verifies an order VivaWallet confirmed as paid. A failure is stored on
// the order and reported the first time only; the next runs keep trying.
func verifyPaidOrder(order database.Order, viva paymentprovider.TransactionVerificationResponse) {
	reason, err := paymentprovider.ReconcilePaidOrder(order.ID, viva)
	if err == nil {
		return
	}
	first, dbErr := database.Db.SetOrderReconcileFailed(order.ID, reason+": "+err.Error())
	if dbErr != nil {
		log.Errorw("verifyPaidOrder: failed to store reconcile failure", "order_id", order.ID, "error", dbErr)
	}
	if !first {
		return
	}
	log.Errorw(
		"reconcileUnverifiedOrders: order paid at VivaWallet could not be verified",
		"order_id", order.ID,
		"order_code", order.OrderCode.String,
		"transaction_id", order.TransactionID,
		"order_timestamp", order.Timestamp,
		"age", time.Since(order.Timestamp).Round(time.Minute).String(),
		"reason", reason,
		"error", err,
		"viva_status_id", viva.StatusID,
		"viva_amount", viva.Amount,
	)
}

// invalidateIfAbandoned marks an order invalidated once it has had a full
// invalidateTimeout to resolve. Kept as a separate, deliberately narrow check
// so nothing upstream of it can invalidate an order VivaWallet confirmed was paid.
func invalidateIfAbandoned(orderID int, timestamp time.Time) {
	if time.Since(timestamp) < invalidateTimeout {
		return
	}
	if err := database.Db.InvalidateOrder(orderID); err != nil {
		log.Errorw("invalidateIfAbandoned: failed to invalidate order", "order_id", orderID, "error", err)
	}
}

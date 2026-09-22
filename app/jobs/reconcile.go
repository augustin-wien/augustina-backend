// Package jobs holds background maintenance tasks that run for the lifetime of the process.
package jobs

import (
	"errors"
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

// StartOrderReconciliation runs a background job that periodically checks orders still
// marked unverified against VivaWallet directly. This exists to catch the case that
// produces no error log on its own: VivaWallet's webhook never arrives (or is lost)
// even though the customer did pay, so the order sits verified=false indefinitely,
// indistinguishable from a customer who simply abandoned checkout. Deliberately
// alert-only: it flags the mismatch so a human can confirm and re-verify, rather than
// silently finalizing payment records from a background job.
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
		if order.TransactionID == "" {
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

		_, err := paymentprovider.VerifyTransactionID(order.TransactionID, false)
		if err == nil {
			// VivaWallet confirms the transaction succeeded, but our own database still
			// shows the order unverified: the webhook that should have finalized it never
			// arrived or was lost. This is the one case a log-based alert can never catch
			// on its own, since no error was ever logged for it before now. Never
			// invalidated, however old it gets - a human has to resolve this one.
			log.Errorw(
				"reconcileUnverifiedOrders: order paid at VivaWallet but not verified locally",
				"order_id", order.ID,
				"transaction_id", order.TransactionID,
			)
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

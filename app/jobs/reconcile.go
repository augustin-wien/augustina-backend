// Package jobs holds background maintenance tasks that run for the lifetime of the process.
package jobs

import (
	"errors"
	"strings"
	"sync"
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

// realertInterval is how often the same paid-but-unverified order is alerted again. Without
// it every reconcile run re-alerts every such order until a human resolves it, burying new
// cases under repeats of old ones.
const realertInterval = 24 * time.Hour

// placeholderTransactionIDPrefix marks the transaction id CreateOrder stores until
// VivaWallet reports back (see database/queries.orders.go). An order still carrying it
// never had a real transaction id, so there is nothing to look up at VivaWallet.
const placeholderTransactionIDPrefix = "manual-"

var (
	alertedMu   sync.Mutex
	lastAlerted = map[int]time.Time{}
)

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

	forgetResolvedAlerts(orders)

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
			// VivaWallet confirms the transaction succeeded, but our own database still
			// shows the order unverified: the webhook that should have finalized it never
			// arrived or was lost. This is the one case a log-based alert can never catch
			// on its own, since no error was ever logged for it before now. Never
			// invalidated, however old it gets - a human has to resolve this one.
			if shouldAlert(order.ID, time.Now()) {
				log.Errorw(
					"reconcileUnverifiedOrders: order paid at VivaWallet but not verified locally",
					"order_id", order.ID,
					"order_code", order.OrderCode.String,
					"transaction_id", order.TransactionID,
					"order_timestamp", order.Timestamp,
					"age", time.Since(order.Timestamp).Round(time.Minute).String(),
					"viva_status_id", viva.StatusID,
					"viva_amount", viva.Amount,
					"viva_order_code", viva.OrderCode.String(),
					"viva_ins_date", viva.InsDate,
				)
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

// shouldAlert reports whether a paid-but-unverified order is due an alert: the first time
// it is seen, then again once every realertInterval while it stays unresolved.
func shouldAlert(orderID int, now time.Time) bool {
	alertedMu.Lock()
	defer alertedMu.Unlock()

	if last, ok := lastAlerted[orderID]; ok && now.Sub(last) < realertInterval {
		return false
	}
	lastAlerted[orderID] = now
	return true
}

// forgetResolvedAlerts drops alert state for orders that are no longer unverified, so a
// resolved order does not hold memory for the lifetime of the process.
func forgetResolvedAlerts(unverified []database.Order) {
	stillOpen := make(map[int]bool, len(unverified))
	for _, o := range unverified {
		stillOpen[o.ID] = true
	}

	alertedMu.Lock()
	defer alertedMu.Unlock()
	for id := range lastAlerted {
		if !stillOpen[id] {
			delete(lastAlerted, id)
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

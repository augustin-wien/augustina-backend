// Package jobs holds background maintenance tasks that run for the lifetime of the process.
package jobs

import (
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
			continue
		}
		if time.Since(order.Timestamp) < paymentTimeout {
			// Still a normal in-flight checkout window, not a reconciliation candidate yet.
			continue
		}

		if _, err := paymentprovider.VerifyTransactionID(order.TransactionID, false); err == nil {
			// VivaWallet confirms the transaction succeeded, but our own database still
			// shows the order unverified: the webhook that should have finalized it never
			// arrived or was lost. This is the one case a log-based alert can never catch
			// on its own, since no error was ever logged for it before now.
			log.Errorw(
				"reconcileUnverifiedOrders: order paid at VivaWallet but not verified locally",
				"order_id", order.ID,
				"transaction_id", order.TransactionID,
			)
		}
		// A non-nil error here is the normal case: either the customer genuinely never
		// completed payment (abandoned checkout), or a transient VivaWallet API error -
		// either way it is not evidence of a lost webhook, so it needs no alert.
	}
}

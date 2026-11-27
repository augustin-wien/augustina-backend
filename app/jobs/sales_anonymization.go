package jobs

import (
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/reports"
)

// salesAnonymizationInterval is how often the job checks for sales to anonymize
const salesAnonymizationInterval = time.Hour

// payoutRetentionYears is how long payouts stay assigned to the vendor (BAO § 132), counted from
// the end of the calendar year of the payout
const payoutRetentionYears = 7

// StartSalesAnonymization runs a background job that anonymizes old sales as configured in the
// settings: SalesAnonymizationEnabled removes customer data, VendorAnonymizationEnabled detaches
// paid out sales from the vendors. Both are off by default.
func StartSalesAnonymization() {
	go func() {
		anonymizeOldSales(time.Now())
		ticker := time.NewTicker(salesAnonymizationInterval)
		for now := range ticker.C {
			anonymizeOldSales(now)
		}
	}()
}

// anonymizeOldSales runs the anonymizations that are enabled in the settings
func anonymizeOldSales(now time.Time) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		log.Errorw("anonymizeOldSales: failed to load the settings", "error", err)
		return
	}
	if !settings.SalesAnonymizationEnabled && !settings.VendorAnonymizationEnabled {
		return
	}
	days := settings.AnonymizationRetentionDays
	if days < database.MinAnonymizationRetentionDays {
		log.Errorw("anonymizeOldSales: retention too short, skipped", "days", days)
		return
	}
	cutoff := now.AddDate(0, 0, -days).UTC()

	if settings.VendorAnonymizationEnabled {
		local := now.In(reports.Location)
		payoutCutoff := time.Date(local.Year()-payoutRetentionYears, 1, 1, 0, 0, 0, 0, reports.Location).UTC()
		res, err := database.Db.AnonymizeVendorData(cutoff, payoutCutoff)
		if err != nil {
			log.Errorw("anonymizeOldSales: failed to detach sales from vendors", "error", err)
		} else if res.Total() > 0 {
			log.Infow("anonymizeOldSales: detached paid out sales from vendors",
				"payments", res.Payments, "orders", res.Orders, "payouts", res.Payouts)
		}
	}

	if settings.SalesAnonymizationEnabled {
		res, err := database.Db.AnonymizeSalesData(cutoff)
		if err != nil {
			log.Errorw("anonymizeOldSales: failed to remove customer data", "error", err)
		} else if res.Total() > 0 {
			log.Infow("anonymizeOldSales: removed customer data of old sales",
				"orders", res.Orders, "vouchers", res.Vouchers, "mails", res.MailTasks,
				"external_payments", res.ExternalPayments)
		}
	}
}

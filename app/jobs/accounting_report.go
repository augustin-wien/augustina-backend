package jobs

import (
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/reports"
)

// accountingReportCheckInterval is how often the job checks whether a month's report is due
const accountingReportCheckInterval = time.Hour

// accountingReportSendHour is the hour (Europe/Vienna) on the first of the month from which the
// report of the previous month is sent, so late webhooks of the last evening are booked by then
const accountingReportSendHour = 6

// sendAccountingReport is replaced in tests
var sendAccountingReport = func(month string) error {
	return reports.SendMonthly(month, mailer.SendWithRetry)
}

// StartMonthlyAccountingReport runs a background job that mails the report of the previous month
// to the accounting once the month is over. The last sent month is stored in the database, so a
// restart neither sends a report twice nor skips one.
func StartMonthlyAccountingReport() {
	go func() {
		sendDueAccountingReport(time.Now())
		ticker := time.NewTicker(accountingReportCheckInterval)
		for now := range ticker.C {
			sendDueAccountingReport(now)
		}
	}()
}

// sendDueAccountingReport sends the report of the month before now unless it was already sent
func sendDueAccountingReport(now time.Time) {
	now = now.In(reports.Location)
	if now.Day() == 1 && now.Hour() < accountingReportSendHour {
		return
	}
	month := reports.PreviousMonth(now)

	sent, err := database.Db.GetAccountingReportSentMonth()
	if err != nil {
		log.Errorw("sendDueAccountingReport: failed to load the last sent month", "error", err)
		return
	}
	if sent >= month {
		return
	}

	err = sendAccountingReport(month)
	if errors.Is(err, reports.ErrNoRecipients) {
		// Not configured; sent as soon as someone fills in the address
		return
	}
	if err != nil {
		log.Errorw("sendDueAccountingReport: failed to send the accounting report", "month", month, "error", err)
		return
	}
	if err = database.Db.SetAccountingReportSentMonth(month); err != nil {
		log.Errorw("sendDueAccountingReport: failed to store the sent month", "month", month, "error", err)
		return
	}
	log.Infow("sendDueAccountingReport: accounting report sent", "month", month)
}

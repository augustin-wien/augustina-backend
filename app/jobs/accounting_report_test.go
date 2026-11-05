package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/reports"
	"github.com/stretchr/testify/require"
)

func TestSendDueAccountingReport(t *testing.T) {
	var sent []string
	sendErr := error(nil)
	old := sendAccountingReport
	t.Cleanup(func() { sendAccountingReport = old })
	sendAccountingReport = func(month string) error {
		if sendErr == nil {
			sent = append(sent, month)
		}
		return sendErr
	}
	require.NoError(t, database.Db.SetAccountingReportSentMonth(""))
	t.Cleanup(func() { _ = database.Db.SetAccountingReportSentMonth("") })

	at := func(month time.Month, day, hour int) time.Time {
		return time.Date(2026, month, day, hour, 0, 0, 0, reports.Location)
	}

	// Too early on the first of the month
	sendDueAccountingReport(at(11, 1, 5))
	require.Empty(t, sent)

	// Not configured: nothing sent, so it is sent once the address is filled in
	sendErr = reports.ErrNoRecipients
	sendDueAccountingReport(at(11, 1, 6))
	sendErr = nil
	month, err := database.Db.GetAccountingReportSentMonth()
	require.NoError(t, err)
	require.Equal(t, "", month)

	// A failed mail is tried again on the next run
	sendErr = errors.New("smtp down")
	sendDueAccountingReport(at(11, 1, 7))
	sendErr = nil
	sendDueAccountingReport(at(11, 1, 8))
	require.Equal(t, []string{"2026-10"}, sent)

	// Sent once only
	sendDueAccountingReport(at(11, 1, 9))
	sendDueAccountingReport(at(11, 20, 9))
	require.Equal(t, []string{"2026-10"}, sent)
	month, err = database.Db.GetAccountingReportSentMonth()
	require.NoError(t, err)
	require.Equal(t, "2026-10", month)

	// Next month
	sendDueAccountingReport(at(12, 1, 6))
	require.Equal(t, []string{"2026-10", "2026-11"}, sent)
}

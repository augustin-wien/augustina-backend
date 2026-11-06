package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/reports"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestMonthlyReport(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	origSend := mailer.Send
	defer func() { mailer.Send = origSend }()
	var sent []*mailer.EmailRequest
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		sent = append(sent, r)
		return true, nil
	}

	vendorLicenseID := "testmonthlyreport"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)
	cashAccountID, err := database.Db.GetAccountTypeID("Cash")
	require.NoError(t, err)
	require.NoError(t, database.Db.CreatePayments([]database.Payment{
		{Sender: vendorAccount.ID, Receiver: cashAccountID, Amount: 1250},
	}))

	month := time.Now().In(reports.Location).Format("2006-01")
	url := "/api/reports/monthly/" + month + "/"

	utils.TestRequest(t, r, "GET", url, nil, 401)
	utils.TestRequestWithAuth(t, r, "GET", "/api/reports/monthly/2026-13/", nil, 400, adminUserToken)

	res := utils.TestRequestWithAuth(t, r, "GET", url, nil, 200, adminUserToken)
	var report reports.MonthlyReport
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &report))
	require.Equal(t, month, report.Month)
	require.Equal(t, 1250, report.Payouts)
	require.Equal(t, 1, report.PayoutCount)

	res = utils.TestRequestWithAuth(t, r, "GET", url+"csv/", nil, 200, adminUserToken)
	require.Equal(t, "text/csv; charset=utf-8", res.Header().Get("Content-Type"))
	require.Contains(t, res.Header().Get("Content-Disposition"), "monatsbericht_"+month+".csv")
	require.Contains(t, res.Body.String(), ";12,50;1")

	res = utils.TestRequestWithAuth(t, r, "GET", url+"pdf/", nil, 200, adminUserToken)
	require.Equal(t, "application/pdf", res.Header().Get("Content-Type"))
	require.True(t, strings.HasPrefix(res.Body.String(), "%PDF-"))

	// Not sent without a recipient
	utils.TestRequestWithAuth(t, r, "POST", url+"send/", nil, 400, adminUserToken)
	require.Empty(t, sent)

	require.NoError(t, database.Db.EntClient.Settings.Update().
		SetAccountingReportEmail("buchhaltung@example.com").
		Exec(context.Background()))
	utils.TestRequestWithAuth(t, r, "POST", url+"send/", nil, 200, adminUserToken)
	require.Len(t, sent, 1)
	msg, _, err := sent[0].BuildMessage()
	require.NoError(t, err)
	require.Contains(t, string(msg), "To: buchhaltung@example.com\r\n")
	require.Len(t, sent[0].Attachments(), 2)
}

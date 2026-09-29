package handlers

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent/pdfdownload"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestResendOrderMail resends the online paper and PDF mails of a digital sale
func TestResendOrderMail(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	require.NoError(t, database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Digital", `<a href="{{.URL}}">read</a>`))
	require.NoError(t, database.Db.CreateOrUpdateMailTemplate("PDFLicenceItemTemplate.html", "PDF", `<a href="{{.URL}}">download</a>`))
	require.NoError(t, database.Db.BackfillOnlinePaperUrl("https://paper.test"))
	config.Config.FrontendURL = "https://frontend.test"

	origSend := mailer.Send
	defer func() { mailer.Send = origSend }()
	var sentBodies []string
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		sentBodies = append(sentBodies, r.Body())
		return true, nil
	}

	vendorLicenseID := "testresendordermail"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	licenseID, err := database.Db.CreateItem(database.Item{Name: "resend-license", Description: "resend-license", Price: 1, IsLicenseItem: true, Type: "license_item"})
	require.NoError(t, err)
	digitalID, err := database.Db.CreateItem(database.Item{Name: "resend-digital", Description: "resend-digital", Price: 300, LicenseItem: null.IntFrom(int64(licenseID)), Type: "online_issue"})
	require.NoError(t, err)
	pdfLicenseID, err := database.Db.CreateItem(database.Item{Name: "resend-pdf-license", Description: "resend-pdf-license", Price: 1, IsLicenseItem: true, Type: "license_item"})
	require.NoError(t, err)
	pdfID, err := database.Db.CreatePDF(database.PDF{Path: "resend.pdf", Timestamp: time.Now()})
	require.NoError(t, err)
	pdfItemID, err := database.Db.CreateItem(database.Item{Name: "resend-pdf", Description: "resend-pdf", Price: 300, LicenseItem: null.IntFrom(int64(pdfLicenseID)), IsPDFItem: true, PDF: null.IntFrom(pdfID)})
	require.NoError(t, err)
	normalItemID, err := database.Db.CreateItem(database.Item{Name: "resend-normal", Description: "resend-normal", Price: 300})
	require.NoError(t, err)

	createOrder := func(email null.String, verified bool, items ...int) int {
		o := database.Order{Vendor: vendor.ID, Verified: verified, CustomerEmail: email}
		for _, item := range items {
			o.Entries = append(o.Entries, database.OrderEntry{Item: item, Quantity: 1, Price: 300, Sender: buyerAccountID, Receiver: vendorAccount.ID, IsSale: true})
		}
		id, err := database.Db.CreateOrder(o)
		require.NoError(t, err)
		return id
	}
	resendURL := func(orderID int) string { return "/api/orders/" + strconv.Itoa(orderID) + "/resend-mail/" }

	email := null.StringFrom("resend@example.com")
	orderID := createOrder(email, true, digitalID, pdfItemID)

	utils.TestRequest(t, r, "POST", resendURL(orderID), nil, 401)

	res := utils.TestRequestWithAuth(t, r, "POST", resendURL(orderID), nil, 200, adminUserToken)
	require.Contains(t, res.Body.String(), `"Sent":2`)
	require.Len(t, sentBodies, 2)
	require.Contains(t, sentBodies[0], "https://paper.test")

	download, err := database.Db.EntClient.PDFDownload.Query().Where(pdfdownload.OrderID(orderID)).Only(context.Background())
	require.NoError(t, err)
	require.True(t, download.EmailSent)
	require.Contains(t, sentBodies[1], "https://frontend.test/pdf/"+download.LinkID)

	// An expired link is reused and valid again
	_, err = download.Update().SetTimestamp(time.Now().AddDate(0, -3, 0)).Save(context.Background())
	require.NoError(t, err)
	utils.TestRequestWithAuth(t, r, "POST", resendURL(orderID), nil, 200, adminUserToken)
	renewed, err := database.Db.EntClient.PDFDownload.Get(context.Background(), download.ID)
	require.NoError(t, err)
	require.Equal(t, download.LinkID, renewed.LinkID)
	require.Less(t, time.Since(renewed.Timestamp), 24*time.Hour)

	sentBodies = nil
	utils.TestRequestWithAuth(t, r, "POST", resendURL(createOrder(null.String{}, true, digitalID)), nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", resendURL(createOrder(email, false, digitalID)), nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", resendURL(createOrder(email, true, normalItemID)), nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", resendURL(999999), nil, 404, adminUserToken)
	require.Empty(t, sentBodies)
}

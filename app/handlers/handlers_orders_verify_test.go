package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/augustin-wien/augustina-backend/wordpress"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestVerifyOrder_EmailSentOnlyOnce ensures that license/pdf emails are only
// sent once even if VerifyOrderAndCreatePayments is called multiple times.
func TestVerifyOrder_EmailSentOnlyOnce(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// fresh DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// create vendor
	vendorLicenseId := "testverifyemail"
	_ = createTestVendor(t, vendorLicenseId)

	// create a PDF resource
	pdf := database.PDF{Path: "testfile.pdf", Timestamp: time.Now()}
	pdfID, err := database.Db.CreatePDF(pdf)
	require.NoError(t, err)

	// create a license item that is a PDF
	licenseItem := database.Item{Name: "license-pdf", Description: "license pdf", Price: 1, IsLicenseItem: true, IsPDFItem: true, PDF: null.IntFrom(pdfID), Type: "license_item"}
	licenseID, err := database.Db.CreateItem(licenseItem)
	require.NoError(t, err)

	// create a normal item that references the license
	item := database.Item{Name: "item-with-license-pdf", Description: "main item", Price: 200, LicenseItem: null.IntFrom(int64(licenseID)), LicenseGroup: null.StringFrom("pdfgroup"), Type: "normal_item"}
	itemID, err := database.Db.CreateItem(item)
	require.NoError(t, err)

	// allow sufficiently large order amounts
	setMaxOrderAmount(t, 5000)

	// Ensure mail templates exist so emails can be built during verification
	err = database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Your license", "Please access your license at {{.URL}}")
	utils.CheckError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("PDFLicenceItemTemplate.html", "Your PDF", "Download your PDF at {{.URL}}")
	utils.CheckError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("welcome", "Welcome", "Welcome to Augustina")
	utils.CheckError(t, err)

	// Stub email building and sending so we can assert welcome emails are sent
	origBuild := database.BuildEmailRequestFromTemplate
	origSend := mailer.Send
	defer func() {
		database.BuildEmailRequestFromTemplate = origBuild
		mailer.Send = origSend
	}()

	var pdfCount, licenseCount, welcomeCount int32
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		subj := "Welcome"
		if name == "digitalLicenceItemTemplate.html" {
			subj = "Your license"
		}
		if name == "PDFLicenceItemTemplate.html" {
			subj = "Your PDF"
		}
		r, err := mailer.NewRequest(to, subj, "body")
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	// Capture sends
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		switch r.Subject() {
		case "Your PDF":
			atomic.AddInt32(&pdfCount, 1)
		case "Your license":
			atomic.AddInt32(&licenseCount, 1)
		case "Welcome":
			atomic.AddInt32(&welcomeCount, 1)
		}
		return true, nil
	}

	customerEmail := "verify_once@example.com"
	// A PDF purchase must not create a Keycloak account; remove leftovers of earlier runs
	_ = keycloak.KeycloakClient.DeleteUser(customerEmail)

	// create order via API (this will prepend license entry internally)
	requestWithEmail := `{
		"entries": [
			{ "item": ` + strconv.Itoa(itemID) + `, "quantity": 1 }
		],
		"vendorLicenseID": "` + vendorLicenseId + `",
		"customerEmail": "` + customerEmail + `"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithEmail, 200)
	var respMap map[string]string
	err = json.Unmarshal(res.Body.Bytes(), &respMap)
	utils.CheckError(t, err)
	url := respMap["SmartCheckoutURL"]
	require.NotEmpty(t, url)
	// In test mode we may get either the real Viva wallet URL or a local test URL.
	// Assert it's a URL that indicates a checkout/success redirect.
	require.True(t, strings.Contains(url, "checkout") || strings.Contains(url, "success"))

	// fetch order: extract OrderCode from SmartCheckoutURL rather than assuming "0"
	orderCodeStr := "0"
	if idx := strings.Index(url, "s="); idx != -1 {
		substr := url[idx+2:]
		if j := strings.Index(substr, "&"); j != -1 {
			substr = substr[:j]
		}
		orderCodeStr = substr
	}
	order, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)

	// call verify twice
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
	require.NoError(t, err)
	// second call should not resend emails or duplicate payments
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
	require.NoError(t, err)

	// check payments: should only create payments once (license + item)
	payments, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false, false, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(payments))

	// check pdf downloads for order: only one download should exist and EmailSent true
	pdfDownloads, err := database.Db.GetPDFDownloadByOrderId(order.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(pdfDownloads))
	require.True(t, pdfDownloads[0].EmailSent)
	require.EqualValues(t, licenseID, pdfDownloads[0].ItemID.ValueOrZero())
	require.EqualValues(t, order.ID, pdfDownloads[0].OrderID.ValueOrZero())

	// Only the PDF download link is mailed: no online paper mail, no welcome
	// mail and no Keycloak account
	time.Sleep(100 * time.Millisecond)
	require.EqualValues(t, 1, atomic.LoadInt32(&pdfCount))
	require.EqualValues(t, 0, atomic.LoadInt32(&licenseCount))
	require.EqualValues(t, 0, atomic.LoadInt32(&welcomeCount))
	_, err = keycloak.KeycloakClient.GetUserByEmail(customerEmail)
	require.Error(t, err)

	// cleanup
	for _, payment := range payments {
		database.Db.DeletePayment(payment.ID)
	}
	for _, entry := range order.Entries {
		database.Db.DeleteOrderEntry(entry.ID)
	}
	database.Db.DeleteOrder(order.ID)
}

// TestVerifyOrder_MultipleDigitalItems_EmailSentOnce ensures that when an
// order contains multiple digital/license items the notification mail is
// only sent once.
func TestVerifyOrder_MultipleDigitalItems_EmailSentOnce(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// fresh DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// create vendor
	vendorLicenseId := "testverifyemail"
	_ = createTestVendor(t, vendorLicenseId)

	// create a PDF resource
	pdf := database.PDF{Path: "testfile.pdf", Timestamp: time.Now()}
	pdfID, err := database.Db.CreatePDF(pdf)
	require.NoError(t, err)

	// create a license item that is a PDF (first license)
	licenseItem := database.Item{Name: "license-pdf", Description: "license pdf", Price: 1, IsLicenseItem: true, IsPDFItem: true, PDF: null.IntFrom(pdfID), Type: "license_item"}
	licenseID, err := database.Db.CreateItem(licenseItem)
	require.NoError(t, err)

	// create a second PDF resource and license (second license)
	pdf2 := database.PDF{Path: "testfile2.pdf", Timestamp: time.Now()}
	pdfID2, err := database.Db.CreatePDF(pdf2)
	require.NoError(t, err)
	licenseItem2 := database.Item{Name: "license-pdf-2", Description: "license pdf 2", Price: 1, IsLicenseItem: true, IsPDFItem: true, PDF: null.IntFrom(pdfID2), Type: "license_item"}
	licenseID2, err := database.Db.CreateItem(licenseItem2)
	require.NoError(t, err)

	// create two normal items that reference their respective licenses (to test multiple entries)
	item1 := database.Item{Name: "item-with-license-pdf-1", Description: "main item 1", Price: 200, LicenseItem: null.IntFrom(int64(licenseID)), LicenseGroup: null.StringFrom("pdfgroup1"), Type: "normal_item"}
	itemID1, err := database.Db.CreateItem(item1)
	require.NoError(t, err)

	item2 := database.Item{Name: "item-with-license-pdf-2", Description: "main item 2", Price: 150, LicenseItem: null.IntFrom(int64(licenseID2)), LicenseGroup: null.StringFrom("pdfgroup2"), Type: "normal_item"}
	itemID2, err := database.Db.CreateItem(item2)
	require.NoError(t, err)

	// allow sufficiently large order amounts
	setMaxOrderAmount(t, 5000)

	// Ensure mail templates exist so emails can be built during verification
	err = database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Your license", "Please access your license at {{.URL}}")
	utils.CheckError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("PDFLicenceItemTemplate.html", "Your PDF", "Download your PDF at {{.URL}}")
	utils.CheckError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("welcome", "Welcome", "Welcome to Augustina")
	utils.CheckError(t, err)

	// Stub email building and sending so we can assert digital emails are sent only once
	origBuild := database.BuildEmailRequestFromTemplate
	origSend := mailer.Send
	defer func() {
		database.BuildEmailRequestFromTemplate = origBuild
		mailer.Send = origSend
	}()

	var pdfCount, licenseCount int32
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		subj := "Digital"
		if name == "digitalLicenceItemTemplate.html" {
			subj = "Your license"
		}
		if name == "PDFLicenceItemTemplate.html" {
			subj = "Your PDF"
		}
		r, err := mailer.NewRequest(to, subj, "body")
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		switch r.Subject() {
		case "Your PDF":
			atomic.AddInt32(&pdfCount, 1)
		case "Your license":
			atomic.AddInt32(&licenseCount, 1)
		}
		return true, nil
	}

	customerEmail := "verify_multi@example.com"

	// create order via API with two different digital items
	// The email should only be sent once per order, not per item
	requestWithEmail := `{
		"entries": [
			{ "item": ` + strconv.Itoa(itemID1) + `, "quantity": 1 },
			{ "item": ` + strconv.Itoa(itemID2) + `, "quantity": 1 }
		],
		"vendorLicenseID": "` + vendorLicenseId + `",
		"customerEmail": "` + customerEmail + `"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithEmail, 200)
	var respMap map[string]string
	err = json.Unmarshal(res.Body.Bytes(), &respMap)
	utils.CheckError(t, err)
	url := respMap["SmartCheckoutURL"]
	require.NotEmpty(t, url)

	// extract order code and get order
	orderCodeStr := "0"
	if idx := strings.Index(url, "s="); idx != -1 {
		substr := url[idx+2:]
		if j := strings.Index(substr, "&"); j != -1 {
			substr = substr[:j]
		}
		orderCodeStr = substr
	}
	order, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)

	// call verify twice
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
	require.NoError(t, err)
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
	require.NoError(t, err)

	// Wait briefly for async sends to execute
	time.Sleep(100 * time.Millisecond)

	// one download mail per PDF, and no online paper mail since both are PDFs
	require.EqualValues(t, 2, atomic.LoadInt32(&pdfCount))
	require.EqualValues(t, 0, atomic.LoadInt32(&licenseCount))

	// cleanup
	payments, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false, false, false)
	require.NoError(t, err)
	for _, payment := range payments {
		database.Db.DeletePayment(payment.ID)
	}
	for _, entry := range order.Entries {
		database.Db.DeleteOrderEntry(entry.ID)
	}
	database.Db.DeleteOrder(order.ID)
}

// TestVerifyPaymentOrder_InviteURL checks that the verify endpoint returns a WordPress
// one-time login URL for new users when WordPressInviteURL is configured.
func TestVerifyPaymentOrder_InviteURL(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	require.NoError(t, err)

	// Fake WordPress invite API
	const fakeInviteURL = "https://wordpress.test/invite/abc123"
	var invitesMu sync.Mutex
	var invites []wordpress.Invite
	lastInvite := func() wordpress.Invite {
		invitesMu.Lock()
		defer invitesMu.Unlock()
		require.NotEmpty(t, invites)
		return invites[len(invites)-1]
	}
	wpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var invite wordpress.Invite
		_ = json.NewDecoder(r.Body).Decode(&invite)
		invitesMu.Lock()
		invites = append(invites, invite)
		invitesMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"magic_link":"` + fakeInviteURL + `","expires_at":"2030-01-01T00:00:00+00:00"}`))
	}))
	defer wpServer.Close()

	// Configure settings to use the fake server
	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	settings.WordPressInviteURL = wpServer.URL
	settings.WordPressInviteAPIKey = "test-api-key"
	settings.WordPressInviteTTL = 604800
	settings.MaxOrderAmount = 10000
	err = database.Db.UpdateSettings(settings)
	require.NoError(t, err)
	defer func() {
		s, _ := database.Db.GetSettings()
		if s != nil {
			s.WordPressInviteURL = ""
			s.WordPressInviteAPIKey = ""
			_ = database.Db.UpdateSettings(s)
		}
	}()

	// Ensure mail templates exist
	err = database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Your license", "Access: {{.URL}}{{if .InviteURL}} Login: {{.InviteURL}}{{end}}")
	require.NoError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("welcome", "Welcome", "Welcome!")
	require.NoError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("abonementConfirmation", "Abo", "Abo confirmed{{if .InviteURL}} Login: {{.InviteURL}}{{end}}")
	require.NoError(t, err)

	// Create vendor and item with license
	vendorLicenseID := "testwpinvite"
	createTestVendor(t, vendorLicenseID)
	itemIDStr, _ := CreateTestItemWithLicense(t)

	customerEmail := "wpinvite_newuser@example.com"
	keycloak.KeycloakClient.DeleteUser(customerEmail)
	defer keycloak.KeycloakClient.DeleteUser(customerEmail)

	// Enable debug payment mode so the order gets a deterministic ?s= checkout URL
	origDebug := config.Config.DEBUG_payments
	config.Config.DEBUG_payments = true
	defer func() { config.Config.DEBUG_payments = origDebug }()

	// Create order
	itemIDInt, _ := strconv.Atoi(itemIDStr)
	reqData := createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemIDInt, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
	}
	res := utils.TestRequest(t, r, "POST", "/api/orders/", reqData, 200)
	var orderResp createOrderResponse
	err = json.Unmarshal(res.Body.Bytes(), &orderResp)
	require.NoError(t, err)

	u, err := url.Parse(orderResp.SmartCheckoutURL)
	require.NoError(t, err)
	orderCode := u.Query().Get("s")
	require.NotEmpty(t, orderCode)

	// Verify in development mode (calls VerifyOrderAndCreatePayments + returns response)
	origDev := config.Config.Development
	config.Config.Development = true
	defer func() { config.Config.Development = origDev }()

	resVerify := utils.TestRequestStr(t, r, "GET", "/api/orders/verify/?s="+orderCode+"&t=0", "", 200)

	var verifyResp VerifyPaymentOrderResponse
	err = json.Unmarshal(resVerify.Body.Bytes(), &verifyResp)
	require.NoError(t, err)
	require.Equal(t, fakeInviteURL, verifyResp.InviteURL, "InviteURL should be set for a new user")
	// The item's license group is the issue number WordPress grants
	require.Equal(t, customerEmail, lastInvite().Email)
	require.Equal(t, "testedition", lastInvite().IssueNumber)
	require.Equal(t, 604800, lastInvite().TTL)

	// A returning customer gets a login link as well, with the stored names
	customer, err := database.Db.GetCustomerByEmail(customerEmail)
	require.NoError(t, err)
	customer.FirstName = "Maria"
	customer.LastName = "Muster"
	_, err = database.Db.UpdateCustomer(customer)
	require.NoError(t, err)
	res = utils.TestRequest(t, r, "POST", "/api/orders/", reqData, 200)
	err = json.Unmarshal(res.Body.Bytes(), &orderResp)
	require.NoError(t, err)
	u, err = url.Parse(orderResp.SmartCheckoutURL)
	require.NoError(t, err)
	resVerify = utils.TestRequestStr(t, r, "GET", "/api/orders/verify/?s="+u.Query().Get("s")+"&t=0", "", 200)
	verifyResp = VerifyPaymentOrderResponse{}
	err = json.Unmarshal(resVerify.Body.Bytes(), &verifyResp)
	require.NoError(t, err)
	require.Equal(t, fakeInviteURL, verifyResp.InviteURL, "InviteURL should be set for a returning customer")
	require.Equal(t, "testedition", lastInvite().IssueNumber)
	require.Equal(t, "Maria", lastInvite().FirstName)
	require.Equal(t, "Muster", lastInvite().LastName)
}

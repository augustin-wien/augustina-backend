package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestAboPurchaseE2E exercises the full abonement purchase path at the HTTP handler
// level: POST /api/orders/ → GET /api/orders/verify/ → asserts that a Customer record,
// an Abonement record, and a confirmation email are all produced.
func TestAboPurchaseE2E(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	skipIfNoCustomerAbonementTables(t)

	const (
		vendorLicenseID = "abo-e2e-vendor-001"
		customerEmail   = "abo-e2e-customer@example.com"
		mockOrderCode   = "7712345678900001"
		mockTxID        = "abo-e2e-tx-001"
	)

	// Vendor
	createTestVendor(t, vendorLicenseID)

	// Abonement item — CreateTestItem doesn't support Type, so create directly
	aboItemID, err := database.Db.CreateItem(database.Item{
		Name:        "Jahresabonnement E2E",
		Description: "Annual subscription for the e2e handler test",
		Price:       2400,
		Type:        "abonement",
	})
	require.NoError(t, err)

	// Seed abonement confirmation template (migration 037 also does this, but be explicit)
	require.NoError(t, database.Db.CreateOrUpdateMailTemplate(
		"abonementConfirmation",
		"Your {{.ItemName}} Subscription",
		"<p>Hello {{.CustomerName}}, subscription active from {{.FromDate}} to {{.ToDate}}.</p>",
	))

	// Intercept the abonement confirmation email to assert it was triggered
	var capturedEmailTo string
	origBuild := database.BuildEmailRequestFromTemplate
	defer func() { database.BuildEmailRequestFromTemplate = origBuild }()
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		if name == "abonementConfirmation" && len(to) > 0 {
			capturedEmailTo = to[0]
		}
		return origBuild(name, to, data)
	}

	// Mock VivaWallet — handles token endpoint and order creation
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/connect/token":
			w.Write([]byte(`{"access_token":"fake","expires_in":3600,"token_type":"Bearer","scope":"api"}`))
		case "/checkout/v2/orders":
			w.Write([]byte(`{"orderCode":` + mockOrderCode + `}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Override VivaWallet config; restore after test
	origAPIURL := config.Config.VivaWalletAPIURL
	origAccountsURL := config.Config.VivaWalletAccountsURL
	origCheckoutURL := config.Config.VivaWalletSmartCheckoutURL
	origDebug := config.Config.DEBUG_payments
	origProd := database.Db.IsProduction
	origDev := config.Config.Development
	origClientID := config.Config.VivaWalletSmartCheckoutClientID
	origClientKey := config.Config.VivaWalletSmartCheckoutClientKey
	origSourceCode := config.Config.VivaWalletSourceCode
	defer func() {
		config.Config.VivaWalletAPIURL = origAPIURL
		config.Config.VivaWalletAccountsURL = origAccountsURL
		config.Config.VivaWalletSmartCheckoutURL = origCheckoutURL
		config.Config.DEBUG_payments = origDebug
		database.Db.IsProduction = origProd
		config.Config.Development = origDev
		config.Config.VivaWalletSmartCheckoutClientID = origClientID
		config.Config.VivaWalletSmartCheckoutClientKey = origClientKey
		config.Config.VivaWalletSourceCode = origSourceCode
	}()

	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.VivaWalletSmartCheckoutURL = mockServer.URL + "/checkout?s="
	config.Config.DEBUG_payments = false
	config.Config.Development = true // skips VivaWallet signature check during verify
	database.Db.IsProduction = true
	if config.Config.VivaWalletSmartCheckoutClientID == "" {
		config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	}
	if config.Config.VivaWalletSmartCheckoutClientKey == "" {
		config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
	}
	if config.Config.VivaWalletSourceCode == "" {
		config.Config.VivaWalletSourceCode = "dummy"
	}

	// Step 1: create order via HTTP — contacts mocked VivaWallet and returns a checkout URL
	reqData := createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: aboItemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
	}
	res := utils.TestRequest(t, r, "POST", "/api/orders/", reqData, 200)

	var orderResp createOrderResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &orderResp))
	require.Contains(t, orderResp.SmartCheckoutURL, mockOrderCode, "checkout URL must embed the order code")

	// Step 2: simulate VivaWallet payment callback — development mode skips signature
	// verification and calls VerifyOrderAndCreatePayments directly
	utils.TestRequest(t, r, "GET", "/api/orders/verify/?s="+mockOrderCode+"&t="+mockTxID, nil, 200)

	// Step 3: customer DB record must exist
	customer, err := database.Db.GetCustomerByEmail(customerEmail)
	require.NoError(t, err)
	require.Equal(t, customerEmail, customer.Email)

	// Step 4: exactly one abonement must be created for that customer
	abonements, err := database.Db.ListAbonementsByCustomer(customer.ID)
	require.NoError(t, err)
	require.Len(t, abonements, 1)
	abo := abonements[0]
	require.Equal(t, aboItemID, abo.ItemID)
	require.Equal(t, "active", abo.Status)
	require.WithinDuration(t, time.Now(), abo.FromDate, 10*time.Second)
	require.WithinDuration(t, time.Now().AddDate(1, 0, 0), abo.ToDate, 10*time.Second)

	// Step 5: abonement confirmation email must have been triggered for the customer
	require.Equal(t, customerEmail, capturedEmailTo)
}

// TestAboPurchaseGrantsCurrentOnlineIssue buys an abonement whose license group
// differs from the published online_issue's one (WordPress only reads numeric
// /customer/newspapers/{issue_number} groups) and asserts the buyer ends up in
// the current issue's Keycloak group.
func TestAboPurchaseGrantsCurrentOnlineIssue(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	skipIfNoCustomerAbonementTables(t)

	const (
		vendorLicenseID = "abo-issue-vendor-001"
		customerEmail   = "abo-issue-customer@example.com"
		mockOrderCode   = "7712345678900002"
		mockTxID        = "abo-issue-tx-001"
		aboGroup        = "abo_issue_test"
		oldIssueGroup   = "9627"
		currentGroup    = "9628"
		unpublishedNext = "9629"
	)

	keycloak.KeycloakClient.DeleteUser(customerEmail)
	defer keycloak.KeycloakClient.DeleteUser(customerEmail)

	createTestVendor(t, vendorLicenseID)

	// Items with a license start disabled; enabling publishes them
	createIssue := func(number string, publish bool) {
		id, _, err := database.Db.CreateOnlineIssueWithLicense(database.Item{
			Name: "Online Ausgabe " + number, Description: "Online issue " + number, Price: 300,
			LicenseGroup: null.StringFrom(number),
		}, 50)
		require.NoError(t, err)
		if publish {
			enableTestItem(t, id)
		}
	}
	createIssue(oldIssueGroup, true)
	createIssue(currentGroup, true)
	createIssue(unpublishedNext, false)
	aboItemID, _, err := database.Db.CreateAbonementItemWithLicense(database.Item{
		Name: "Jahresabo Ausgabentest", Description: "Annual subscription", Price: 2400,
		LicenseGroup: null.StringFrom(aboGroup),
	}, 50)
	require.NoError(t, err)
	enableTestItem(t, aboItemID)

	origBuild := database.BuildEmailRequestFromTemplate
	defer func() { database.BuildEmailRequestFromTemplate = origBuild }()
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		return nil, nil
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/connect/token":
			w.Write([]byte(`{"access_token":"fake","expires_in":3600,"token_type":"Bearer","scope":"api"}`))
		case "/checkout/v2/orders":
			w.Write([]byte(`{"orderCode":` + mockOrderCode + `}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	origAPIURL := config.Config.VivaWalletAPIURL
	origAccountsURL := config.Config.VivaWalletAccountsURL
	origCheckoutURL := config.Config.VivaWalletSmartCheckoutURL
	origDebug := config.Config.DEBUG_payments
	origProd := database.Db.IsProduction
	origDev := config.Config.Development
	origClientID := config.Config.VivaWalletSmartCheckoutClientID
	origClientKey := config.Config.VivaWalletSmartCheckoutClientKey
	origSourceCode := config.Config.VivaWalletSourceCode
	defer func() {
		config.Config.VivaWalletAPIURL = origAPIURL
		config.Config.VivaWalletAccountsURL = origAccountsURL
		config.Config.VivaWalletSmartCheckoutURL = origCheckoutURL
		config.Config.DEBUG_payments = origDebug
		database.Db.IsProduction = origProd
		config.Config.Development = origDev
		config.Config.VivaWalletSmartCheckoutClientID = origClientID
		config.Config.VivaWalletSmartCheckoutClientKey = origClientKey
		config.Config.VivaWalletSourceCode = origSourceCode
	}()
	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.VivaWalletSmartCheckoutURL = mockServer.URL + "/checkout?s="
	config.Config.DEBUG_payments = false
	config.Config.Development = true
	database.Db.IsProduction = true
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
	config.Config.VivaWalletSourceCode = "dummy"

	reqData := createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: aboItemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
	}
	utils.TestRequest(t, r, "POST", "/api/orders/", reqData, 200)
	utils.TestRequest(t, r, "GET", "/api/orders/verify/?s="+mockOrderCode+"&t="+mockTxID, nil, 200)

	customer, err := database.Db.GetCustomerByEmail(customerEmail)
	require.NoError(t, err)
	abonements, err := database.Db.ListAbonementsByCustomer(customer.ID)
	require.NoError(t, err)
	require.Len(t, abonements, 1)
	require.Equal(t, "active", abonements[0].Status)

	require.Contains(t, customer.LicenseGroups, currentGroup, "customer record must hold the current issue's license group")
	require.NotContains(t, customer.LicenseGroups, oldIssueGroup)
	require.NotContains(t, customer.LicenseGroups, unpublishedNext)

	kcUser, err := keycloak.KeycloakClient.GetUserByEmail(customerEmail)
	require.NoError(t, err)
	groups, err := keycloak.KeycloakClient.GetUserGroups(*kcUser.ID)
	require.NoError(t, err)
	var paths []string
	for _, g := range groups {
		if g.Path != nil {
			paths = append(paths, *g.Path)
		}
	}
	newspapers := "/" + keycloak.KeycloakClient.CustomerGroup + "/" + keycloak.KeycloakClient.NewspaperGroup + "/"
	require.Contains(t, paths, newspapers+currentGroup, "buyer must be in the current issue's Keycloak group")
	require.NotContains(t, paths, newspapers+unpublishedNext)
}

// enableTestItem publishes an item that was created disabled
func enableTestItem(t *testing.T, id int) {
	item, err := database.Db.GetItemIncludingDisabled(id)
	require.NoError(t, err)
	item.Disabled = false
	require.NoError(t, database.Db.UpdateItem(id, item))
}

func TestCheckWordPressIssue(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":"shop_unauthorized","message":"Ungültiger oder fehlender API-Schlüssel."}`))
			return
		}
		switch r.URL.Path {
		case "/wp-json/augustin/v1/shop/issue/651":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"651","title":"Augustin Nr. 651","status":"publish","published":true,"is_demo":false}`))
		case "/wp-json/augustin/v1/shop/issue/650":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"650","title":"Augustin Nr. 650","status":"draft","published":false,"is_demo":false}`))
		case "/wp-json/augustin/v1/shop/issue/652":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"652","title":"Augustin Nr. 652","status":"publish","published":true,"is_demo":true}`))
		default:
			_, _ = w.Write([]byte(`{"exists":false}`))
		}
	}))
	defer wp.Close()

	setInvite := func(url, key string) {
		settings, err := database.Db.GetSettings()
		require.NoError(t, err)
		settings.WordPressInviteURL, settings.WordPressInviteAPIKey = url, key
		require.NoError(t, database.Db.UpdateSettings(settings))
	}
	defer setInvite("", "")

	check := func(licenseGroup string) WordPressIssueCheck {
		res := utils.TestRequestWithAuth(t, r, "GET", "/api/items/wordpress-issue/?licenseGroup="+licenseGroup, nil, 200, adminUserToken)
		var out WordPressIssueCheck
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
		return out
	}

	setInvite("", "")
	require.False(t, check("651").Configured)

	setInvite(wp.URL+"/wp-json/augustin/v1/shop/create-invite", "key")
	res := check("651")
	require.True(t, res.Configured)
	require.True(t, res.OK, res.Message)
	require.Equal(t, "Augustin Nr. 651", res.Issue.Title)

	res = check("650")
	require.False(t, res.OK)
	require.Contains(t, res.Message, "nicht veröffentlicht")
	res = check("652")
	require.False(t, res.OK)
	require.Contains(t, res.Message, "Demo")
	res = check("digital_edition")
	require.False(t, res.OK)
	require.Contains(t, res.Message, "keine Ausgabe")
	require.False(t, check("a%20b").OK)

	setInvite(wp.URL+"/wp-json/augustin/v1/shop/create-invite", "wrong")
	res = check("651")
	require.False(t, res.OK)
	require.Contains(t, res.Message, "API-Schlüssel")

	// Only admins
	utils.TestRequest(t, r, "GET", "/api/items/wordpress-issue/?licenseGroup=651", nil, 401)
}

// TestNotifyAbonementsChecksWordPress checks that an online issue WordPress has
// not published is neither granted nor mailed to the abonnents
func TestNotifyAbonementsChecksWordPress(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/augustin/v1/shop/issue/9650":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"9650","title":"Augustin Nr. 9650","status":"draft","published":false,"is_demo":false}`))
		case "/wp-json/augustin/v1/shop/issue/9651":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"9651","title":"Augustin Nr. 9651","status":"publish","published":true,"is_demo":false}`))
		default:
			_, _ = w.Write([]byte(`{"exists":false}`))
		}
	}))
	defer wp.Close()
	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	settings.WordPressInviteURL = wp.URL + "/wp-json/augustin/v1/shop/create-invite"
	settings.WordPressInviteAPIKey = "key"
	require.NoError(t, database.Db.UpdateSettings(settings))
	defer func() {
		s, _ := database.Db.GetSettings()
		if s != nil {
			s.WordPressInviteURL, s.WordPressInviteAPIKey = "", ""
			_ = database.Db.UpdateSettings(s)
		}
	}()

	origBuild := database.BuildEmailRequestFromTemplate
	origSend := mailer.Send
	defer func() {
		database.BuildEmailRequestFromTemplate = origBuild
		mailer.Send = origSend
	}()
	database.BuildEmailRequestFromTemplate = func(string, []string, interface{}) (*mailer.EmailRequest, error) {
		return mailer.NewRequest([]string{"x@example.com"}, "s", "b")
	}
	mailer.Send = func(*mailer.EmailRequest) (bool, error) { return true, nil }

	newspapers := "/" + keycloak.KeycloakClient.CustomerGroup + "/" + keycloak.KeycloakClient.NewspaperGroup + "/"
	for _, n := range []string{"9650", "9651"} {
		_ = keycloak.KeycloakClient.DeleteSubGroupByPath(newspapers + n)
		defer func(n string) { _ = keycloak.KeycloakClient.DeleteSubGroupByPath(newspapers + n) }(n)
	}

	notify := func(number string, status int) string {
		id, _, err := database.Db.CreateOnlineIssueWithLicense(database.Item{
			Name: "Online Ausgabe " + number, Description: "Online issue " + number, Price: 300,
			LicenseGroup: null.StringFrom(number),
		}, 50)
		require.NoError(t, err)
		enableTestItem(t, id)
		res := utils.TestRequestWithAuth(t, r, "POST", "/api/items/"+strconv.Itoa(id)+"/notify-abonements/", nil, status, adminUserToken)
		return res.Body.String()
	}

	body := notify("9650", 400)
	require.Contains(t, body, "nicht veröffentlicht")
	_, err = keycloak.KeycloakClient.GetGroupByPath(newspapers + "9650")
	require.Error(t, err, "a refused issue gets no keycloak group")

	notify("9651", 200)
	_, err = keycloak.KeycloakClient.GetGroupByPath(newspapers + "9651")
	require.NoError(t, err)
}

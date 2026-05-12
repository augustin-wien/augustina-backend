package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
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

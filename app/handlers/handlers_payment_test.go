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
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestCreatePaymentOrder_LargeOrderCode(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testlargeordercode"
	createTestVendor(t, vendorLicenseID)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item", 100, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Mock VivaWallet Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		if r.URL.Path == "/checkout/v2/orders" {
			w.Header().Set("Content-Type", "application/json")
			// Return the large order code
			w.Write([]byte(`{"orderCode": 9995519122790771}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	// Save original config
	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalSmartCheckoutURL := config.Config.VivaWalletSmartCheckoutURL
	originalDebugPayments := config.Config.DEBUG_payments
	originalIsProduction := database.Db.IsProduction
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey
	originalSourceCode := config.Config.VivaWalletSourceCode

	// Restore config after test
	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.VivaWalletSmartCheckoutURL = originalSmartCheckoutURL
		config.Config.DEBUG_payments = originalDebugPayments
		database.Db.IsProduction = originalIsProduction
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
		config.Config.VivaWalletSourceCode = originalSourceCode
	}()

	// Set config to mock server
	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.VivaWalletSmartCheckoutURL = mockServer.URL + "?s="
	config.Config.DEBUG_payments = false
	database.Db.IsProduction = true

	// Ensure client credentials are set
	if config.Config.VivaWalletSmartCheckoutClientID == "" {
		config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	}
	if config.Config.VivaWalletSmartCheckoutClientKey == "" {
		config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
	}
	if config.Config.VivaWalletSourceCode == "" {
		config.Config.VivaWalletSourceCode = "dummy"
	}

	// Create Order Request
	reqData := createOrderRequest{
		Entries: []createOrderRequestEntry{
			{Item: itemID, Quantity: 1},
		},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom("test@example.com"),
	}

	// Execute Request
	res := utils.TestRequest(t, r, "POST", "/api/orders/", reqData, 200)

	// Parse Response
	var resp createOrderResponse
	err = json.Unmarshal(res.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Check if CheckoutURL contains the correct order code
	expectedOrderCode := "9995519122790771"
	require.Contains(t, resp.SmartCheckoutURL, expectedOrderCode)

	// Also verify it didn't become ...772
	require.NotContains(t, resp.SmartCheckoutURL, "9995519122790772")
}

func TestVerifyPaymentOrder_SavesTransactionID(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testverifytransaction"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Create Order
	orderCode := "1234567890"
	order := database.Order{
		OrderCode:     null.StringFrom(orderCode),
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		CustomerEmail: null.StringFrom("test@example.com"),
	}
	orderID, err := database.Db.CreateOrder(order)
	require.NoError(t, err)
	require.NotZero(t, orderID)

	// Mock VivaWallet Server (needed if verification logic is triggered)
	// However, in development mode (which tests usually run in), verification might be skipped or mocked differently.
	// Let's check handlers.go:
	// if database.Db.IsProduction && !config.Config.Development && !config.Config.DEBUG_payments { ... VerifyTransactionID ... }
	// if config.Config.Development { ... VerifyOrderAndCreatePayments ... }

	// We want to test that TransactionID is saved regardless of verification outcome,
	// but the handler returns early if verification fails in production mode.
	// In development mode, it proceeds to VerifyOrderAndCreatePayments.

	// Let's simulate development mode to avoid external calls and focus on saving TransactionID.
	originalDevelopment := config.Config.Development
	config.Config.Development = true
	defer func() { config.Config.Development = originalDevelopment }()

	transactionID := "test-transaction-id-123"
	url := "/api/orders/verify/?s=" + orderCode + "&t=" + transactionID

	// Execute Request
	utils.TestRequest(t, r, "GET", url, nil, 200)

	// Verify TransactionID is saved in database
	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCode)
	require.NoError(t, err)
	require.Equal(t, transactionID, updatedOrder.TransactionID)
}

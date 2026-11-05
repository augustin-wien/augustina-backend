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

func TestAdminAddTransactionIDToOrder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testadminaddtx"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Create Order
	orderCode := "9876543210"
	order := database.Order{
		OrderCode:     null.StringFrom(orderCode),
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		CustomerEmail: null.StringFrom("test@example.com"),
	}
	orderID, err := database.Db.CreateOrder(order)
	require.NoError(t, err)
	require.NotZero(t, orderID)

	// Prepare Request
	transactionID := "manual-tx-id-999"
	reqData := addTransactionIDRequest{
		TransactionID: transactionID,
	}

	// Execute Request
	url := "/api/orders/unverified/code/" + orderCode + "/transactionID/"
	utils.TestRequestWithAuth(t, r, "POST", url, reqData, 200, adminUserToken)

	// Verify TransactionID is saved in database
	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCode)
	require.NoError(t, err)
	require.Equal(t, transactionID, updatedOrder.TransactionID)
}

// mockVivaWalletTransaction runs the order verification in production mode against a test server
// that knows a single successful transaction, and restores the configuration after the test
func mockVivaWalletTransaction(t *testing.T, transactionID, orderCode string) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/connect/token":
			_, _ = w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
		case "/checkout/v2/transactions/" + transactionID:
			_, _ = w.Write([]byte(`{"orderCode": ` + orderCode + `, "statusId": "F", "amount": 6}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey
	originalDebugPayments := config.Config.DEBUG_payments
	originalDevelopment := config.Config.Development
	originalIsProduction := database.Db.IsProduction
	t.Cleanup(func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
		config.Config.DEBUG_payments = originalDebugPayments
		config.Config.Development = originalDevelopment
		database.Db.IsProduction = originalIsProduction
	})

	config.Config.VivaWalletAPIURL = server.URL
	config.Config.VivaWalletAccountsURL = server.URL
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
	config.Config.DEBUG_payments = false
	config.Config.Development = false
	database.Db.IsProduction = true
}

// The verify endpoint is public and answers with the order's download links and login link, so a
// paid transaction must only open the order it was made for
func TestVerifyPaymentOrder_RejectsTransactionOfAnotherOrder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorLicenseID := "testverifyforeigntx"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	const victimOrderCode, attackerOrderCode = "1111222233334444", "5555666677778888"
	for orderCode, transactionID := range map[string]string{victimOrderCode: "victim-tx", attackerOrderCode: "attacker-tx"} {
		_, err := database.Db.CreateOrder(database.Order{
			OrderCode:     null.StringFrom(orderCode),
			TransactionID: transactionID,
			Verified:      true,
			Vendor:        vendor.ID,
			CustomerEmail: null.StringFrom(orderCode + "@example.com"),
		})
		require.NoError(t, err)
	}

	mockVivaWalletTransaction(t, "attacker-tx", attackerOrderCode)

	utils.TestRequest(t, r, "GET", "/api/orders/verify/?s="+victimOrderCode+"&t=attacker-tx", nil, http.StatusBadRequest)

	victimOrder, err := database.Db.GetOrderByOrderCode(victimOrderCode)
	require.NoError(t, err)
	require.Equal(t, "victim-tx", victimOrder.TransactionID)

	// The transaction still opens its own order
	utils.TestRequest(t, r, "GET", "/api/orders/verify/?s="+attackerOrderCode+"&t=attacker-tx", nil, http.StatusOK)
}

func TestVerifyPaymentOrder_KeepsConfirmedTransactionWhileWebhookIsPending(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorLicenseID := "testverifypendingtx"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	const orderCode = "9999888877776666"
	_, err = database.Db.CreateOrder(database.Order{OrderCode: null.StringFrom(orderCode), Vendor: vendor.ID})
	require.NoError(t, err)

	mockVivaWalletTransaction(t, "pending-tx", orderCode)

	// Paid but not yet verified by the webhook, which the frontend polls for
	utils.TestRequest(t, r, "GET", "/api/orders/verify/?s="+orderCode+"&t=pending-tx", nil, http.StatusBadRequest)

	order, err := database.Db.GetOrderByOrderCode(orderCode)
	require.NoError(t, err)
	require.False(t, order.Verified)
	require.Equal(t, "pending-tx", order.TransactionID)
}

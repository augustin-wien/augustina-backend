package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/paymentprovider"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestVivaWalletWebhookSuccess_LargeOrderCode(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testwebhooklargeordercode"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Get Accounts
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item Webhook", 100, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Order in DB
	orderCodeStr := "9555233246002521"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "test-transaction-id"

	order := database.Order{
		OrderCode:     null.StringFrom(orderCodeStr),
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		CustomerEmail: null.StringFrom("webhook@example.com"),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    100,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := database.Db.CreateOrder(order)
	require.NoError(t, err)
	order.ID = orderID

	// Mock VivaWallet Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		if r.URL.Path == "/checkout/v2/transactions/"+transactionID {
			w.Header().Set("Content-Type", "application/json")
			// Return verification response
			resp := paymentprovider.TransactionVerificationResponse{
				Email:             "webhook@example.com",
				Amount:            1.00, // 100 cents = 1.00 EUR
				OrderCode:         paymentprovider.VivaOrderCode(orderCodeInt),
				StatusID:          "F",
				TransactionTypeID: 0,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	// Save original config
	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalTransactionCostsName := config.Config.TransactionCostsName
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey

	// Restore config after test
	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.TransactionCostsName = originalTransactionCostsName
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
	}()

	// Set config to mock server
	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.TransactionCostsName = "Transaction Costs" // Needs to be set for verification logic
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"

	// Create Transaction Costs Item if not exists (needed for verification logic)
	CreateTestItem(t, config.Config.TransactionCostsName, 1, "", "")

	// Create Webhook Request
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         paymentprovider.VivaOrderCode(orderCodeInt),
			TransactionID:     transactionID,
			Amount:            1.00,
			StatusID:          "F",
			TransactionTypeID: 0,
		},
	}
	payloadBytes, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler
	VivaWalletWebhookSuccess(rec, req)

	// Assert Response
	require.Equal(t, http.StatusOK, rec.Code)

	// Assert Order Verified
	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.True(t, updatedOrder.Verified)
	require.Equal(t, transactionID, updatedOrder.TransactionID)
}

func TestVivaWalletWebhookSuccess_ValidPayment(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testwebhooksuccess"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Get Accounts
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item", 5000, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Order in DB
	orderCodeStr := "1234567890123456"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "test-txn-id-12345"

	order := database.Order{
		OrderCode:     null.StringFrom(orderCodeStr),
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		CustomerEmail: null.StringFrom("test@example.com"),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    5000,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := database.Db.CreateOrder(order)
	require.NoError(t, err)
	order.ID = orderID

	// Mock VivaWallet Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		if r.URL.Path == "/checkout/v2/transactions/"+transactionID {
			w.Header().Set("Content-Type", "application/json")
			resp := paymentprovider.TransactionVerificationResponse{
				Email:             "test@example.com",
				Amount:            50.00,
				OrderCode:         paymentprovider.VivaOrderCode(orderCodeInt),
				StatusID:          "F",
				TransactionTypeID: 0,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	// Save and set config
	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalTransactionCostsName := config.Config.TransactionCostsName
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey

	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.TransactionCostsName = originalTransactionCostsName
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
	}()

	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.TransactionCostsName = "Transaction Costs"
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"

	CreateTestItem(t, config.Config.TransactionCostsName, 1, "", "")

	// Create Webhook Request
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         paymentprovider.VivaOrderCode(orderCodeInt),
			TransactionID:     transactionID,
			Amount:            50.00,
			StatusID:          "F",
			TransactionTypeID: 0,
			Email:             "test@example.com",
		},
	}
	payloadBytes, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler
	VivaWalletWebhookSuccess(rec, req)

	// Assert Response
	require.Equal(t, http.StatusOK, rec.Code)

	var response webhookResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "OK", response.Status)

	// Assert Order Verified
	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.True(t, updatedOrder.Verified)
	require.Equal(t, transactionID, updatedOrder.TransactionID)
}

func TestVivaWalletWebhookSuccess_InvalidJSON(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Create Request with invalid JSON
	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler
	VivaWalletWebhookSuccess(rec, req)

	// Assert Response is BadRequest
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVivaWalletWebhookSuccess_EmptyBody(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	originalSourceCode := config.Config.VivaWalletSourceCode
	defer func() { config.Config.VivaWalletSourceCode = originalSourceCode }()
	config.Config.VivaWalletSourceCode = "1234"

	// Create Request with empty body
	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler
	VivaWalletWebhookSuccess(rec, req)

	// An empty JSON object parses, but carries neither a known order nor our source code,
	// so it is acknowledged like a payment from another sales channel
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestVivaWalletWebhookSuccess_NonexistentOrder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	originalSourceCode := config.Config.VivaWalletSourceCode
	defer func() { config.Config.VivaWalletSourceCode = originalSourceCode }()
	config.Config.VivaWalletSourceCode = "1234"

	// Create Webhook Request with nonexistent order code
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         9999999999999999,
			TransactionID:     "nonexistent-txn-id",
			Amount:            50.00,
			StatusID:          "F",
			TransactionTypeID: 0,
			SourceCode:        "1234",
		},
	}
	payloadBytes, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler
	VivaWalletWebhookSuccess(rec, req)

	// HandlePaymentSuccessfulResponse fails because the order does not exist;
	// the handler returns 500 so VivaWallet retries the delivery
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// A payment from another sales channel on the same VivaWallet account, such as the donation
// terminal, has no order here and must be acknowledged instead of retried forever
func TestVivaWalletWebhookSuccess_ForeignTransactionIsAcknowledged(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	originalSourceCode := config.Config.VivaWalletSourceCode
	defer func() { config.Config.VivaWalletSourceCode = originalSourceCode }()
	config.Config.VivaWalletSourceCode = "1234"

	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:     6289177113801428,
			TransactionID: "terminal-txn-id",
			Amount:        10.00,
			StatusID:      "F",
			SourceCode:    "Default",
			TerminalID:    16000001,
		},
	}
	payloadBytes, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	VivaWalletWebhookSuccess(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

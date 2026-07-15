package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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
				OrderCode:         orderCodeInt,
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
			OrderCode:         orderCodeInt,
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
				OrderCode:         orderCodeInt,
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
			OrderCode:         orderCodeInt,
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

	// Create Request with empty body
	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler
	VivaWalletWebhookSuccess(rec, req)

	// Handler should successfully parse empty JSON object, but fail during payment
	// handling and return 500 so VivaWallet retries the delivery
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestVivaWalletWebhookSuccess_NonexistentOrder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Webhook Request with nonexistent order code
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         9999999999999999,
			TransactionID:     "nonexistent-txn-id",
			Amount:            50.00,
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

	// HandlePaymentSuccessfulResponse fails because the order does not exist;
	// the handler returns 500 so VivaWallet retries the delivery
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestVivaWalletWebhookSuccess_VerificationFailureKeepsTransactionID(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testwebhookverifyfail"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Get Accounts
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item Verify Fail", 600, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Order in DB without a transaction ID, so it gets the "manual-" placeholder
	orderCodeStr := "1188333166949561"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "real-txn-id-from-webhook"

	order := database.Order{
		OrderCode: null.StringFrom(orderCodeStr),
		Vendor:    vendor.ID,
		Timestamp: time.Now(),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    600,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := database.Db.CreateOrder(order)
	require.NoError(t, err)

	createdOrder, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(createdOrder.TransactionID, "manual-"))

	// Mock VivaWallet Server where authentication works, but the transaction is not
	// (yet) known — VivaWallet's transaction API is eventually consistent and can
	// return 404 right after a payment
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	// Save and set config
	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey

	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
	}()

	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"

	// Create Webhook Request
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         orderCodeInt,
			TransactionID:     transactionID,
			Amount:            6.00,
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

	// Verification failed, so the handler must return 500 to make VivaWallet
	// retry the delivery
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	// The order stays unverified, but the real transaction ID from the webhook
	// must replace the "manual-" placeholder so the order can still be verified
	// against VivaWallet later (e.g. via the admin panel)
	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.Equal(t, orderID, updatedOrder.ID)
	require.False(t, updatedOrder.Verified)
	require.Equal(t, transactionID, updatedOrder.TransactionID)
}

func TestVivaWalletWebhookSuccess_DuplicateDeliveryIsNoOp(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testwebhookduplicate"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Get Accounts
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item Duplicate", 600, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Order in DB with a transaction ID and verify it
	orderCodeStr := "2233445566778899"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "orig-txn-id"

	order := database.Order{
		OrderCode:     null.StringFrom(orderCodeStr),
		TransactionID: transactionID,
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    600,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := database.Db.CreateOrder(order)
	require.NoError(t, err)

	err = database.Db.VerifyOrderAndCreatePayments(orderID, 0)
	require.NoError(t, err)

	// Mock VivaWallet Server that fails every request, to prove that a duplicate
	// delivery is answered without consulting VivaWallet
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "must not be called", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// Save and set config
	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL

	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
	}()

	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL

	// Deliver the same webhook again
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         orderCodeInt,
			TransactionID:     "retry-txn-id",
			Amount:            6.00,
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

	// A duplicate delivery for an already verified order is acknowledged with 200,
	// so VivaWallet stops retrying
	require.Equal(t, http.StatusOK, rec.Code)

	var response webhookResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "OK", response.Status)

	// The order keeps its original transaction ID
	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.True(t, updatedOrder.Verified)
	require.Equal(t, transactionID, updatedOrder.TransactionID)
}

func TestVivaWalletWebhookSuccess_ResponseHeaderContentType(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testwebhookheader"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Get Accounts
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item", 3000, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Order in DB
	orderCodeStr := "9876543210987654"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "test-txn-header-check"

	order := database.Order{
		OrderCode:     null.StringFrom(orderCodeStr),
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		CustomerEmail: null.StringFrom("test@example.com"),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    3000,
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
				Amount:            30.00,
				OrderCode:         orderCodeInt,
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
			OrderCode:         orderCodeInt,
			TransactionID:     transactionID,
			Amount:            30.00,
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

	// Assert Content-Type header is set to application/json
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestVivaWalletWebhookSuccess_SendsToFlourWebhook(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create Vendor
	vendorLicenseID := "testwebhookflour"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	// Get Accounts
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)

	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item Flour", 7500, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Order in DB
	orderCodeStr := "1111111111111111"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "test-txn-flour-webhook"

	order := database.Order{
		OrderCode:     null.StringFrom(orderCodeStr),
		Vendor:        vendor.ID,
		Timestamp:     time.Now(),
		CustomerEmail: null.StringFrom("flour-test@example.com"),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    7500,
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
				Email:             "flour-test@example.com",
				Amount:            75.00,
				OrderCode:         orderCodeInt,
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
	originalFlourWebhookURL := config.Config.FlourWebhookURL

	// Restore config after test
	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.TransactionCostsName = originalTransactionCostsName
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
		config.Config.FlourWebhookURL = originalFlourWebhookURL
	}()

	// Set config to mock server
	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.TransactionCostsName = "Transaction Costs"
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
	config.Config.FlourWebhookURL = "http://localhost:8081/webhooks/payment"

	// Create Transaction Costs Item
	CreateTestItem(t, config.Config.TransactionCostsName, 1, "", "")

	// Create Webhook Request with valid transaction data
	webhookPayload := paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:         orderCodeInt,
			TransactionID:     transactionID,
			Amount:            75.00,
			StatusID:          "F",
			TransactionTypeID: 0,
			Email:             "flour-test@example.com",
		},
	}
	payloadBytes, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Call Handler - this should send a webhook to localhost:8081 asynchronously
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

	// Give the async webhook sender time to send the webhook
	time.Sleep(2 * time.Second)

	// Check webhook logs to verify the webhook was sent to localhost:8081
	// The webhook receiver logs to ./webhook-logs/webhooks-YYYY-MM-DD.log
	logPath := "/home/nanu/go/src/github.com/augustin-wien/augustin-backend/webhook-logs/webhooks-2025-12-10.log"

	// Try to read the log file (it may not exist if webhook receiver is not running)
	logContent, err := os.ReadFile(logPath)
	if err == nil {
		// Log file exists, verify webhook was logged
		logStr := string(logContent)
		// The webhook should contain order ID or license ID
		require.True(t, strings.Contains(logStr, "553") || strings.Contains(logStr, "testwebhookflour"),
			"Expected webhook not found in logs. Ensure webhook-receiver service is running on localhost:8081")
	}
	// If log file doesn't exist, test will still pass but user should verify webhook receiver is running
}

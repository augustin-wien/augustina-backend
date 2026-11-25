package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
			OrderCode:         paymentprovider.VivaOrderCode(orderCodeInt),
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
			OrderCode:         paymentprovider.VivaOrderCode(orderCodeInt),
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

// Anyone can post to the webhook, so before VivaWallet confirms a transaction its id must not
// replace a transaction id that is already stored for the order
func TestVivaWalletWebhookSuccess_UnconfirmedTransactionKeepsStoredOne(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorLicenseID := "testwebhookkeepstx"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)

	orderCodeStr := "3344556677889900"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	_, err = database.Db.CreateOrder(database.Order{
		OrderCode:     null.StringFrom(orderCodeStr),
		TransactionID: "confirmed-tx",
		Vendor:        vendor.ID,
	})
	require.NoError(t, err)

	// VivaWallet does not know the posted transaction
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

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

	payloadBytes, err := json.Marshal(paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:     paymentprovider.VivaOrderCode(orderCodeInt),
			TransactionID: "forged-tx",
			Amount:        6.00,
			StatusID:      "F",
		},
	})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	VivaWalletWebhookSuccess(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	order, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.False(t, order.Verified)
	require.Equal(t, "confirmed-tx", order.TransactionID)
}

func TestVivaWalletWebhookSuccess_LicenseItemArchivedAfterOrder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorLicenseID := "testwebhookarchivedlicense"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	itemID, _ := strconv.Atoi(CreateTestItem(t, "Issue", 5000, "", ""))
	licenseItemID, _ := strconv.Atoi(CreateTestItem(t, "Issue License", 2000, "", ""))
	require.NoError(t, database.Db.EntClient.Item.UpdateOneID(licenseItemID).SetIsLicenseItem(true).Exec(context.Background()))

	orderCodeStr := "6543210987654321"
	orderCodeInt, _ := strconv.ParseInt(orderCodeStr, 10, 64)
	transactionID := "test-txn-archived-license"
	_, err = database.Db.CreateOrder(database.Order{
		OrderCode: null.StringFrom(orderCodeStr),
		Vendor:    vendor.ID,
		Timestamp: time.Now(),
		Entries: []database.OrderEntry{
			{Item: itemID, Quantity: 1, Price: 5000, Sender: buyerAccountID, Receiver: vendorAccount.ID, IsSale: true},
			{Item: licenseItemID, Quantity: 1, Price: 2000, Sender: buyerAccountID, Receiver: vendorAccount.ID, IsSale: true},
		},
	})
	require.NoError(t, err)

	// The license item is taken out of the shop while the customer is paying
	require.NoError(t, database.Db.EntClient.Item.UpdateOneID(licenseItemID).SetArchived(true).SetDisabled(true).Exec(context.Background()))

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		if r.URL.Path == "/checkout/v2/transactions/"+transactionID {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(paymentprovider.TransactionVerificationResponse{
				Amount:    50.00,
				OrderCode: paymentprovider.VivaOrderCode(orderCodeInt),
				StatusID:  "F",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

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

	payload, _ := json.Marshal(paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:     paymentprovider.VivaOrderCode(orderCodeInt),
			TransactionID: transactionID,
			Amount:        50.00,
			StatusID:      "F",
		},
	})
	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	VivaWalletWebhookSuccess(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	updatedOrder, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)
	require.True(t, updatedOrder.Verified, "the archived license item is still left out of the sum")
}

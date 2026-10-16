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

const priceTestTransactionID = "price-test-transaction-id"

// setupPriceWebhookTest points the VivaWallet config at a mock server that returns the given
// merchant reference for priceTestTransactionID, and restores the config when the test ends
func setupPriceWebhookTest(t *testing.T, orderCode int64, merchantTrns string) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		if r.URL.Path == "/checkout/v2/transactions/"+priceTestTransactionID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(paymentprovider.TransactionVerificationResponse{
				Amount:       10.00,
				OrderCode:    paymentprovider.VivaOrderCode(orderCode),
				StatusID:     "F",
				MerchantTrns: merchantTrns,
			})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(mockServer.Close)

	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalTransactionCostsName := config.Config.TransactionCostsName
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey
	t.Cleanup(func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.TransactionCostsName = originalTransactionCostsName
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
	})

	config.Config.VivaWalletAPIURL = mockServer.URL
	config.Config.VivaWalletAccountsURL = mockServer.URL
	config.Config.TransactionCostsName = "Transaction Costs"
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
}

func postPriceWebhook(orderCode int64, totalCommission float64) *httptest.ResponseRecorder {
	payload := paymentprovider.TransactionPriceRequest{
		EventData: paymentprovider.PriceEventData{
			OrderCode:       paymentprovider.VivaOrderCode(orderCode),
			TransactionID:   priceTestTransactionID,
			TotalCommission: totalCommission,
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/webhooks/vivawallet/price/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	VivaWalletWebhookPrice(rec, req)
	return rec
}

func TestVivaWalletWebhookPrice_ForeignTransactionIsAcknowledged(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	setupPriceWebhookTest(t, 6289177113801428, "Spende")

	rec := postPriceWebhook(6289177113801428, 0.15)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestVivaWalletWebhookPrice_MissingOwnOrderIsRetried(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	setupPriceWebhookTest(t, 6289177113801428, "Ein gutes Leben für alle!")

	rec := postPriceWebhook(6289177113801428, 0.15)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestVivaWalletWebhookPrice_DuplicateDeliveryBooksCostsOnce(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	vendorLicenseID := "testpricewebhook"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)

	itemID, _ := strconv.Atoi(CreateTestItem(t, "Test Item Price Webhook", 1000, "", ""))

	orderCode := int64(9555233246002522)
	_, err = database.Db.CreateOrder(database.Order{
		OrderCode: null.StringFrom(strconv.FormatInt(orderCode, 10)),
		Vendor:    vendor.ID,
		Timestamp: time.Now(),
		Entries: []database.OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Price:    1000,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	})
	require.NoError(t, err)

	setupPriceWebhookTest(t, orderCode, "Ein gutes Leben für alle!")
	transactionCostsIDStr := CreateTestItem(t, config.Config.TransactionCostsName, 1, "", "")
	transactionCostsID, _ := strconv.Atoi(transactionCostsIDStr)

	require.Equal(t, http.StatusOK, postPriceWebhook(orderCode, 0.15).Code)
	require.Equal(t, http.StatusOK, postPriceWebhook(orderCode, 0.15).Code)

	order, err := database.Db.GetOrderByOrderCode(strconv.FormatInt(orderCode, 10))
	require.NoError(t, err)
	costEntries := 0
	for _, entry := range order.Entries {
		if entry.Item == transactionCostsID {
			costEntries++
			require.Equal(t, 15, entry.Quantity)
		}
	}
	require.NotZero(t, costEntries)

	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	if settings.OrgaCoversTransactionCosts {
		require.Equal(t, 2, costEntries)
	} else {
		require.Equal(t, 1, costEntries)
	}
}

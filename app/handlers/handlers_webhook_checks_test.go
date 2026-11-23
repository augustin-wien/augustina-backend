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
	"github.com/augustin-wien/augustina-backend/paymentprovider"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// fakeVivaTransaction answers every transaction lookup with the given details
func fakeVivaTransaction(t *testing.T, orderCode int64, amount float64, statusID string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/connect/token" {
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
			return
		}
		_ = json.NewEncoder(w).Encode(paymentprovider.TransactionVerificationResponse{
			OrderCode: paymentprovider.VivaOrderCode(orderCode),
			Amount:    amount,
			StatusID:  statusID,
		})
	}))
	t.Cleanup(server.Close)

	orig := config.Config
	t.Cleanup(func() { config.Config = orig })
	config.Config.VivaWalletAPIURL = server.URL
	config.Config.VivaWalletAccountsURL = server.URL
	config.Config.VivaWalletSmartCheckoutClientID = "id"
	config.Config.VivaWalletSmartCheckoutClientKey = "key"
}

func TestSuccessWebhookChecks(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	const orderCode int64 = 7012345678901234
	setup := func(t *testing.T, transactionID string, verified bool) database.Order {
		t.Helper()
		require.NoError(t, database.Db.InitEmptyTestDb())
		createTestVendor(t, "testwebhookchecks")
		vendor, err := database.Db.GetVendorByLicenseID("testwebhookchecks")
		require.NoError(t, err)
		buyer, err := database.Db.GetAccountTypeID("UserAnon")
		require.NoError(t, err)
		vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
		require.NoError(t, err)
		itemID, err := strconv.Atoi(CreateTestItem(t, "Webhook Check Item", 100, "", ""))
		require.NoError(t, err)
		id, err := database.Db.CreateOrder(database.Order{
			OrderCode:     null.StringFrom(strconv.FormatInt(orderCode, 10)),
			TransactionID: transactionID,
			Verified:      verified,
			Vendor:        vendor.ID,
			Timestamp:     time.Now(),
			Entries:       []database.OrderEntry{{Item: itemID, Quantity: 1, Sender: buyer, Receiver: vendorAccount.ID, IsSale: true}},
		})
		require.NoError(t, err)
		order, err := database.Db.GetOrderByID(id)
		require.NoError(t, err)
		return order
	}
	webhook := func(transactionID string, amount float64, statusID string) paymentprovider.TransactionSuccessRequest {
		var req paymentprovider.TransactionSuccessRequest
		req.EventData.OrderCode = paymentprovider.VivaOrderCode(orderCode)
		req.EventData.TransactionID = transactionID
		req.EventData.Amount = amount
		req.EventData.StatusID = statusID
		return req
	}

	t.Run("already verified order is skipped", func(t *testing.T) {
		setup(t, "done-tx", true)
		require.NoError(t, paymentprovider.HandlePaymentSuccessfulResponse(webhook("other-tx", 1, "F")))
	})

	t.Run("invalid transaction id", func(t *testing.T) {
		setup(t, "", false)
		err := paymentprovider.HandlePaymentSuccessfulResponse(webhook("a/b", 1, "F"))
		require.ErrorIs(t, err, paymentprovider.ErrInvalidTransactionID)
	})

	t.Run("amount differs from VivaWallet", func(t *testing.T) {
		order := setup(t, "", false)
		fakeVivaTransaction(t, orderCode, 2, "F")
		require.Error(t, paymentprovider.HandlePaymentSuccessfulResponse(webhook("tx-amount", 1, "F")))
		// The real transaction id replaces the placeholder even though verification failed
		order, err := database.Db.GetOrderByID(order.ID)
		require.NoError(t, err)
		require.Equal(t, "tx-amount", order.TransactionID)
		require.False(t, order.Verified)
	})

	t.Run("status differs from VivaWallet", func(t *testing.T) {
		setup(t, "", false)
		fakeVivaTransaction(t, orderCode, 1, "F")
		require.Error(t, paymentprovider.HandlePaymentSuccessfulResponse(webhook("tx-status", 1, "MW")))
	})

	t.Run("paid amount differs from the order", func(t *testing.T) {
		order := setup(t, "", false)
		fakeVivaTransaction(t, orderCode, 5, "F")
		require.Error(t, paymentprovider.HandlePaymentSuccessfulResponse(webhook("tx-sum", 5, "F")))
		order, err := database.Db.GetOrderByID(order.ID)
		require.NoError(t, err)
		require.False(t, order.Verified)
	})

	t.Run("confirmed transaction replaces a stored transaction id", func(t *testing.T) {
		order := setup(t, "old-tx", false)
		fakeVivaTransaction(t, orderCode, 1, "F")
		require.NoError(t, paymentprovider.HandlePaymentSuccessfulResponse(webhook("new-tx", 1, "F")))
		order, err := database.Db.GetOrderByID(order.ID)
		require.NoError(t, err)
		require.True(t, order.Verified)
		require.Equal(t, "new-tx", order.TransactionID)
	})
}

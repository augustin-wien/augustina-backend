package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/paymentprovider"
	"github.com/stretchr/testify/require"
)

func webhookFor(code int64, tx string, amount float64) paymentprovider.TransactionSuccessRequest {
	var req paymentprovider.TransactionSuccessRequest
	req.EventData.OrderCode = paymentprovider.VivaOrderCode(code)
	req.EventData.TransactionID = tx
	req.EventData.Amount = amount
	req.EventData.StatusID = "F"
	return req
}

func TestSuccessWebhookPaths(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())
	viva := startSweepViva(t)

	t.Run("simulated in development", func(t *testing.T) {
		config.Config.Development = true
		v, a := sweepVendor(t)
		orderID, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
		require.NoError(t, paymentprovider.HandlePaymentSuccessfulResponse(webhookFor(code, fmt.Sprintf("dev-simulation-%d", code), 3)))
		order, err := database.Db.GetOrderByID(orderID)
		require.NoError(t, err)
		require.True(t, order.Verified)
	})

	t.Run("verified order is sent to Odoo", func(t *testing.T) {
		config.Config.Development = false
		var received atomic.Int32
		odoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		defer odoo.Close()
		settings, err := database.Db.GetSettings()
		require.NoError(t, err)
		settings.OdooWebhookURL = odoo.URL
		require.NoError(t, database.Db.UpdateSettings(settings))
		t.Cleanup(func() {
			settings.OdooWebhookURL = ""
			require.NoError(t, database.Db.UpdateSettings(settings))
		})

		v, a := sweepVendor(t)
		orderID, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
		tx := fmt.Sprintf("odoo-tx-%d", code)
		viva.register(tx, code, 3)
		require.NoError(t, paymentprovider.HandlePaymentSuccessfulResponse(webhookFor(code, tx, 3)))
		require.Eventually(t, func() bool {
			order, err := database.Db.GetOrderByID(orderID)
			return err == nil && order.OdooSyncedAt.Valid && received.Load() == 1
		}, 5*time.Second, 50*time.Millisecond)
	})

	t.Run("transaction of another order", func(t *testing.T) {
		config.Config.Development = false
		v, a := sweepVendor(t)
		orderID, code := sweepUnverifiedOrder(t, v, a, sweepItem(t, "normal_item"))
		tx := fmt.Sprintf("foreign-tx-%d", code)
		viva.register(tx, code+1, 3)
		require.Error(t, paymentprovider.HandlePaymentSuccessfulResponse(webhookFor(code, tx, 3)))
		order, err := database.Db.GetOrderByID(orderID)
		require.NoError(t, err)
		require.False(t, order.Verified)
	})
}

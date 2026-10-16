package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/paymentprovider"
	"github.com/stretchr/testify/require"
)

func setDonationTracking(t *testing.T, enabled bool) {
	err := database.Db.EntClient.Settings.Update().
		SetDonationTrackingEnabled(enabled).
		Exec(context.Background())
	require.NoError(t, err)
}

// postForeignSuccessWebhook posts a success webhook for a terminal payment of 10 € with the
// price test's transaction id
func postForeignSuccessWebhook(t *testing.T) *httptest.ResponseRecorder {
	originalSourceCode := config.Config.VivaWalletSourceCode
	t.Cleanup(func() { config.Config.VivaWalletSourceCode = originalSourceCode })
	config.Config.VivaWalletSourceCode = "1234"

	payloadBytes, _ := json.Marshal(paymentprovider.TransactionSuccessRequest{
		EventData: paymentprovider.EventData{
			OrderCode:     6289177113801428,
			TransactionID: priceTestTransactionID,
			Amount:        10.00,
			StatusID:      "F",
			SourceCode:    "Default",
			SourceName:    "Spendenterminal",
			TerminalID:    16000001,
			InsDate:       "2026-10-15T19:42:41.48+02:00",
		},
	})
	req := httptest.NewRequest("POST", "/webhooks/vivawallet/success/", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	VivaWalletWebhookSuccess(rec, req)
	return rec
}

func listExternalPayments(t *testing.T, query url.Values) []ExternalPayment {
	req := httptest.NewRequest("GET", "/api/external-payments/?"+query.Encode(), nil)
	rec := httptest.NewRecorder()
	ListExternalPayments(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var payments []ExternalPayment
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payments))
	return payments
}

func TestExternalPayments_RecordedFromBothWebhooks(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}
	setDonationTracking(t, true)
	setupPriceWebhookTest(t, 6289177113801428, "Spende")

	// The price event can arrive before the success event, and both are retried
	require.Equal(t, http.StatusOK, postPriceWebhook(6289177113801428, 0.29).Code)
	require.Equal(t, http.StatusOK, postForeignSuccessWebhook(t).Code)
	require.Equal(t, http.StatusOK, postForeignSuccessWebhook(t).Code)

	payments := listExternalPayments(t, url.Values{})
	require.Len(t, payments, 1)
	p := payments[0]
	require.Equal(t, priceTestTransactionID, p.TransactionID)
	require.Equal(t, "6289177113801428", p.OrderCode)
	require.Equal(t, 1000, p.Amount)
	require.Equal(t, 29, p.Fee)
	require.Equal(t, "F", p.StatusID)
	require.Equal(t, "Default", p.SourceCode)
	require.Equal(t, "Spendenterminal", p.SourceName)
	require.Equal(t, 16000001, p.TerminalID)
	require.Equal(t, "Spende", p.MerchantTrns)
	require.True(t, p.PaidAt.Equal(time.Date(2026, 10, 15, 17, 42, 41, 480000000, time.UTC)))
}

func TestExternalPayments_NotRecordedWhileTrackingIsOff(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}
	setDonationTracking(t, false)

	require.Equal(t, http.StatusOK, postForeignSuccessWebhook(t).Code)

	require.Empty(t, listExternalPayments(t, url.Values{}))
}

func TestExternalPayments_ListFiltersByPeriod(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	day := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	for i, paidAt := range []time.Time{day.AddDate(0, 0, -1), day, day.AddDate(0, 0, 1)} {
		require.NoError(t, database.Db.RecordExternalPayment(database.ExternalPaymentUpdate{
			TransactionID: "txn-" + string(rune('a'+i)),
			Amount:        500,
			PaidAt:        paidAt,
		}))
	}

	payments := listExternalPayments(t, url.Values{
		"from": {day.Add(-time.Hour).Format(time.RFC3339)},
		"to":   {day.Add(time.Hour).Format(time.RFC3339)},
	})
	require.Len(t, payments, 1)
	require.Equal(t, "txn-b", payments[0].TransactionID)

	require.Len(t, listExternalPayments(t, url.Values{}), 3)

	req := httptest.NewRequest("GET", "/api/external-payments/?from=yesterday", nil)
	rec := httptest.NewRecorder()
	ListExternalPayments(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

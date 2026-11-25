package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestUpdateCSS(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	const cssPath = "public/style.css"
	original, err := os.ReadFile(cssPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.WriteFile(cssPath, original, 0o644))
	})

	css := "body { color: #c00; }\n"
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/settings/css/", css, 200, adminUserToken)

	written, err := os.ReadFile(cssPath)
	require.NoError(t, err)
	require.Equal(t, css, string(written))

	res := utils.TestRequest(t, r, "GET", "/public/style.css", nil, 200)
	require.Equal(t, css, res.Body.String())

	// Only admins may change the stylesheet
	utils.TestRequestStr(t, r, "PUT", "/api/settings/css/", css, 401)
}

func TestResendOdooWebhook(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	var received map[string]any
	odoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer odoo.Close()

	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	origURL := settings.OdooWebhookURL
	settings.OdooWebhookURL = odoo.URL
	require.NoError(t, database.Db.UpdateSettings(settings))
	t.Cleanup(func() {
		settings.OdooWebhookURL = origURL
		require.NoError(t, database.Db.UpdateSettings(settings))
	})

	orderID := createAdminVerifyTestOrder(t, "testlicense-resend", "7777777777")

	res := utils.TestRequestWithAuth(t, r, "POST", "/api/orders/resend/"+strconv.Itoa(orderID)+"/", nil, 200, adminUserToken)
	var resp resendResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	require.Equal(t, "ok", resp.Status)
	require.Equal(t, orderID, resp.OrderID)

	require.NotNil(t, received)
	require.EqualValues(t, orderID, received["order_id"])

	order, err := database.Db.GetOrderByID(orderID)
	require.NoError(t, err)
	require.True(t, order.OdooSyncedAt.Valid)

	utils.TestRequestWithAuth(t, r, "POST", "/api/orders/resend/abc/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/orders/resend/0/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/orders/resend/999999/", nil, 404, adminUserToken)
}

func TestVivaWalletVerificationKey(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	origKey := config.Config.VivaWalletVerificationKey
	t.Cleanup(func() { config.Config.VivaWalletVerificationKey = origKey })

	config.Config.VivaWalletVerificationKey = "test-verification-key"
	for _, path := range []string{"/api/webhooks/vivawallet/success/", "/api/webhooks/vivawallet/failure/", "/api/webhooks/vivawallet/price/"} {
		res := utils.TestRequest(t, r, "GET", path, nil, 200)
		require.JSONEq(t, `{"Key":"test-verification-key"}`, res.Body.String(), path)
	}

	config.Config.VivaWalletVerificationKey = ""
	utils.TestRequest(t, r, "GET", "/api/webhooks/vivawallet/success/", nil, 400)
}

func TestAdminAddTransactionIDToOrderErrors(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	orderCode := "6666666666"
	orderID := createAdminVerifyTestOrder(t, "testlicense-txid", orderCode)
	url := "/api/orders/unverified/code/" + orderCode + "/transactionID/"

	utils.TestRequestStrWithAuth(t, r, "POST", url, "{not json", 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", url, addTransactionIDRequest{}, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/orders/unverified/code/0000000000/transactionID/", addTransactionIDRequest{TransactionID: "tx"}, 400, adminUserToken)

	_, err := database.Db.EntClient.Order.UpdateOneID(orderID).SetVerified(true).Save(t.Context())
	require.NoError(t, err)
	utils.TestRequestWithAuth(t, r, "POST", url, addTransactionIDRequest{TransactionID: "tx"}, 400, adminUserToken)
}

func TestVivaWalletWebhookFailure(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	res := utils.TestRequestStr(t, r, "POST", "/api/webhooks/vivawallet/failure/", `{"EventData": {"OrderCode": 1234567890, "TransactionId": "tx"}}`, 200)
	require.JSONEq(t, `{"Status":"OK"}`, res.Body.String())

	utils.TestRequestStr(t, r, "POST", "/api/webhooks/vivawallet/failure/", "{not json", 400)
}

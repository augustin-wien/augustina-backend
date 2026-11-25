package integrations

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

func TestSendJSONToOdooWebhookHeadersAndRejection(t *testing.T) {
	var auth, dbFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, dbFilter = r.Header.Get("Authorization"), r.Header.Get("X-Odoo-dbfilter")
		// Odoo rejects the payload itself; that is not retried
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	orig := config.Config
	t.Cleanup(func() { config.Config = orig })
	config.Config.OdooWebhookURL = srv.URL
	config.Config.OdooWebhookToken = "secret"
	config.Config.OdooDBFilter = "augustin"

	require.NoError(t, sendJSONToOdooWebhook(1, map[string]int{"order_id": 1}))
	require.Equal(t, "Bearer secret", auth)
	require.Equal(t, "augustin", dbFilter)

	config.Config.OdooWebhookURL = "://invalid"
	require.Error(t, sendJSONToOdooWebhook(1, map[string]int{}))
	require.Error(t, sendJSONToOdooWebhook(1, make(chan int)))
}

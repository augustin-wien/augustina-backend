package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// withFailingKeycloakAdmin routes the backend's Keycloak calls through a proxy
// that passes logins and the lookups of the auth middleware to Keycloak, but
// answers every other admin API request with a server error
func withFailingKeycloakAdmin(t *testing.T) {
	t.Helper()
	target, err := url.Parse(strings.TrimRight(config.Config.KeycloakHostname, "/"))
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authLookup := r.Method == http.MethodGet &&
			(strings.Contains(r.URL.Path, "/role-mappings/") || strings.HasSuffix(r.URL.Path, "/groups"))
		if strings.Contains(r.URL.Path, "/admin/realms/") && !authLookup {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	orig := keycloak.KeycloakClient.Client
	keycloak.KeycloakClient.Client = gocloak.NewClient(srv.URL)
	t.Cleanup(func() { keycloak.KeycloakClient.Client = orig })
}

func TestHandlersReportKeycloakErrors(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	origConfig := config.Config
	t.Cleanup(func() { config.Config = origConfig })
	config.Config.SMTPSenderAddress = "noreply@example.com"
	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	settings.OnlinePaperUrl = "https://zeitung.example.com"
	require.NoError(t, database.Db.UpdateSettings(settings))

	v, _ := sweepVendor(t)
	var customer *database.Customer
	vendorJSON := fmt.Sprintf(`{"licenseID":%q,"firstName":"Neu","lastName":"Name","email":"changed-%d@example.com"}`, v.LicenseID.String, v.ID)
	customer, err = database.Db.CreateCustomer(&database.Customer{Email: "kc-fail-customer@example.com", KeycloakID: "kc-missing"})
	require.NoError(t, err)
	unlinked, err := database.Db.CreateCustomer(&database.Customer{Email: "kc-fail-unlinked@example.com"})
	require.NoError(t, err)
	issue, err := database.Db.CreateItem(database.Item{Name: "KC Fail Issue", Description: "Test item", Price: 300, Type: "online_issue", LicenseGroup: null.StringFrom("kc_fail")})
	require.NoError(t, err)

	withFailingKeycloakAdmin(t)

	cases := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/api/vendors/", `{"licenseID":"kc-fail-new","firstName":"A","lastName":"B","email":"kc-fail-new@example.com"}`, 400},
		{"PUT", fmt.Sprintf("/api/vendors/%d/", v.ID), vendorJSON, 400},
		{"POST", fmt.Sprintf("/api/vendors/%d/verify-email/", v.ID), "", 400},
		{"POST", fmt.Sprintf("/api/customers/%d/password-reset-email/", customer.ID), "", 502},
		{"POST", fmt.Sprintf("/api/customers/%d/verify-email/", unlinked.ID), "", 502},
		{"PUT", fmt.Sprintf("/api/customers/%d/", customer.ID), `{"email":"kc-fail-new-address@example.com"}`, 400},
		{"POST", fmt.Sprintf("/api/items/%d/notify-abonements/", issue), "", 500},
		// Deleting a vendor goes on when the Keycloak user cannot be removed
		{"DELETE", fmt.Sprintf("/api/vendors/%d/", v.ID), "", 204},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			utils.TestRequestStrWithAuth(t, r, tc.method, tc.path, tc.body, tc.want, adminUserToken)
		})
	}
}

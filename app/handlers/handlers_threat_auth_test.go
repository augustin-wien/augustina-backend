package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// publicRoutes may be called without a token
var publicRoutes = map[string]bool{
	"GET /api/hello/":                        true,
	"GET /api/settings/":                     true,
	"POST /api/feedback/":                    true,
	"GET /api/vendors/check/{licenseID}/":    true,
	"GET /api/items/":                        true,
	"GET /api/campaigns/active/":             true,
	"POST /api/campaigns/{id}/click/":        true,
	"POST /api/campaigns/{id}/view/":         true,
	"POST /api/orders/":                      true,
	"POST /api/orders/failure/":              true,
	"GET /api/orders/verify/":                true,
	"GET /api/pdf/{id}/":                     true,
	"GET /api/pdf/{id}/validate/":            true,
	"GET /api/webhooks/vivawallet/success/":  true,
	"POST /api/webhooks/vivawallet/success/": true,
	"GET /api/webhooks/vivawallet/failure/":  true,
	"POST /api/webhooks/vivawallet/failure/": true,
	"GET /api/webhooks/vivawallet/price/":    true,
	"POST /api/webhooks/vivawallet/price/":   true,
	"GET /healthz":                           true,
	"GET /readyz":                            true,
	"GET /img/thumbs/*":                      true,
}

// staticPrefixes serve files and are public
var staticPrefixes = []string{"/img/", "/public/", "/docs/", "/swagger/"}

// customerRoutes need a customer, vendorRoutes a vendor, authRoutes any login
var (
	customerRoutes = map[string]bool{"GET /api/customers/me/abonements/": true, "GET /api/customers/me/payments/": true}
	vendorRoutes   = map[string]bool{"GET /api/vendors/me/": true}
	authRoutes     = map[string]bool{"GET /api/auth/hello/": true}
)

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

type route struct{ method, pattern, path string }

// protectedRoutes lists every registered route that is not public
func protectedRoutes(t *testing.T) []route {
	t.Helper()
	var out []route
	require.NoError(t, chi.Walk(r, func(method, pattern string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if publicRoutes[method+" "+pattern] {
			return nil
		}
		for _, p := range staticPrefixes {
			if strings.HasPrefix(pattern, p) {
				return nil
			}
		}
		out = append(out, route{method, pattern, routeParam.ReplaceAllString(pattern, "1")})
		return nil
	}))
	require.NotEmpty(t, out)
	return out
}

func call(method, path, auth string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return utils.SubmitRequest(req, r)
}

// loginAs creates a Keycloak user in the given group (none if empty) and returns its token
func loginAs(t *testing.T, email, group string) (string, *gocloak.JWT) {
	t.Helper()
	_ = keycloak.KeycloakClient.DeleteUser(email)
	id, err := keycloak.KeycloakClient.CreateUser(email, "Threat", "Actor", email, "password")
	require.NoError(t, err)
	t.Cleanup(func() { _ = keycloak.KeycloakClient.DeleteUser(email) })
	if group != "" {
		require.NoError(t, keycloak.KeycloakClient.AssignGroup(id, group))
	}
	token, err := keycloak.KeycloakClient.GetUserToken(email, "password")
	require.NoError(t, err)
	return id, token
}

// withOdooEnabled makes the Odoo API answer, so its routes are checked as well
// Reading the settings mirrors the Odoo URL into the config, so it is set in both.
func withOdooEnabled(t *testing.T) {
	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	orig := settings.OdooWebhookURL
	settings.OdooWebhookURL = "http://127.0.0.1:1"
	require.NoError(t, database.Db.UpdateSettings(settings))
	t.Cleanup(func() {
		settings.OdooWebhookURL = orig
		require.NoError(t, database.Db.UpdateSettings(settings))
	})
}

func TestEveryProtectedRouteNeedsAToken(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())
	withOdooEnabled(t)

	forged := map[string]string{
		"X-Auth-User": "admin", "X-Auth-User-Validated": "true", "X-Auth-Roles-admin": "admin",
		"X-Auth-Roles-backoffice": "backoffice", "X-Auth-Roles-odoo": "odoo", "X-Auth-Groups-vendors": "vendors",
		"X-Auth-Groups-customer": "customer",
	}
	none := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","realm_access":{"roles":["admin"]}}`)) + "."
	selfSigned := none + base64.RawURLEncoding.EncodeToString([]byte("signature"))

	for _, rt := range protectedRoutes(t) {
		t.Run(rt.method+" "+rt.pattern, func(t *testing.T) {
			for name, auth := range map[string]string{
				"no token":          "",
				"garbage token":     "Bearer garbage",
				"basic auth":        "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin")),
				"unsigned JWT":      "Bearer " + none,
				"self-signed JWT":   "Bearer " + selfSigned,
				"token without key": "Bearer ",
			} {
				res := call(rt.method, rt.path, auth, forged)
				require.Equal(t, http.StatusUnauthorized, res.Code, "%s: %s", name, res.Body.String())
			}
		})
	}
}

func TestLowPrivilegeUsersCannotUseAdminRoutes(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())
	withOdooEnabled(t)

	_, plain := loginAs(t, "threat-plain@example.com", "")
	_, customer := loginAs(t, "threat-customer@example.com", keycloak.KeycloakClient.CustomerGroup)
	_, vendor := loginAs(t, "threat-vendor@example.com", keycloak.KeycloakClient.VendorGroup)
	forgedAdmin := map[string]string{"X-Auth-Roles-admin": "admin", "X-Auth-Roles-backoffice": "backoffice", "X-Auth-Roles-odoo": "odoo"}

	for _, rt := range protectedRoutes(t) {
		key := rt.method + " " + rt.pattern
		if authRoutes[key] {
			continue
		}
		t.Run(key, func(t *testing.T) {
			users := map[string]*gocloak.JWT{"plain user": plain, "customer": customer, "vendor": vendor}
			if strings.HasPrefix(rt.pattern, "/api/odoo/") || strings.HasPrefix(rt.pattern, "/api/flour/") {
				// Backoffice admins have no business in the Odoo API either
				users["backoffice admin"] = adminUserToken
			}
			for name, token := range users {
				if customerRoutes[key] && name == "customer" || vendorRoutes[key] && name == "vendor" {
					continue
				}
				res := call(rt.method, rt.path, "Bearer "+token.AccessToken, forgedAdmin)
				require.Equal(t, http.StatusForbidden, res.Code, "%s: %s", name, res.Body.String())
			}
		})
	}
}

// TestCustomersOnlySeeTheirOwnData checks that a customer cannot read another
// customer's payments or abonements by forging identity headers
func TestCustomersOnlySeeTheirOwnData(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	attackerID, attacker := loginAs(t, "threat-attacker@example.com", keycloak.KeycloakClient.CustomerGroup)
	_, err := database.Db.CreateCustomer(&database.Customer{Email: "threat-attacker@example.com", KeycloakID: attackerID})
	require.NoError(t, err)

	victimEmail := "threat-victim@example.com"
	victim, err := database.Db.CreateCustomer(&database.Customer{Email: victimEmail, KeycloakID: "kc-victim"})
	require.NoError(t, err)
	v, a := sweepVendor(t)
	item := sweepItem(t, "normal_item")
	buyer, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	orderID, err := database.Db.CreateOrder(database.Order{
		Vendor: v.ID, OrderCode: null.StringFrom("7700000001"), CustomerEmail: null.StringFrom(victimEmail),
		Entries: []database.OrderEntry{{Item: item, Quantity: 1, Sender: buyer, Receiver: a.ID, IsSale: true}},
	})
	require.NoError(t, err)
	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orderID, 0))
	_, err = database.Db.CreateAbonement(&database.Abonement{CustomerID: victim.ID, Status: "active"})
	require.NoError(t, err)

	forged := map[string]string{"X-Auth-User": "kc-victim", "X-Auth-User-Email": victimEmail, "X-Auth-User-Email-Verified": "true"}
	for _, path := range []string{"/api/customers/me/payments/", "/api/customers/me/abonements/"} {
		res := call("GET", path, "Bearer "+attacker.AccessToken, forged)
		require.NotContains(t, res.Body.String(), victimEmail, path)
		require.NotContains(t, res.Body.String(), fmt.Sprintf(`"customer_id":%d`, victim.ID), path)
	}
	// The admin routes for a customer stay closed
	res := call("GET", fmt.Sprintf("/api/customers/%d/", victim.ID), "Bearer "+attacker.AccessToken, forged)
	require.Equal(t, http.StatusForbidden, res.Code)
}

// TestDevelopmentRoutesAreOffInProduction checks that API documentation is only served in development
func TestDevelopmentRoutesAreOffInProduction(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	orig := config.Config.Development
	config.Config.Development = false
	prod := GetRouter()
	config.Config.Development = orig
	t.Cleanup(func() { r = GetRouter() })

	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json", "/docs/", "/docs/swagger.json"} {
		req := httptest.NewRequest("GET", path, nil)
		res := utils.SubmitRequest(req, prod)
		require.NotEqual(t, http.StatusOK, res.Code, path)
	}
}

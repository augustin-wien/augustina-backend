package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/stretchr/testify/require"
)

func TestCustomerAuthMiddlewareAllowsCustomerGroup(t *testing.T) {
	originalCustomerGroup := keycloak.KeycloakClient.CustomerGroup
	keycloak.KeycloakClient.CustomerGroup = "customer"
	defer func() {
		keycloak.KeycloakClient.CustomerGroup = originalCustomerGroup
	}()

	called := false
	handler := CustomerAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-User-Validated", "true")
	req.Header.Set("X-Auth-Groups-customer", "customer")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, res.Code)
}

func TestCustomerAuthMiddlewareRejectsMissingCustomerAccess(t *testing.T) {
	originalCustomerGroup := keycloak.KeycloakClient.CustomerGroup
	keycloak.KeycloakClient.CustomerGroup = "customer"
	defer func() {
		keycloak.KeycloakClient.CustomerGroup = originalCustomerGroup
	}()

	called := false
	handler := CustomerAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-User-Validated", "true")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	require.False(t, called)
	require.Equal(t, http.StatusForbidden, res.Code)
}

// TestClearIncomingAuthHeadersStripsForgedHeaders verifies that every client-supplied
// X-Auth-* header is removed and that validation is reset to false. This is the core
// defense against privilege escalation via forged auth headers.
func TestClearIncomingAuthHeadersStripsForgedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	forged := []string{
		"X-Auth-User",
		"X-Auth-User-Name",
		"X-Auth-User-Email",
		"X-Auth-User-Validated",
		"X-Auth-Roles-admin",
		"X-Auth-Roles-backoffice",
		"X-Auth-Roles-customer",
		"X-Auth-Roles-flour",
		"X-Auth-Roles-odoo",
		"X-Auth-Groups-customer",
		"X-Auth-Groups-Vendors",
		"X-Auth-Foo-Bar",
	}
	for _, h := range forged {
		req.Header.Set(h, "forged")
	}

	clearIncomingAuthHeaders(req)

	for _, h := range forged {
		if h == "X-Auth-User-Validated" {
			continue
		}
		require.Empty(t, req.Header.Get(h), "expected forged header %s to be stripped", h)
	}
	// A non X-Auth- header must be left untouched.
	req.Header.Set("Authorization", "Bearer token")
	require.Equal(t, "false", req.Header.Get("X-Auth-User-Validated"))
	require.Equal(t, "Bearer token", req.Header.Get("Authorization"))
}

// TestAdminAuthMiddlewareRejectsForgedBackofficeAfterClear proves the escalation is closed:
// a request carrying a forged X-Auth-Roles-backoffice header no longer reaches admin routes
// once the auth headers are stripped, even after the request is later marked validated.
func TestAdminAuthMiddlewareRejectsForgedBackofficeAfterClear(t *testing.T) {
	called := false
	handler := AdminAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Attacker forges the backoffice role.
	req.Header.Set("X-Auth-Roles-backoffice", "backoffice")

	// AuthMiddleware strips forged headers before validating the token.
	clearIncomingAuthHeaders(req)
	// Simulate a successfully validated but low-privilege token.
	req.Header.Set("X-Auth-User-Validated", "true")

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.False(t, called, "forged backoffice role must not grant admin access")
	require.Equal(t, http.StatusForbidden, res.Code)
}

// TestAdminAuthMiddlewareAllowsGenuineAdminAfterClear is the positive counterpart: a role
// set legitimately (as AuthMiddleware would from a validated token) still grants access.
func TestAdminAuthMiddlewareAllowsGenuineAdminAfterClear(t *testing.T) {
	called := false
	handler := AdminAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	clearIncomingAuthHeaders(req)
	// AuthMiddleware would set these from the validated token.
	req.Header.Set("X-Auth-User-Validated", "true")
	req.Header.Add("X-Auth-Roles-admin", "admin")

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, res.Code)
}

// TestCustomerAuthMiddlewareRejectsForgedCustomerAfterClear covers the customer role/group
// path of the same escalation.
func TestCustomerAuthMiddlewareRejectsForgedCustomerAfterClear(t *testing.T) {
	originalCustomerGroup := keycloak.KeycloakClient.CustomerGroup
	keycloak.KeycloakClient.CustomerGroup = "customer"
	defer func() {
		keycloak.KeycloakClient.CustomerGroup = originalCustomerGroup
	}()

	called := false
	handler := CustomerAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Roles-customer", "customer")
	req.Header.Set("X-Auth-Groups-customer", "customer")

	clearIncomingAuthHeaders(req)
	req.Header.Set("X-Auth-User-Validated", "true")

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.False(t, called, "forged customer role/group must not grant customer access")
	require.Equal(t, http.StatusForbidden, res.Code)
}

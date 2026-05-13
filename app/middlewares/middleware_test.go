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

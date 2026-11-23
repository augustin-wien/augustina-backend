package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/stretchr/testify/require"
)

func TestRoleMiddlewaresBasics(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
	for name, mw := range map[string]func(http.Handler) http.Handler{
		"vendor": VendorAuthMiddleware, "admin": AdminAuthMiddleware, "customer": CustomerAuthMiddleware,
	} {
		t.Run(name, func(t *testing.T) {
			h := mw(ok)
			serve := func(method string, headers map[string]string) int {
				req := httptest.NewRequest(method, "/", nil)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, req)
				return rr.Code
			}
			require.Equal(t, http.StatusTeapot, serve(http.MethodOptions, nil))
			require.Equal(t, http.StatusUnauthorized, serve(http.MethodGet, map[string]string{"X-Auth-User-Validated": "false"}))
			require.Equal(t, http.StatusForbidden, serve(http.MethodGet, map[string]string{"X-Auth-User-Validated": "true"}))
			require.Equal(t, http.StatusTeapot, serve(http.MethodGet, map[string]string{"X-Auth-User-Validated": "true", "X-Auth-Roles-admin": "admin"}))
		})
	}
}

// TestAuthMiddlewareKeycloakFailures checks the answers when Keycloak cannot
// confirm the token or load the user's roles or groups
func TestAuthMiddlewareKeycloakFailures(t *testing.T) {
	fail := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "admin", "expires_in": 300})
		case strings.HasSuffix(r.URL.Path, "/userinfo"):
			if fail == "userinfo" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"sub":"user-1","preferred_username":"u","email":"u@example.com","email_verified":true}`))
		case strings.Contains(r.URL.Path, "/role-mappings/"):
			if fail == "roles" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`[{"name":"backoffice"}]`))
		case strings.HasSuffix(r.URL.Path, "/groups"):
			if fail == "groups" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`[{"name":"vendors"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	k := &keycloak.KeycloakClient
	origClient, origRealm, origCtx := k.Client, k.Realm, k.Context
	t.Cleanup(func() { k.Client, k.Realm, k.Context = origClient, origRealm, origCtx })
	k.Client, k.Realm = gocloak.NewClient(srv.URL), "augustin"
	if k.Context == nil {
		k.Context = context.Background()
	}

	var seen http.Header
	h := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.WriteHeader(http.StatusTeapot)
	}))
	serve := func(auth string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		return rr.Code
	}

	require.Equal(t, http.StatusTeapot, serve("Bearer token"))
	require.Equal(t, "backoffice", seen.Get("X-Auth-Roles-backoffice"))
	require.Equal(t, "vendors", seen.Get("X-Auth-Groups-vendors"))
	require.Equal(t, "true", seen.Get("X-Auth-User-Email-Verified"))

	require.Equal(t, http.StatusUnauthorized, serve(""))
	fail = "userinfo"
	require.Equal(t, http.StatusUnauthorized, serve("Bearer token"))
	fail = "roles"
	require.Equal(t, http.StatusInternalServerError, serve("Bearer token"))
	fail = "groups"
	require.Equal(t, http.StatusInternalServerError, serve("Bearer token"))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusTeapot, rr.Code)
}

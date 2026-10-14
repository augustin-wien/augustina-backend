package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeKeycloak issues numbered service account tokens and answers admin API
// requests with 401 for the tokens in rejected
type fakeKeycloak struct {
	logins     atomic.Int32
	failLogins atomic.Bool
	rejected   []string
}

func newFakeKeycloak(t *testing.T, rejected ...string) (*Keycloak, *fakeKeycloak) {
	fake := &fakeKeycloak{rejected: rejected}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token"):
			if fake.failLogins.Load() {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
				return
			}
			n := fake.logins.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": fmt.Sprintf("token-%d", n), "expires_in": 300})
		case strings.Contains(r.URL.Path, "/admin/realms/"):
			if slices.Contains(fake.rejected, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"HTTP 401 Unauthorized"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"user-1"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	k := &Keycloak{
		hostname:     srv.URL,
		ClientID:     "backend",
		ClientSecret: "secret",
		Realm:        "augustin",
		Context:      context.Background(),
	}
	k.Client = k.newGoCloakClient()
	k.checkAdminToken()
	require.Equal(t, "token-1", k.clientToken.AccessToken)
	return k, fake
}

func TestRejectedAdminTokenIsRenewed(t *testing.T) {
	k, fake := newFakeKeycloak(t, "token-1")

	user, err := k.GetUserByID("user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", *user.ID)
	require.EqualValues(t, 2, fake.logins.Load())
	require.Equal(t, "token-2", k.clientToken.AccessToken)
}

func TestRejectedAdminTokenIsRetriedOnlyOnce(t *testing.T) {
	k, fake := newFakeKeycloak(t, "token-1", "token-2")

	_, err := k.GetUserByID("user-1")
	require.ErrorContains(t, err, "401")
	require.EqualValues(t, 2, fake.logins.Load())
}

func TestAdminTokenIsRenewedBeforeExpiry(t *testing.T) {
	k, fake := newFakeKeycloak(t)
	require.WithinDuration(t, time.Now().Add(300*time.Second-tokenRenewMargin), k.clientTokenRenewAt, time.Second)

	k.checkAdminToken()
	require.EqualValues(t, 1, fake.logins.Load())

	k.clientTokenRenewAt = time.Now().Add(-time.Second)
	k.checkAdminToken()
	require.EqualValues(t, 2, fake.logins.Load())
	require.Equal(t, "token-2", k.clientToken.AccessToken)
}

func TestFailedLoginKeepsAdminToken(t *testing.T) {
	k, fake := newFakeKeycloak(t)
	fake.failLogins.Store(true)

	k.clientTokenRenewAt = time.Now().Add(-time.Second)
	k.checkAdminToken()
	require.NotNil(t, k.clientToken)
	require.Equal(t, "token-1", k.clientToken.AccessToken)
}

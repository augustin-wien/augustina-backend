package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// newUserLookupKeycloak answers user searches with the users in byEmail, bySearch
// or byUsername, matched by the query parameter, and user creation with createStatus.
func newUserLookupKeycloak(t *testing.T, byUsername, byEmail, bySearch []map[string]string, createStatus int) (*Keycloak, *atomic.Int32) {
	var creates atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token"):
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token", "expires_in": 300})
		case strings.HasSuffix(r.URL.Path, "/users") && r.Method == http.MethodPost:
			creates.Add(1)
			w.WriteHeader(createStatus)
		case strings.HasSuffix(r.URL.Path, "/users"):
			q := r.URL.Query()
			users := []map[string]string{}
			switch {
			case q.Get("username") != "":
				users = byUsername
			case q.Get("email") != "":
				users = byEmail
			case q.Get("search") != "":
				users = bySearch
			}
			_ = json.NewEncoder(w).Encode(users)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	k := &Keycloak{hostname: srv.URL, ClientID: "backend", ClientSecret: "secret", Realm: "augustin", Context: context.Background()}
	k.Client = k.newGoCloakClient()
	return k, &creates
}

func TestGetOrCreateUserFindsUserByEmail(t *testing.T) {
	k, creates := newUserLookupKeycloak(t, nil, []map[string]string{{"id": "user-7", "username": "reader", "email": "reader@example.com"}}, nil, http.StatusCreated)

	id, created, err := k.GetOrCreateUser("reader@example.com")
	require.NoError(t, err)
	require.Equal(t, "user-7", id)
	require.False(t, created)
	require.Zero(t, creates.Load())
}

func TestGetOrCreateUserReusesUserOnConflict(t *testing.T) {
	k, _ := newUserLookupKeycloak(t, nil, nil, []map[string]string{{"id": "user-8", "username": "Reader@Example.com"}}, http.StatusConflict)

	id, created, err := k.GetOrCreateUser("reader@example.com")
	require.NoError(t, err)
	require.Equal(t, "user-8", id)
	require.False(t, created)
}

func TestAssignGroupRejectsEmptyUserID(t *testing.T) {
	k, _ := newUserLookupKeycloak(t, nil, nil, nil, http.StatusCreated)

	require.ErrorContains(t, k.AssignGroup("", "customer"), "userID is empty")
	require.ErrorContains(t, k.AssignDigitalLicenseGroup("", "digital_edition"), "userID is empty")
}

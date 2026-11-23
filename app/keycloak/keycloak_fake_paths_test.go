package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/stretchr/testify/require"
)

// scriptedKeycloak is a fake Keycloak admin API whose answers the tests set
type scriptedKeycloak struct {
	groupSearch map[string]string // search term -> JSON list of groups
	groups      map[string]string // group id -> JSON group
	users       map[string]string // query (email=, username=, search=) -> JSON list of users
	userByID    map[string]string // user id -> JSON user
	createUser  int               // status of POST /users
	updateUser  int               // status of PUT /users/{id}
	addGroup    int               // status of PUT /users/{id}/groups/{gid}
}

func newScriptedKeycloak(t *testing.T, s *scriptedKeycloak) *Keycloak {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/admin/realms/augustin")
		q := r.URL.Query()
		reply := func(status int, body string) {
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "admin", "expires_in": 300})
		case path == "/groups":
			reply(0, orDefault(s.groupSearch[q.Get("search")], "[]"))
		case strings.HasPrefix(path, "/groups/"):
			if g, ok := s.groups[strings.TrimPrefix(path, "/groups/")]; ok {
				reply(0, g)
			} else {
				reply(http.StatusNotFound, `{}`)
			}
		case path == "/users" && r.Method == http.MethodPost:
			if s.createUser == http.StatusCreated || s.createUser == 0 {
				w.Header().Set("Location", r.URL.String()+"/new-user")
				w.WriteHeader(http.StatusCreated)
				return
			}
			reply(s.createUser, `{"errorMessage":"User exists"}`)
		case path == "/users":
			key := ""
			for _, k := range []string{"email", "username", "search"} {
				if v := q.Get(k); v != "" {
					key = k + "=" + v
				}
			}
			reply(0, orDefault(s.users[key], "[]"))
		case strings.Contains(path, "/groups/") && r.Method == http.MethodPut:
			reply(orDefaultInt(s.addGroup, http.StatusNoContent), "")
		case strings.HasPrefix(path, "/users/") && r.Method == http.MethodPut:
			reply(orDefaultInt(s.updateUser, http.StatusNoContent), "")
		case strings.HasPrefix(path, "/users/"):
			if u, ok := s.userByID[strings.TrimPrefix(path, "/users/")]; ok {
				reply(0, u)
			} else {
				reply(http.StatusNotFound, `{}`)
			}
		default:
			reply(http.StatusNotFound, `{}`)
		}
	}))
	t.Cleanup(srv.Close)
	k := &Keycloak{
		hostname: srv.URL, ClientID: "backend", ClientSecret: "secret", Realm: "augustin",
		Context: context.Background(), VendorGroup: "vendors", CustomerGroup: "customer", NewspaperGroup: "newspapers",
	}
	k.Client = gocloak.NewClient(srv.URL)
	return k
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func orDefaultInt(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}

func TestGetGroupByPathFallbacks(t *testing.T) {
	k := newScriptedKeycloak(t, &scriptedKeycloak{
		groupSearch: map[string]string{
			"customer": `[{"id":"c","name":"customer","path":"/customer"}]`,
			// Newer Keycloak versions leave subgroups out of the group itself, so
			// the global search with its hierarchy is the fallback
			"newspapers": `[{"id":"c","name":"customer","path":"/customer","subGroups":[{"id":"n","name":"newspapers","path":"/customer/newspapers"}]}]`,
		},
		groups: map[string]string{
			"c": `{"id":"c","name":"customer","path":"/customer"}`,
			"n": `{"id":"n","name":"newspapers","path":"/customer/newspapers","subGroups":[{"id":"e","name":"edition","path":"/customer/newspapers/edition"}]}`,
		},
	})

	g, err := k.GetGroupByPath("/customer/newspapers/edition")
	require.NoError(t, err)
	require.Equal(t, "e", *g.ID)

	_, err = k.GetGroupByPath("/")
	require.Error(t, err)
	_, err = k.GetGroupByPath("/missing")
	require.Error(t, err)

	g, err = k.GetGroup("customer")
	require.NoError(t, err)
	require.Equal(t, "c", *g.ID)
}

func TestUserCreationConflicts(t *testing.T) {
	existing := `[{"id":"u-1","username":"taken@example.com","email":"taken@example.com"}]`
	k := newScriptedKeycloak(t, &scriptedKeycloak{
		createUser: http.StatusConflict,
		users:      map[string]string{"search=taken@example.com": existing},
		addGroup:   http.StatusInternalServerError,
	})

	// A user that the exact lookups miss is found by the fuzzy search after a conflict
	id, err := k.GetOrCreateVendor("taken@example.com", false)
	require.NoError(t, err)
	require.Equal(t, "u-1", id)
	id, created, err := k.GetOrCreateCustomerUser("", "taken@example.com", "A", "B")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, "u-1", id)

	_, err = k.GetOrCreateVendor("unknown@example.com", false)
	require.Error(t, err)
	_, _, err = k.GetOrCreateCustomerUser("", "unknown@example.com", "A", "B")
	require.Error(t, err)

	// Assigning the vendor group of a found user fails
	_, err = k.UpdateVendor("old@example.com", "taken@example.com", "lic", "A", "B", false)
	require.Error(t, err)
}

func TestUpdateVendorReportsFailedUpdate(t *testing.T) {
	k := newScriptedKeycloak(t, &scriptedKeycloak{
		users: map[string]string{
			"email=old@example.com":   `[{"id":"u-old","username":"old@example.com","email":"old@example.com"}]`,
			"email=other@example.com": `[{"id":"u-old","username":"old@example.com","email":"old@example.com"}]`,
		},
		userByID:   map[string]string{"u-old": `{"id":"u-old","username":"old@example.com","email":"old@example.com"}`},
		updateUser: http.StatusInternalServerError,
	})
	_, err := k.UpdateVendor("old@example.com", "other@example.com", "lic", "A", "B", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "keycloakID=u-old")
}

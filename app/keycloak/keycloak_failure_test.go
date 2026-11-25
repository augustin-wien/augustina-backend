package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

// newFailingKeycloak returns a client whose service account can log in, but
// whose admin API answers every request with a server error.
func newFailingKeycloak(t *testing.T) *Keycloak {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token") {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token", "expires_in": 300})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	t.Cleanup(srv.Close)

	k := &Keycloak{
		hostname:       srv.URL,
		ClientID:       "backend",
		ClientSecret:   "secret",
		Realm:          "augustin",
		Context:        context.Background(),
		VendorGroup:    "vendors",
		CustomerGroup:  "customer",
		NewspaperGroup: "newspapers",
	}
	k.Client = k.newGoCloakClient()
	return k
}

// TestKeycloakErrorsArePassedOn checks that admin API failures reach the caller
func TestKeycloakErrorsArePassedOn(t *testing.T) {
	k := newFailingKeycloak(t)

	origConfig := config.Config
	t.Cleanup(func() { config.Config = origConfig })
	config.Config.SMTPSenderAddress = "noreply@example.com"
	config.Config.OnlinePaperUrl = "https://zeitung.example.com"

	errOf := func(_ any, err error) error { return err }
	errOf2 := func(_, _ any, err error) error { return err }

	cases := map[string]func() error{
		"GetRole":                      func() error { return errOf(k.GetRole("admin")) },
		"CreateRole":                   func() error { return k.CreateRole("x") },
		"DeleteRole":                   func() error { return k.DeleteRole("x") },
		"AssignRole":                   func() error { return k.AssignRole("user", "x") },
		"UnassignRole":                 func() error { return k.UnassignRole("user", "x") },
		"AssignGroup":                  func() error { return k.AssignGroup("user", "vendors") },
		"AssignDigitalLicenseGroup":    func() error { return k.AssignDigitalLicenseGroup("user", "edition") },
		"CreateGroup":                  func() error { return errOf(k.CreateGroup("x")) },
		"CreateSubGroup":               func() error { return errOf(k.CreateSubGroup("x", "parent")) },
		"EnsureGroupPath":              func() error { return k.EnsureGroupPath("/customer/newspapers/x") },
		"GetGroupByPath":               func() error { return errOf(k.GetGroupByPath("/customer/x")) },
		"GetGroup":                     func() error { return errOf(k.GetGroup("x")) },
		"DeleteGroup":                  func() error { return k.DeleteGroup("x") },
		"DeleteSubGroupByPath":         func() error { return k.DeleteSubGroupByPath("/customer/x") },
		"GetUser":                      func() error { return errOf(k.GetUser("a@example.com")) },
		"GetUserByEmail":               func() error { return errOf(k.GetUserByEmail("a@example.com")) },
		"GetUserByID":                  func() error { return errOf(k.GetUserByID("user")) },
		"GetRoles":                     func() error { return errOf(k.GetRoles()) },
		"GetUserRoles":                 func() error { return errOf(k.GetUserRoles("user")) },
		"GetUserGroups":                func() error { return errOf(k.GetUserGroups("user")) },
		"CreateUser":                   func() error { return errOf(k.CreateUser("a@example.com", "A", "B", "a@example.com", "pw")) },
		"GetOrCreateVendor":            func() error { return errOf(k.GetOrCreateVendor("a@example.com", true)) },
		"GetOrCreateUser":              func() error { return errOf2(k.GetOrCreateUser("a@example.com")) },
		"GetOrCreateCustomerUser":      func() error { return errOf2(k.GetOrCreateCustomerUser("kc", "a@example.com", "A", "B")) },
		"SendPasswordResetEmail":       func() error { return k.SendPasswordResetEmail("a@example.com") },
		"SendVerifyEmail":              func() error { return k.SendVerifyEmail("a@example.com") },
		"SendPasswordResetEmailToUser": func() error { return k.SendPasswordResetEmailToUser("user") },
		"SendVerifyEmailToUser":        func() error { return k.SendVerifyEmailToUser("user") },
		"DeleteUser":                   func() error { return k.DeleteUser("a@example.com") },
		"UpdateUserPassword":           func() error { return k.UpdateUserPassword("a@example.com", "pw") },
		"UpdateUser":                   func() error { return k.UpdateUser("a@example.com", "A", "B", "b@example.com") },
		"UpdateUserById":               func() error { return k.UpdateUserById("user", "a", "A", "B", "b@example.com") },
		"UpdateVendor":                 func() error { return errOf(k.UpdateVendor("a@example.com", "b@example.com", "lic", "A", "B", true)) },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, call())
		})
	}

	// Groups that cannot be assigned are logged, not returned
	require.NoError(t, k.SyncLicenseGroupsDiffToKeycloak("user", []string{"old"}, []string{"new"}))
}

func TestKeycloakWithoutClient(t *testing.T) {
	k := &Keycloak{}
	require.Error(t, k.AssignDigitalLicenseGroup("user", "x"))
	_, _, err := k.GetOrCreateCustomerUser("", "a@example.com", "", "")
	require.Error(t, err)
}

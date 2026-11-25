package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// sweepDirect calls the handler prepared by setup once for every database call
// it makes, failing that call, like sweepHandler but without the router
func sweepDirect(t *testing.T, want int, setup func(t *testing.T) (http.HandlerFunc, *http.Request)) {
	t.Helper()
	for failAt := int64(1); ; failAt++ {
		require.Less(t, failAt, int64(400), "too many database calls")
		h, req := setup(t)
		calls := &atomic.Int64{}
		orig := database.Db
		drv := &sweepDriver{Driver: entsql.OpenDB(dialect.Postgres, orig.DB), calls: calls, failAt: failAt}
		database.Db = database.Database{EntClient: ent.NewClient(ent.Driver(drv)), DB: orig.DB, IsProduction: orig.IsProduction}
		rr := httptest.NewRecorder()
		require.NotPanics(t, func() { h(rr, req) }, "failing call %d", failAt)
		database.Db = orig
		if calls.Load() < failAt {
			require.Equal(t, want, rr.Code, rr.Body.String())
			return
		}
	}
}

func TestVendorHandlersSurviveEachDatabaseFailure(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	t.Run("vendor overview", func(t *testing.T) {
		sweepDirect(t, 200, func(t *testing.T) (http.HandlerFunc, *http.Request) {
			v, a := sweepVendor(t)
			sweepVerifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("X-Auth-User-Email", v.Email)
			return GetVendorOverview, req
		})
	})
	t.Run("odoo vendor update", func(t *testing.T) {
		sweepDirect(t, 200, func(t *testing.T) (http.HandlerFunc, *http.Request) {
			v, _ := sweepVendor(t)
			body := fmt.Sprintf(`{"licenseID":%q,"firstName":"Odoo","lastName":"Update","email":%q}`, v.LicenseID.String, v.Email)
			req := httptest.NewRequest("PUT", "/", strings.NewReader(body))
			rc := chi.NewRouteContext()
			rc.URLParams.Add("licenseID", v.LicenseID.String)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
			t.Cleanup(func() { _ = keycloak.KeycloakClient.DeleteUser(v.Email) })
			return UpdateVendorByLicenseID, req
		})
	})
	t.Run("update and delete vendor", func(t *testing.T) {
		sweepHandler(t, 204, func(t *testing.T, run int) sweepRequest {
			v, _ := sweepVendor(t)
			body := fmt.Sprintf(`{"licenseID":%q,"firstName":"Neu","lastName":"Name","email":%q,"isDisabled":true}`, v.LicenseID.String, v.Email)
			utils.TestRequestStrWithAuth(t, r, "PUT", fmt.Sprintf("/api/vendors/%d/?locations=keep", v.ID), body, 200, adminUserToken)
			return sweepRequest{method: "DELETE", path: fmt.Sprintf("/api/vendors/%d/?locations=delete", v.ID)}
		})
	})
}

func TestVendorHandlerConflicts(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	// A vendor's Keycloak user cannot take over the address of another Keycloak user
	licenseID := "conflict-vendor"
	_ = keycloak.KeycloakClient.DeleteUser(licenseID + "@example.com")
	vendorID := createTestVendor(t, licenseID)
	t.Cleanup(func() { _ = keycloak.KeycloakClient.DeleteUser(licenseID + "@example.com") })
	body := fmt.Sprintf(`{"licenseID":%q,"firstName":"A","lastName":"B","email":%q}`, licenseID, adminUserEmail)
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/", body, 409, adminUserToken)

	// Without an own address and without a license ID no internal address can be made
	utils.TestRequestStrWithAuth(t, r, "POST", "/api/vendors/", `{"licenseID":"","firstName":"A","lastName":"B","email":""}`, 400, adminUserToken)

	// A vendor with an own address but no Keycloak user cannot get a mail
	origSender := config.Config.SMTPSenderAddress
	t.Cleanup(func() { config.Config.SMTPSenderAddress = origSender })
	config.Config.SMTPSenderAddress = "noreply@example.com"
	id, err := database.Db.CreateVendor(database.Vendor{FirstName: "Own", LastName: "Mail", Email: "own-mail-no-keycloak@example.com", HasOwnEmail: true, LicenseID: null.StringFrom("own-mail-1")})
	require.NoError(t, err)
	utils.TestRequestWithAuth(t, r, "POST", fmt.Sprintf("/api/vendors/%d/password-reset-email/", id), nil, 502, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/999999/verify-email/", nil, 404, adminUserToken)
}

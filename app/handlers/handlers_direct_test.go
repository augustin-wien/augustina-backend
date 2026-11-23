package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// callHandler calls h directly, with the given chi URL parameters and headers,
// as if the auth middleware had already run
func callHandler(h http.HandlerFunc, method, body string, params, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func TestOdooVendorHandlers(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID := "odoo-direct-1"
	_, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Odoo", LastName: "Vendor", Email: "odoo-direct-1@example.com",
		LicenseID: null.StringFrom(licenseID), IsBlocked: true, BlockedNote: "bleibt",
	})
	require.NoError(t, err)

	res := callHandler(GetVendorByLicenseID, "GET", "", map[string]string{"licenseID": licenseID}, nil)
	require.Equal(t, http.StatusOK, res.Code)
	var vendor database.Vendor
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &vendor))
	require.Equal(t, "Odoo", vendor.FirstName)

	require.Equal(t, http.StatusBadRequest, callHandler(GetVendorByLicenseID, "GET", "", map[string]string{"licenseID": ""}, nil).Code)
	require.Equal(t, http.StatusBadRequest, callHandler(GetVendorByLicenseID, "GET", "", map[string]string{"licenseID": "unknown"}, nil).Code)

	params := map[string]string{"licenseID": licenseID}
	require.Equal(t, http.StatusBadRequest, callHandler(UpdateVendorByLicenseID, "PUT", "{}", map[string]string{"licenseID": ""}, nil).Code)
	require.Equal(t, http.StatusBadRequest, callHandler(UpdateVendorByLicenseID, "PUT", "{}", map[string]string{"licenseID": "unknown"}, nil).Code)
	require.Equal(t, http.StatusBadRequest, callHandler(UpdateVendorByLicenseID, "PUT", "{not json", params, nil).Code)

	// Odoo does not know about blocking, so the block set in the backoffice stays
	res = callHandler(UpdateVendorByLicenseID, "PUT", `{"licenseID":"odoo-direct-1","firstName":"Neu","lastName":"Name","email":"odoo-direct-1@example.com"}`, params, nil)
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	vendor, err = database.Db.GetVendorByLicenseID(licenseID)
	require.NoError(t, err)
	require.Equal(t, "Neu", vendor.FirstName)
	require.True(t, vendor.IsBlocked)
	require.Equal(t, "bleibt", vendor.BlockedNote)
}

func TestSelfServiceHandlers(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	// Vendor overview
	vendorEmail := "self-service-vendor@example.com"
	_, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Self", LastName: "Vendor", Email: vendorEmail, LicenseID: null.StringFrom("self-service-1"),
	})
	require.NoError(t, err)

	require.Equal(t, http.StatusBadRequest, callHandler(GetVendorOverview, "GET", "", nil, nil).Code)
	require.Equal(t, http.StatusBadRequest, callHandler(GetVendorOverview, "GET", "", nil, map[string]string{"X-Auth-User-Email": "nobody@example.com"}).Code)
	res := callHandler(GetVendorOverview, "GET", "", nil, map[string]string{"X-Auth-User-Email": vendorEmail})
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	var overview VendorOverview
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &overview))
	require.Equal(t, "self-service-1", overview.LicenseID)

	// Customer payments and abonements
	customer, err := database.Db.CreateCustomer(&database.Customer{KeycloakID: "kc-self-service", Email: "self-service-customer@example.com"})
	require.NoError(t, err)
	_, err = database.Db.CreateAbonement(&database.Abonement{
		CustomerID: customer.ID, FromDate: time.Now().AddDate(0, -1, 0), ToDate: time.Now().AddDate(0, 1, 0), Status: "active",
	})
	require.NoError(t, err)

	require.Equal(t, http.StatusUnauthorized, callHandler(ListMyPayments, "GET", "", nil, nil).Code)
	require.Equal(t, http.StatusForbidden, callHandler(ListMyPayments, "GET", "", nil, map[string]string{"X-Auth-User-Email": customer.Email}).Code)
	res = callHandler(ListMyPayments, "GET", "", nil, map[string]string{"X-Auth-User-Email": customer.Email, "X-Auth-User-Email-Verified": "true"})
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	require.Equal(t, http.StatusUnauthorized, callHandler(ListMyAbonements, "GET", "", nil, nil).Code)
	require.Equal(t, http.StatusNotFound, callHandler(ListMyAbonements, "GET", "", nil, map[string]string{"X-Auth-User": "kc-unknown"}).Code)
	res = callHandler(ListMyAbonements, "GET", "", nil, map[string]string{"X-Auth-User": "kc-self-service"})
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())
	var abos []database.Abonement
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &abos))
	require.Len(t, abos, 1)
}

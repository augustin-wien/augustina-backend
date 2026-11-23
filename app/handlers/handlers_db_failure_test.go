package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// withBrokenDatabase replaces the global database with one whose connection is
// closed, so every query fails, and restores it when the test ends.
func withBrokenDatabase(t *testing.T) {
	t.Helper()
	pdb, err := sql.Open("postgres", "postgres://invalid@127.0.0.1:1/invalid?sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, pdb.Close())

	orig := database.Db
	database.Db = database.Database{
		EntClient: ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, pdb))),
		DB:        pdb,
	}
	t.Cleanup(func() { database.Db = orig })
}

// TestHandlersReportDatabaseErrors checks that the handlers answer with an
// error status, instead of panicking or reporting success, when the database
// fails.
func TestHandlersReportDatabaseErrors(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	withBrokenDatabase(t)

	// Unsynced orders are only looked up while the Odoo integration is on
	origOdooURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = "http://127.0.0.1:1"
	t.Cleanup(func() { config.Config.OdooWebhookURL = origOdooURL })

	jsonVendor := `{"licenseID":"dbfail-1","firstName":"a","lastName":"b","email":"dbfail-1@example.com"}`
	jsonItem := `{"Name":"x","Description":"y","Price":100,"Type":"normal_item"}`
	jsonPayment := `{"Sender":1,"Receiver":2,"Amount":1}`
	jsonAbo := `{"customer_id":1,"from_date":"2026-01-01","to_date":"2026-12-31","status":"active"}`
	jsonCustomer := `{"email":"dbfail@example.com","firstname":"a","lastname":"b"}`
	jsonLocation := `{"name":"Spar","address":"Street 1","zip":"1010"}`
	jsonCampaign := `{"name":"c","active":true}`
	jsonOrder := `{"entries":[{"item":1,"quantity":1}],"vendorLicenseID":"dbfail-1"}`

	cases := []struct {
		method, path, body string
		auth               bool
	}{
		// Public
		{"GET", "/api/hello/", "", false},
		{"GET", "/api/auth/hello/", "", true},
		{"GET", "/api/items/", "", false},
		{"GET", "/api/settings/", "", false},
		{"GET", "/api/vendors/check/dbfail-1/", "", false},
		{"POST", "/api/orders/", jsonOrder, false},
		{"GET", "/api/orders/verify/?s=1234567890&t=tx", "", false},
		{"GET", "/api/pdf/link/validate/", "", false},
		{"GET", "/api/pdf/link/", "", false},
		{"GET", "/api/campaigns/active/", "", false},
		{"POST", "/api/campaigns/1/view/", "", false},
		{"POST", "/api/campaigns/1/click/", "", false},
		// Settings
		{"GET", "/api/settings/admin/", "", true},
		// Vendors
		{"GET", "/api/vendors/", "", true},
		{"POST", "/api/vendors/", jsonVendor, true},
		{"GET", "/api/vendors/1/", "", true},
		{"PUT", "/api/vendors/1/", jsonVendor, true},
		{"DELETE", "/api/vendors/1/", "", true},
		{"POST", "/api/vendors/recalculate-balances/", "", true},
		{"GET", "/api/vendors/statistics/", "", true},
		{"GET", "/api/vendors/1/locations/", "", true},
		{"POST", "/api/vendors/1/locations/", jsonLocation, true},
		{"PATCH", "/api/vendors/1/locations/1/", jsonLocation, true},
		{"DELETE", "/api/vendors/1/locations/1/", "", true},
		{"GET", "/api/vendors/1/comments/", "", true},
		{"POST", "/api/vendors/1/comments/", `{"comment":"x"}`, true},
		{"PATCH", "/api/vendors/1/comments/1/", `{"id":1,"comment":"x"}`, true},
		{"DELETE", "/api/vendors/1/comments/1/", "", true},
		{"POST", "/api/vendors/1/password-reset-email/", "", true},
		{"POST", "/api/vendors/1/verify-email/", "", true},
		{"POST", "/api/vendors/dbfail-1/pos-order/", `{"entries":[{"item":1,"quantity":1}]}`, true},
		{"GET", "/api/vendors/dbfail-1/pos-orders/", "", true},
		{"POST", "/api/vendors/dbfail-1/id-scan/", "", true},
		{"POST", "/api/vendors/dbfail-1/nextcloud-folder/", "", true},
		// Items
		{"GET", "/api/items/backoffice/", "", true},
		{"GET", "/api/items/backoffice/?includeArchived=true", "", true},
		{"GET", "/api/items/licensegroups/", "", true},
		{"GET", "/api/items/archived/", "", true},
		{"POST", "/api/items/", jsonItem, true},
		{"PUT", "/api/items/1/", jsonItem, true},
		{"DELETE", "/api/items/1/", "", true},
		{"POST", "/api/items/1/restore/", "", true},
		{"POST", "/api/items/1/notify-abonements/", "", true},
		{"GET", "/api/items/1/pdf/", "", true},
		// Campaigns
		{"GET", "/api/campaigns/", "", true},
		{"POST", "/api/campaigns/", jsonCampaign, true},
		{"PUT", "/api/campaigns/1/", jsonCampaign, true},
		{"DELETE", "/api/campaigns/1/", "", true},
		// Reports
		{"GET", "/api/reports/monthly/2026-01/", "", true},
		{"GET", "/api/reports/monthly/2026-01/csv/", "", true},
		{"GET", "/api/reports/monthly/2026-01/pdf/", "", true},
		{"POST", "/api/reports/monthly/2026-01/send/", "", true},
		// Orders
		{"GET", "/api/orders/unverified/", "", true},
		{"GET", "/api/orders/verified/", "", true},
		{"GET", "/api/orders/statistics/", "", true},
		{"GET", "/api/orders/unsynced/", "", true},
		{"GET", "/api/orders/unverified/code/1234567890/verify/", "", true},
		{"POST", "/api/orders/unverified/code/1234567890/transactionID/", `{"transactionID":"tx"}`, true},
		{"POST", "/api/orders/resend/1/", "", true},
		{"POST", "/api/orders/1/resend-mail/", "", true},
		// Customers
		{"GET", "/api/customers/", "", true},
		{"POST", "/api/customers/", jsonCustomer, true},
		{"GET", "/api/customers/1/", "", true},
		{"PUT", "/api/customers/1/", jsonCustomer, true},
		{"DELETE", "/api/customers/1/", "", true},
		{"GET", "/api/customers/1/abonements/", "", true},
		{"POST", "/api/customers/1/password-reset-email/", "", true},
		{"POST", "/api/customers/1/verify-email/", "", true},
		// Abonements
		{"GET", "/api/abonements/", "", true},
		{"POST", "/api/abonements/", jsonAbo, true},
		{"GET", "/api/abonements/active/", "", true},
		{"GET", "/api/abonements/by-date/?date=2026-01-01", "", true},
		{"GET", "/api/abonements/1/", "", true},
		{"PUT", "/api/abonements/1/", jsonAbo, true},
		{"DELETE", "/api/abonements/1/", "", true},
		// POS, external payments, payments
		{"GET", "/api/pos-orders/", "", true},
		{"GET", "/api/external-payments/", "", true},
		{"GET", "/api/payments/", "", true},
		{"POST", "/api/payments/", jsonPayment, true},
		{"POST", "/api/payments/batch/", `{"Payments":[` + jsonPayment + `]}`, true},
		{"GET", "/api/payments/forpayout/", "", true},
		{"GET", "/api/payments/forpayout/?vendor=dbfail-1", "", true},
		{"GET", "/api/payments/statistics/", "", true},
		{"POST", "/api/payments/payout/", `{"VendorLicenseID":"dbfail-1"}`, true},
		// Locations and map
		{"GET", "/api/locations/", "", true},
		{"POST", "/api/locations/", jsonLocation, true},
		{"PATCH", "/api/locations/1/", jsonLocation, true},
		{"DELETE", "/api/locations/1/", "", true},
		{"GET", "/api/map/", "", true},
		// Mail templates
		{"GET", "/api/mail-templates/", "", true},
		{"GET", "/api/mail-templates/x/", "", true},
		{"POST", "/api/mail-templates/", `{"name":"x","subject":"s","body":"b"}`, true},
		{"POST", "/api/mail-templates/x/send/", `{"to":["a@example.com"]}`, true},
		{"DELETE", "/api/mail-templates/x/", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.auth {
				req.Header.Set("Authorization", "Bearer "+adminUserToken.AccessToken)
			}
			var rr *httptest.ResponseRecorder
			require.NotPanics(t, func() { rr = utils.SubmitRequest(req, r) })
			require.GreaterOrEqual(t, rr.Code, http.StatusBadRequest, "body: %s", rr.Body.String())
			require.NotEqual(t, "404 page not found\n", rr.Body.String(), "route not registered")
			require.NotEqual(t, http.StatusMethodNotAllowed, rr.Code)
			require.NotEqual(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// TestHandlersRejectInvalidInput checks that malformed IDs and bodies are
// answered with a client error before anything is changed.
func TestHandlersRejectInvalidInput(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	const badJSON = "{not json"
	cases := []struct {
		method, path, body string
	}{
		// Malformed IDs
		{"GET", "/api/abonements/abc/", ""},
		{"PUT", "/api/abonements/abc/", "{}"},
		{"DELETE", "/api/abonements/abc/", ""},
		{"GET", "/api/customers/abc/", ""},
		{"PUT", "/api/customers/abc/", "{}"},
		{"DELETE", "/api/customers/abc/", ""},
		{"GET", "/api/customers/abc/abonements/", ""},
		{"POST", "/api/customers/abc/password-reset-email/", ""},
		{"POST", "/api/customers/abc/verify-email/", ""},
		{"PUT", "/api/campaigns/abc/", "{}"},
		{"DELETE", "/api/campaigns/abc/", ""},
		{"POST", "/api/campaigns/abc/view/", ""},
		{"GET", "/api/vendors/abc/locations/", ""},
		{"POST", "/api/vendors/abc/locations/", "{}"},
		{"PATCH", "/api/vendors/1/locations/abc/", "{}"},
		{"PATCH", "/api/vendors/abc/locations/1/", "{}"},
		{"DELETE", "/api/vendors/1/locations/abc/", ""},
		{"DELETE", "/api/vendors/abc/locations/1/", ""},
		{"POST", "/api/vendors/abc/password-reset-email/", ""},
		{"PATCH", "/api/locations/abc/", "{}"},
		{"DELETE", "/api/locations/abc/", ""},
		{"PUT", "/api/items/abc/", ""},
		{"POST", "/api/orders/abc/resend-mail/", ""},
		{"GET", "/api/reports/monthly/2026-13/", ""},
		{"GET", "/api/reports/monthly/abc/csv/", ""},
		{"GET", "/api/reports/monthly/abc/pdf/", ""},
		{"POST", "/api/reports/monthly/abc/send/", ""},
		// Malformed bodies
		{"POST", "/api/abonements/", badJSON},
		{"PUT", "/api/abonements/1/", badJSON},
		{"POST", "/api/customers/", badJSON},
		{"PUT", "/api/customers/1/", badJSON},
		{"POST", "/api/campaigns/", badJSON},
		{"PUT", "/api/campaigns/1/", badJSON},
		{"POST", "/api/vendors/1/locations/", badJSON},
		{"PATCH", "/api/vendors/1/locations/1/", badJSON},
		{"POST", "/api/locations/", badJSON},
		{"PATCH", "/api/locations/1/", badJSON},
		{"POST", "/api/payments/payout/", badJSON},
		{"POST", "/api/vendors/x/pos-order/", badJSON},
		{"POST", "/api/mail-templates/x/send/", badJSON},
		{"POST", "/api/mail-templates/x/send/", `{"to":[]}`},
		{"POST", "/api/settings/wordpress-invite/test/", badJSON},
		{"PUT", "/api/items/1/", badJSON},
		// Required query parameters
		{"GET", "/api/payments/statistics/?from=x", ""},
		{"GET", "/api/vendors/statistics/?from=x", ""},
		{"GET", "/api/external-payments/?from=x", ""},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path+" "+tc.body, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+adminUserToken.AccessToken)
			rr := utils.SubmitRequest(req, r)
			require.GreaterOrEqual(t, rr.Code, http.StatusBadRequest, rr.Body.String())
			require.Less(t, rr.Code, http.StatusInternalServerError, rr.Body.String())
			require.NotEqual(t, "404 page not found\n", rr.Body.String(), "route not registered")
			require.NotEqual(t, http.StatusMethodNotAllowed, rr.Code)
			require.NotEqual(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

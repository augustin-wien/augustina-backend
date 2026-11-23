package handlers

import (
	"encoding/json"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// TestInvalidQueryParametersAnswerOnce checks that an invalid filter is
// answered with exactly one error response.
func TestInvalidQueryParametersAnswerOnce(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	for _, path := range []string{
		"/api/payments/?from=yesterday",
		"/api/payments/?to=tomorrow",
		"/api/payments/?payouts=perhaps",
		"/api/payments/?sales=perhaps",
		"/api/payments/forpayout/?from=yesterday",
		"/api/payments/forpayout/?to=tomorrow",
		"/api/payments/forpayout/?vendor=unknown-license",
		"/api/orders/statistics/?from=yesterday",
		"/api/orders/statistics/?to=tomorrow",
	} {
		t.Run(path, func(t *testing.T) {
			res := utils.TestRequestWithAuth(t, r, "GET", path, nil, 400, adminUserToken)
			var body map[string]any
			require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body), "response must be a single JSON document: %s", res.Body.String())
		})
	}

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/orders/statistics/?from=2026-01-01T00:00:00Z&to=2026-12-31T00:00:00Z", nil, 200, adminUserToken)
	var stats database.OrderStatistics
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &stats))

	utils.TestRequestWithAuth(t, r, "GET", "/api/payments/forpayout/?from=2026-01-01T00:00:00Z&to=2026-12-31T00:00:00Z", nil, 200, adminUserToken)
}

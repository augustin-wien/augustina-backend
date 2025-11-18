package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/notifications"
	"github.com/augustin-wien/augustina-backend/utils"
)

// TestRouter_LocationsEndpoints performs basic integration tests for location endpoints.
// This test requires a test database configured via env vars used by Database.InitEmptyTestDb().
// If a test DB isn't available the test will be skipped.
func TestRouter_LocationsEndpoints(t *testing.T) {
	// Disable external notifications during tests to avoid network calls
	// and make sure demo data is not created.
	_ = os.Setenv("NOTIFICATIONS_MATRIX_ENABLED", "false")
	_ = os.Setenv("NOTIFICATIONS_EMAIL_ENABLED", "false")
	_ = os.Setenv("CREATE_DEMO_DATA", "false")

	// Disable notifications service if already initialized by TestMain to prevent network calls.
	if notifications.NotificationsClient.Client != nil {
		notifications.NotificationsClient.Client.Disabled = true
		notifications.NotificationsClient.SentryEnabled = false
	}

	// If the global DB is not initialized by TestMain, initialize an empty test DB; otherwise reuse it.
	if database.Db.Dbpool == nil {
		if err := database.Db.InitEmptyTestDb(); err != nil {
			t.Skipf("skipping test - no test DB available: %v", err)
			return
		}
		defer database.Db.CloseDbPool()
	}

	// Ensure the working_times table exists in the test DB (migration may not have been applied).
	ctx := context.Background()
	_, err := database.Db.Dbpool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS working_times (
			id SERIAL PRIMARY KEY,
			day TEXT DEFAULT 'monday',
			open_time TEXT,
			close_time TEXT,
			closed BOOLEAN DEFAULT false,
			location_working_times INTEGER REFERENCES locations(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("failed to ensure working_times table exists: %v", err)
	}
	// Ensure the FK has ON DELETE CASCADE in case the table already existed without it.
	_, _ = database.Db.Dbpool.Exec(ctx, `ALTER TABLE working_times DROP CONSTRAINT IF EXISTS working_times_location_working_times_fkey;`)
	_, err = database.Db.Dbpool.Exec(ctx, `ALTER TABLE working_times ADD CONSTRAINT working_times_location_working_times_fkey FOREIGN KEY (location_working_times) REFERENCES locations(id) ON DELETE CASCADE;`)
	if err != nil {
		t.Fatalf("failed to ensure working_times FK cascade: %v", err)
	}

	// Use the router initialized by TestMain (package-level `r`) and helpers.

	// Use existing TestMain-initialized router `r` and admin token `adminUserToken`.
	// Create vendor using helper from handlers_test.go
	vendorIDStr := createTestVendor(t, "testlicense-for-router")

	// 1) Create a location via the API (uses admin auth in Test helpers)
	locPayload := `{"name":"Test Location","address":"123 Test St","longitude":1.23,"latitude":4.56,"zip":"1000","working_time":"G"}`
	res := utils.TestRequestStrWithAuth(t, r, "POST", "/api/vendors/"+vendorIDStr+"/locations/", locPayload, 200, adminUserToken)
	_ = res

	// 2) List locations and assert one exists
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorIDStr+"/locations/", nil, 200, adminUserToken)
	var locations []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &locations); err != nil {
		t.Fatalf("failed to decode locations response: %v", err)
	}
	if len(locations) == 0 {
		t.Fatalf("expected at least one location returned")
	}
	idf, ok := locations[0]["id"].(float64)
	if !ok {
		t.Fatalf("unexpected id type in response")
	}
	locID := int(idf)

	// 3) Update the location (PATCH)
	updatePayload := fmt.Sprintf(`{"id":%d, "name":"Updated Location", "address":"123 Test St", "longitude":1.23, "latitude":4.56, "zip":"1000"}`, locID)
	utils.TestRequestStrWithAuth(t, r, "PATCH", "/api/vendors/"+vendorIDStr+"/locations/"+fmt.Sprintf("%d/", locID), updatePayload, 200, adminUserToken)

	// 4) Delete the location
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+vendorIDStr+"/locations/"+fmt.Sprintf("%d/", locID), nil, 200, adminUserToken)

	// 5) Ensure no locations remain
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorIDStr+"/locations/", nil, 200, adminUserToken)
	locations = nil
	if err := json.Unmarshal(res.Body.Bytes(), &locations); err != nil {
		t.Fatalf("failed to decode locations response: %v", err)
	}
	if len(locations) != 0 {
		t.Fatalf("expected 0 locations after delete, got %d", len(locations))
	}
}

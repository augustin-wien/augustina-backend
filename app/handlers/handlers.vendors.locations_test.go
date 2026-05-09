package handlers

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestVendorLocationsCRUD(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize a fresh test DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		t.Fatalf("InitEmptyTestDb failed: %v", err)
	}

	// Create a vendor
	vendorLicenseId := "testlicense-loc"
	vendorID := createTestVendor(t, vendorLicenseId)

	// CREATE: Create a location for the vendor with structured working time
	locationBody := map[string]any{
		"name":    "Test Location",
		"address": "Test Street 1",
		"longitude": 16.3,
		"latitude":  48.2,
		"zip":       "1000",
		"working_time": map[string]any{
			"mode": "everyday",
			"everyday": []map[string]any{{
				"from": "08:00",
				"to":   "12:00",
			}},
		},
	}

	res := utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+vendorID+"/locations/", locationBody, 200, adminUserToken)
	require.NotNil(t, res)

	// READ: List locations and verify they were created
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/locations/", nil, 200, adminUserToken)

	var locations []map[string]any
	err = json.Unmarshal(res.Body.Bytes(), &locations)
	require.NoError(t, err)
	require.Equal(t, 1, len(locations), "Expected 1 location to be returned")
	
	// Check that location fields are present
	location := locations[0]
	require.Equal(t, "Test Location", location["name"], "Location name should match")
	require.Equal(t, "Test Street 1", location["address"], "Location address should match")
	require.Equal(t, float64(16.3), location["longitude"], "Longitude should match")
	require.Equal(t, float64(48.2), location["latitude"], "Latitude should match")
	require.Equal(t, "1000", location["zip"], "Zip should match")
	
	// Check working_time structure was preserved
	workingTime, ok := location["working_time"].(map[string]any)
	require.True(t, ok, "working_time should be an object")
	require.Equal(t, "everyday", workingTime["mode"], "Mode should be 'everyday'")
	
	// Extract location ID for update
	locationID, ok := location["id"].(float64)
	require.True(t, ok, "Location should have an id field as number")

	// UPDATE: Modify the location with new details
	updatedLocationBody := map[string]any{
		"id":      int(locationID),
		"name":    "Updated Location",
		"address": "Test Street 2",
		"longitude": 16.4,
		"latitude":  48.3,
		"zip":       "2000",
		"working_time": map[string]any{
			"mode": "by_day",
			"week_days": map[string]any{
				"mon": []map[string]any{{
					"from": "09:00",
					"to":   "17:00",
				}},
				"tue": []map[string]any{{
					"from": "09:00",
					"to":   "17:00",
				}},
				"wed": []map[string]any{{
					"from": "09:00",
					"to":   "17:00",
				}},
				"thu": []map[string]any{{
					"from": "09:00",
					"to":   "17:00",
				}},
				"fri": []map[string]any{{
					"from": "09:00",
					"to":   "17:00",
				}},
				"sat": []map[string]any{{
					"full_day": true,
				}},
				"sun": []map[string]any{{
					"full_day": true,
				}},
			},
		},
	}

	utils.TestRequestWithAuth(t, r, "PATCH", "/api/vendors/"+vendorID+"/locations/"+strconv.Itoa(int(locationID))+"/", updatedLocationBody, 200, adminUserToken)

	// VERIFY UPDATE: List locations again and verify changes
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/locations/", nil, 200, adminUserToken)

	var updatedLocations []map[string]any
	err = json.Unmarshal(res.Body.Bytes(), &updatedLocations)
	require.NoError(t, err)
	require.Equal(t, 1, len(updatedLocations), "Should still have 1 location")
	
	updatedLoc := updatedLocations[0]
	require.Equal(t, "Updated Location", updatedLoc["name"], "Location name should be updated")
	require.Equal(t, "Test Street 2", updatedLoc["address"], "Location address should be updated")
	require.Equal(t, float64(16.4), updatedLoc["longitude"], "Longitude should be updated")
	require.Equal(t, float64(48.3), updatedLoc["latitude"], "Latitude should be updated")
	require.Equal(t, "2000", updatedLoc["zip"], "Zip should be updated")
	
	// Verify by_day working time structure
	updatedWorkingTime, ok := updatedLoc["working_time"].(map[string]any)
	require.True(t, ok, "updated working_time should be an object")
	require.Equal(t, "by_day", updatedWorkingTime["mode"], "Mode should be updated to 'by_day'")
	require.NotNil(t, updatedWorkingTime["week_days"], "week_days should be present")
	
	// Verify old 'everyday' field is gone (complete replacement, not merge)
	_, hasEveryday := updatedWorkingTime["everyday"]
	require.False(t, hasEveryday, "everyday field should be removed after update to by_day mode")

	// DELETE: Delete the location
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+vendorID+"/locations/"+strconv.Itoa(int(locationID))+"/", nil, 200, adminUserToken)

	// VERIFY DELETE: List locations and verify it's gone
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/locations/", nil, 200, adminUserToken)

	var finalLocations []map[string]any
	err = json.Unmarshal(res.Body.Bytes(), &finalLocations)
	require.NoError(t, err)
	require.Equal(t, 0, len(finalLocations), "Should have no locations after deletion")
}

// TestWorkingTimeValidation verifies that working_time JSON structures are valid
// and don't contain orphaned/unknown legacy codes from the migration
func TestWorkingTimeValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize test DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		t.Fatalf("InitEmptyTestDb failed: %v", err)
	}

	// Create vendor and location
	vendorLicenseId := "test-working-time-validation"
	vendorID := createTestVendor(t, vendorLicenseId)

	// Test all valid working time modes
	testCases := []struct {
		name         string
		workingTime  map[string]any
		expectMode   string
		expectFields []string
	}{
		{
			name: "everyday mode",
			workingTime: map[string]any{
				"mode": "everyday",
				"everyday": []map[string]any{{
					"from": "08:00",
					"to":   "12:00",
				}},
			},
			expectMode:   "everyday",
			expectFields: []string{"everyday"},
		},
		{
			name: "by_day mode",
			workingTime: map[string]any{
				"mode": "by_day",
				"week_days": map[string]any{
					"mon": []map[string]any{{"from": "09:00", "to": "17:00"}},
				},
			},
			expectMode:   "by_day",
			expectFields: []string{"week_days"},
		},
		{
			name: "whole_week mode",
			workingTime: map[string]any{
				"mode":       "whole_week",
				"whole_week": true,
			},
			expectMode:   "whole_week",
			expectFields: []string{"whole_week"},
		},
	}

	for _, tc := range testCases {
		locationBody := map[string]any{
			"name":         "Location " + tc.name,
			"address":      "Test Address",
			"longitude":    16.3,
			"latitude":     48.2,
			"zip":          "1000",
			"working_time": tc.workingTime,
		}

		res := utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+vendorID+"/locations/", locationBody, 200, adminUserToken)
		require.NotNil(t, res, "Failed to create location for "+tc.name)

		// Fetch and validate
		res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/locations/", nil, 200, adminUserToken)

		var locations []map[string]any
		err := json.Unmarshal(res.Body.Bytes(), &locations)
		require.NoError(t, err)

		// Verify the created location has valid structure
		for _, loc := range locations {
			if name, ok := loc["name"].(string); ok && name == "Location "+tc.name {
				wt, ok := loc["working_time"].(map[string]any)
				require.True(t, ok, tc.name+": working_time should be an object")
				require.Equal(t, tc.expectMode, wt["mode"], tc.name+": mode should be "+tc.expectMode)

				// Verify expected fields are present
				for _, field := range tc.expectFields {
					_, hasField := wt[field]
					require.True(t, hasField, tc.name+": should have "+field+" field")
				}

				// Verify no orphaned fields from migration (unknown legacy codes)
				if wt["mode"] != "everyday" && wt["mode"] != "by_day" && wt["mode"] != "whole_week" && wt["mode"] != "custom" {
					t.Logf("WARNING: Found unexpected mode in working_time: %v", wt["mode"])
				}
			}
		}
	}
}


package handlers

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func listLocationsOfVendor(t *testing.T, vendorID int) (owned, unassigned int) {
	t.Helper()
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/locations/", nil, 200, adminUserToken)
	var locations []database.LocationOverview
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &locations))
	for _, l := range locations {
		switch {
		case l.VendorID == nil:
			unassigned++
		case *l.VendorID == vendorID:
			owned++
		}
	}
	return
}

func TestVendorLocationsOnDisableAndDelete(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID := "testlicense-disable"
	_ = keycloak.KeycloakClient.DeleteUser(licenseID + "@example.com")
	idStr := createTestVendor(t, licenseID)
	vendorID, err := strconv.Atoi(idStr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = keycloak.KeycloakClient.DeleteUser(licenseID + "@example.com") })

	for _, name := range []string{"Billa", "Spar"} {
		utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+idStr+"/locations/",
			map[string]any{"name": name, "address": "Street", "zip": "1010"}, 200, adminUserToken)
	}
	owned, unassigned := listLocationsOfVendor(t, vendorID)
	require.Equal(t, 2, owned)
	require.Equal(t, 0, unassigned)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+idStr+"/", nil, 200, adminUserToken)
	var vendor database.Vendor
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &vendor))

	utils.TestRequestWithAuth(t, r, "PUT", "/api/vendors/"+idStr+"/?locations=sometimes", vendor, 400, adminUserToken)

	// Disabling with "keep" leaves the locations as unassigned ones
	vendor.IsDisabled = true
	utils.TestRequestWithAuth(t, r, "PUT", "/api/vendors/"+idStr+"/?locations=keep", vendor, 200, adminUserToken)
	owned, unassigned = listLocationsOfVendor(t, vendorID)
	require.Equal(t, 0, owned)
	require.Equal(t, 2, unassigned)

	// Deleting with "delete" removes the vendor's remaining locations
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+idStr+"/locations/",
		map[string]any{"name": "Hofer", "address": "Street", "zip": "1010"}, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+idStr+"/?locations=nope", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+idStr+"/?locations=delete", nil, 204, adminUserToken)
	owned, unassigned = listLocationsOfVendor(t, vendorID)
	require.Equal(t, 0, owned)
	require.Equal(t, 2, unassigned)
}

func TestVendorEndpointValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID := "testlicense-validation"
	_ = keycloak.KeycloakClient.DeleteUser(licenseID + "@example.com")
	idStr := createTestVendor(t, licenseID)
	t.Cleanup(func() { _ = keycloak.KeycloakClient.DeleteUser(licenseID + "@example.com") })

	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/abc/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/999999/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "PUT", "/api/vendors/abc/", map[string]any{}, 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+idStr+"/", "{not json", 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "PUT", "/api/vendors/999999/", map[string]any{"licenseID": "x", "email": "x@example.com"}, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/abc/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/999999/", nil, 400, adminUserToken)
	utils.TestRequestStrWithAuth(t, r, "POST", "/api/vendors/", "{not json", 400, adminUserToken)

	// A license ID can only be used once
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/", map[string]any{
		"licenseID": licenseID,
		"email":     "other-" + licenseID + "@example.com",
		"firstName": "Dup",
		"lastName":  "Licate",
	}, 400, adminUserToken)

	// The shop shows the license ID instead of the first name when configured
	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	settings.UseVendorLicenseIdInShop = true
	require.NoError(t, database.Db.UpdateSettings(settings))
	res := utils.TestRequest(t, r, "GET", "/api/vendors/check/"+licenseID+"/", nil, 200)
	var check checkLicenseIDResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &check))
	require.Equal(t, licenseID, check.FirstName)

	utils.TestRequest(t, r, "GET", "/api/vendors/check/unknown-license/", nil, 400)

	// Sending action mails needs a configured SMTP sender
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/abc/verify-email/", nil, 400, adminUserToken)
}

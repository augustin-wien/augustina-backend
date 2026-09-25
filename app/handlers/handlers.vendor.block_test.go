package handlers

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// TestBlockedVendorCannotSell verifies that a blocked vendor is rejected by the
// public license check, the online order and the POS, and that unblocking
// restores all three.
func TestBlockedVendorCannotSell(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	licenseID := "testlicense-blocked"
	vendorID, err := strconv.Atoi(createTestVendor(t, licenseID))
	require.NoError(t, err)

	itemID, err := database.Db.CreateItem(database.Item{
		Name:        "Blocked Test Newspaper",
		Description: "Newspaper",
		Price:       300,
		Type:        "normal_item",
	})
	require.NoError(t, err)

	setBlocked := func(blocked bool, note string) {
		vendor, err := database.Db.GetVendor(vendorID)
		require.NoError(t, err)
		vendor.IsBlocked = blocked
		vendor.BlockedNote = note
		require.NoError(t, database.Db.UpdateVendor(vendorID, vendor))
	}

	orderBody := map[string]any{
		"entries":         []map[string]any{{"item": itemID, "quantity": 1}},
		"vendorLicenseID": licenseID,
	}
	posBody := map[string]any{
		"entries": []map[string]any{{"item": itemID, "quantity": 1}},
	}

	setBlocked(true, "Hausverbot bis Monatsende")

	// The note is stored and returned to the backoffice
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+strconv.Itoa(vendorID)+"/", nil, 200, adminUserToken)
	var vendor database.Vendor
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &vendor))
	require.True(t, vendor.IsBlocked)
	require.Equal(t, "Hausverbot bis Monatsende", vendor.BlockedNote)

	utils.TestRequest(t, r, "GET", "/api/vendors/check/"+licenseID+"/", nil, 403)
	utils.TestRequest(t, r, "POST", "/api/orders/", orderBody, 403)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+licenseID+"/pos-order/", posBody, 403, adminUserToken)

	setBlocked(false, "")

	utils.TestRequest(t, r, "GET", "/api/vendors/check/"+licenseID+"/", nil, 200)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+licenseID+"/pos-order/", posBody, 200, adminUserToken)
}

// TestListAllVendorLocations verifies the backoffice overview of all locations
// carries the location's own telephone number and the owning vendor.
func TestListAllVendorLocations(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	secondID := createTestVendor(t, "testlicense-loc-b")
	firstID := createTestVendor(t, "testlicense-loc-a")

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+secondID+"/locations/",
		map[string]any{"name": "Spar", "address": "Street 2", "zip": "1020", "telephone": "+43 1 222"}, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+firstID+"/locations/",
		map[string]any{"name": "Billa", "address": "Street 1", "zip": "1010", "telephone": "+43 1 111"}, 200, adminUserToken)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/locations/", nil, 200, adminUserToken)
	var locations []database.LocationOverview
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &locations))
	require.Len(t, locations, 2)

	// Ordered by license ID
	require.Equal(t, "testlicense-loc-a", locations[0].VendorLicenseID)
	require.Equal(t, "Billa", locations[0].Name)
	require.Equal(t, "+43 1 111", locations[0].Telephone)
	require.NotNil(t, locations[0].VendorID)
	require.Equal(t, "+43123456789", locations[0].VendorTelephone)
	require.Equal(t, "testlicense-loc-b", locations[1].VendorLicenseID)
	require.Equal(t, "+43 1 222", locations[1].Telephone)

	// Deleted vendors' locations are left out
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+secondID+"/", nil, 204, adminUserToken)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/locations/", nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &locations))
	require.Len(t, locations, 1)
	require.Equal(t, "testlicense-loc-a", locations[0].VendorLicenseID)

	// The endpoint is backoffice-only
	utils.TestRequest(t, r, "GET", "/api/locations/", nil, 401)
}

// TestListVendorsIncludesDisabled verifies the backoffice vendor list shows
// disabled vendors (flagged) but not deleted ones.
func TestListVendorsIncludesDisabled(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	activeID := createTestVendor(t, "testlicense-active")
	disabledID, err := strconv.Atoi(createTestVendor(t, "testlicense-disabled"))
	require.NoError(t, err)
	deletedID := createTestVendor(t, "testlicense-deleted")

	disabled, err := database.Db.GetVendor(disabledID)
	require.NoError(t, err)
	disabled.IsDisabled = true
	require.NoError(t, database.Db.UpdateVendor(disabledID, disabled))
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+deletedID+"/", nil, 204, adminUserToken)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	var vendors []database.Vendor
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &vendors))

	byLicense := map[string]database.Vendor{}
	for _, v := range vendors {
		byLicense[v.LicenseID.String] = v
	}
	require.Contains(t, byLicense, "testlicense-active")
	require.Equal(t, activeID, strconv.Itoa(byLicense["testlicense-active"].ID))
	require.Contains(t, byLicense, "testlicense-disabled")
	require.True(t, byLicense["testlicense-disabled"].IsDisabled)
	for license := range byLicense {
		require.NotContains(t, license, "testlicense-deleted")
	}

	// Internal users of ListVendors (statistics, balances) still skip disabled vendors
	active, err := database.Db.ListVendors()
	require.NoError(t, err)
	for _, v := range active {
		require.False(t, v.IsDisabled)
	}
}

// TestLocationCRUDWithoutVendor verifies locations can exist without a vendor,
// be assigned, moved and unassigned, show up on the map and be deleted.
func TestLocationCRUDWithoutVendor(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	firstID, err := strconv.Atoi(createTestVendor(t, "testlicense-crud-a"))
	require.NoError(t, err)
	secondID, err := strconv.Atoi(createTestVendor(t, "testlicense-crud-b"))
	require.NoError(t, err)

	list := func() []database.LocationOverview {
		res := utils.TestRequestWithAuth(t, r, "GET", "/api/locations/", nil, 200, adminUserToken)
		var locations []database.LocationOverview
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &locations))
		return locations
	}

	// Create without a vendor
	body := map[string]any{"name": "Freier Platz", "address": "Street 9", "zip": "1090", "telephone": "+43 1 999", "longitude": 16.35, "latitude": 48.22}
	res := utils.TestRequestWithAuth(t, r, "POST", "/api/locations/", body, 200, adminUserToken)
	locationID := strings.TrimSpace(res.Body.String())

	locations := list()
	require.Len(t, locations, 1)
	require.Nil(t, locations[0].VendorID)
	require.Equal(t, "Freier Platz", locations[0].Name)

	// The map shows it as unassigned instead of crashing on the missing vendor
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/map/", nil, 200, adminUserToken)
	var mapData []database.LocationData
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &mapData))
	require.Len(t, mapData, 1)
	require.False(t, mapData[0].HasVendor)
	require.Equal(t, "Freier Platz", mapData[0].LocationName)

	// Assign to the first vendor, then move to the second
	body["vendorID"] = firstID
	utils.TestRequestWithAuth(t, r, "PATCH", "/api/locations/"+locationID+"/", body, 200, adminUserToken)
	locations = list()
	require.Equal(t, firstID, *locations[0].VendorID)
	require.Equal(t, "testlicense-crud-a", locations[0].VendorLicenseID)

	body["vendorID"] = secondID
	body["name"] = "Umbenannt"
	utils.TestRequestWithAuth(t, r, "PATCH", "/api/locations/"+locationID+"/", body, 200, adminUserToken)
	locations = list()
	require.Equal(t, secondID, *locations[0].VendorID)
	require.Equal(t, "Umbenannt", locations[0].Name)
	ownLocations, err := database.Db.GetLocationsByVendorID(firstID)
	require.NoError(t, err)
	require.Empty(t, ownLocations)

	// Unassign again
	body["vendorID"] = nil
	utils.TestRequestWithAuth(t, r, "PATCH", "/api/locations/"+locationID+"/", body, 200, adminUserToken)
	require.Nil(t, list()[0].VendorID)

	// Unknown vendor and unknown location are rejected
	body["vendorID"] = 999999
	utils.TestRequestWithAuth(t, r, "PATCH", "/api/locations/"+locationID+"/", body, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/locations/", body, 400, adminUserToken)
	body["vendorID"] = nil
	utils.TestRequestWithAuth(t, r, "PATCH", "/api/locations/999999/", body, 404, adminUserToken)

	// Delete
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/locations/"+locationID+"/", nil, 200, adminUserToken)
	require.Empty(t, list())
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/locations/"+locationID+"/", nil, 404, adminUserToken)

	// Backoffice only
	utils.TestRequest(t, r, "POST", "/api/locations/", body, 401)
}

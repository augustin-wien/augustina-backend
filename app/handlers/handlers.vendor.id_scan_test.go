package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestScanVendorIDCard checks that the backoffice can look up a vendor by
// scanning the ID card, which stores the scan time
func TestScanVendorIDCard(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorID, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Scan",
		LastName:  "Me",
		Email:     "scan@vendor.com",
		LicenseID: null.StringFrom("Scan 1"),
	})
	require.NoError(t, err)

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/Scan%201/id-scan/", nil, 403, adminUserToken)
	putSettingsFields(t, map[string]string{"IDScanEnabled": "true"}, 200)

	res := utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/Scan%201/id-scan/", nil, 200, adminUserToken)
	var vendor database.Vendor
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &vendor))
	require.Equal(t, vendorID, vendor.ID)
	require.True(t, vendor.LastIDScan.Valid)
	require.WithinDuration(t, time.Now(), vendor.LastIDScan.Time, time.Minute)

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/unknown/id-scan/", nil, 404, adminUserToken)
	utils.TestRequest(t, r, "POST", "/api/vendors/Scan%201/id-scan/", nil, 401)
}

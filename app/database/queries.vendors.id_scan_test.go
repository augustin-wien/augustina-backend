package database

import (
	"errors"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// Test_RecordVendorIDScan checks that scanning an ID card stores the scan time,
// that saving the vendor keeps it and that unknown or deleted vendors are not found
func Test_RecordVendorIDScan(t *testing.T) {
	Db.InitEmptyTestDb()

	vendorID, err := Db.CreateVendor(Vendor{
		FirstName: "Scan",
		LastName:  "Me",
		Email:     "scan@vendor.com",
		LicenseID: null.StringFrom("scan-1"),
	})
	utils.CheckError(t, err)

	vendor, err := Db.GetVendorWithBalanceUpdate(vendorID)
	utils.CheckError(t, err)
	require.False(t, vendor.LastIDScan.Valid)

	scanned, err := Db.RecordVendorIDScan("scan-1")
	utils.CheckError(t, err)
	require.Equal(t, vendorID, scanned.ID)
	require.True(t, scanned.LastIDScan.Valid)
	require.WithinDuration(t, time.Now(), scanned.LastIDScan.Time, time.Minute)

	// Saving the vendor from the backoffice doesn't touch the scan time
	vendor.LastIDScan = null.Time{}
	utils.CheckError(t, Db.UpdateVendor(vendorID, vendor))
	vendor, err = Db.GetVendorWithBalanceUpdate(vendorID)
	utils.CheckError(t, err)
	require.True(t, vendor.LastIDScan.Valid)
	require.WithinDuration(t, scanned.LastIDScan.Time, vendor.LastIDScan.Time, time.Second)

	_, err = Db.RecordVendorIDScan("unknown")
	require.True(t, errors.Is(err, ErrVendorNotFound))

	utils.CheckError(t, Db.DeleteVendor(vendorID))
	_, err = Db.RecordVendorIDScan("scan-1")
	require.True(t, errors.Is(err, ErrVendorNotFound))
}

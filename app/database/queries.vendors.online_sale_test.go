package database

import (
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// Test_VendorFirstOnlineSale checks that only a verified online order sets a
// vendor's first online sale, both in the vendor list and the vendor detail
func Test_VendorFirstOnlineSale(t *testing.T) {
	Db.InitEmptyTestDb()

	sellerID, err := Db.CreateVendor(Vendor{
		FirstName: "Online",
		LastName:  "Seller",
		Email:     "online@vendor.com",
		LicenseID: null.StringFrom("os-1"),
	})
	utils.CheckError(t, err)
	pendingID, err := Db.CreateVendor(Vendor{
		FirstName: "Pending",
		LastName:  "Seller",
		Email:     "pending@vendor.com",
		LicenseID: null.StringFrom("os-2"),
	})
	utils.CheckError(t, err)

	itemID, err := Db.CreateItem(Item{
		Name:        "Test Item",
		Description: "This is a test item description that is long enough",
		Price:       100,
		Type:        "normal_item",
	})
	utils.CheckError(t, err)

	createOrder := func(vendorID int, code string) int {
		account, err := Db.GetAccountByVendorID(vendorID)
		utils.CheckError(t, err)
		orderID, err := Db.CreateOrder(Order{
			OrderCode: null.StringFrom(code),
			Vendor:    vendorID,
			Entries: []OrderEntry{{
				Item:     itemID,
				Quantity: 1,
				Sender:   account.ID,
				Receiver: account.ID,
				IsSale:   true,
			}},
		})
		utils.CheckError(t, err)
		return orderID
	}

	// Nobody has sold online yet
	vendor, err := Db.GetVendorWithBalanceUpdate(sellerID)
	utils.CheckError(t, err)
	require.False(t, vendor.FirstOnlineSale.Valid)

	// A verified order counts, an unverified (unpaid) one doesn't
	utils.CheckError(t, Db.VerifyOrderAndCreatePayments(createOrder(sellerID, "os-order-1"), 1))
	createOrder(pendingID, "os-order-2")

	vendor, err = Db.GetVendorWithBalanceUpdate(sellerID)
	utils.CheckError(t, err)
	require.True(t, vendor.FirstOnlineSale.Valid)
	vendor, err = Db.GetVendorWithBalanceUpdate(pendingID)
	utils.CheckError(t, err)
	require.False(t, vendor.FirstOnlineSale.Valid)

	vendors, err := Db.ListVendorsWithDisabled()
	utils.CheckError(t, err)
	byID := make(map[int]null.Time)
	for _, v := range vendors {
		byID[v.ID] = v.FirstOnlineSale
	}
	require.True(t, byID[sellerID].Valid)
	require.WithinDuration(t, time.Now(), byID[sellerID].Time, time.Minute)
	require.Contains(t, byID, pendingID)
	require.False(t, byID[pendingID].Valid)
}

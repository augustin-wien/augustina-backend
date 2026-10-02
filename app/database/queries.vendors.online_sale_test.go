package database

import (
	"context"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// Test_VendorLastOnlineSale checks that only a verified online order counts as
// a vendor's online sale and that the latest one is reported, both in the vendor
// list and the vendor detail
func Test_VendorLastOnlineSale(t *testing.T) {
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
	require.False(t, vendor.LastOnlineSale.Valid)

	// A verified order counts, an unverified (unpaid) one doesn't
	olderOrderID := createOrder(sellerID, "os-order-1")
	utils.CheckError(t, Db.VerifyOrderAndCreatePayments(olderOrderID, 1))
	utils.CheckError(t, Db.VerifyOrderAndCreatePayments(createOrder(sellerID, "os-order-3"), 1))
	createOrder(pendingID, "os-order-2")

	// Move the first sale back so the second one is clearly the latest
	lastWeek := time.Now().AddDate(0, 0, -7)
	utils.CheckError(t, Db.EntClient.Order.UpdateOneID(olderOrderID).SetTimestamp(lastWeek).Exec(context.Background()))

	vendor, err = Db.GetVendorWithBalanceUpdate(sellerID)
	utils.CheckError(t, err)
	require.True(t, vendor.LastOnlineSale.Valid)
	require.WithinDuration(t, time.Now(), vendor.LastOnlineSale.Time, time.Minute)
	vendor, err = Db.GetVendorWithBalanceUpdate(pendingID)
	utils.CheckError(t, err)
	require.False(t, vendor.LastOnlineSale.Valid)

	vendors, err := Db.ListVendorsWithDisabled()
	utils.CheckError(t, err)
	byID := make(map[int]null.Time)
	for _, v := range vendors {
		byID[v.ID] = v.LastOnlineSale
	}
	require.True(t, byID[sellerID].Valid)
	require.WithinDuration(t, time.Now(), byID[sellerID].Time, time.Minute)
	require.Contains(t, byID, pendingID)
	require.False(t, byID[pendingID].Valid)
}

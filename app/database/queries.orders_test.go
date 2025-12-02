package database

import (
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func Test_VerifyOrderSetsVerifiedAt(t *testing.T) {
	Db.InitEmptyTestDb()

	// 1. Create Vendor
	vendor := Vendor{
		FirstName: "Test",
		LastName:  "Vendor",
		Email:     "test@vendor.com",
		LicenseID: null.StringFrom("tv-123"),
	}
	vendorID, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	// 2. Create Item
	item := Item{
		Name:        "Test Item",
		Description: "This is a test item description that is long enough",
		Price:       100,
	}
	itemID, err := Db.CreateItem(item)
	utils.CheckError(t, err)

	// 3. Create Order
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	order := Order{
		OrderCode: null.StringFrom("test-order"),
		Vendor:    vendorID,
		Entries: []OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Sender:   vendorAccount.ID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	// 4. Verify Order
	err = Db.VerifyOrderAndCreatePayments(orderID, 1)
	utils.CheckError(t, err)

	// 5. Check VerifiedAt
	updatedOrder, err := Db.GetOrderByID(orderID)
	utils.CheckError(t, err)

	require.True(t, updatedOrder.Verified)
	require.True(t, updatedOrder.VerifiedAt.Valid)
	require.WithinDuration(t, time.Now(), updatedOrder.VerifiedAt.Time, 5*time.Second)
}

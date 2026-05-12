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
		Type:        "normal_item",
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

func Test_VerifyOrderCreatesAbonementForAboItem(t *testing.T) {
	Db.InitEmptyTestDb()

	// 1. Create Vendor
	vendor := Vendor{
		FirstName: "Abo",
		LastName:  "Vendor",
		Email:     "abo-test-vendor@vendor.com",
		LicenseID: null.StringFrom("atv-123"),
	}
	vendorID, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	// 2. Create license_item
	licenseItem := Item{
		Name:          "Test License Item For Abo",
		Description:   "License for test abonement item",
		Price:         100,
		Type:          "license_item",
		IsLicenseItem: true,
		LicenseGroup:  null.StringFrom("test_edition"),
	}
	licenseItemID, err := Db.CreateItem(licenseItem)
	utils.CheckError(t, err)

	// 3. Create abonement item (enabled, linked license_item, license group)
	aboItem := Item{
		Name:         "Test Abonement Item",
		Description:  "A test abonement item with a sufficiently long description",
		Price:        1000,
		Type:         "abonement",
		Disabled:     false,
		LicenseItem:  null.IntFrom(int64(licenseItemID)),
		LicenseGroup: null.StringFrom("test_edition"),
	}
	aboItemID, err := Db.CreateItem(aboItem)
	utils.CheckError(t, err)

	// 4. Create Order with customer email and the abonement item
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	order := Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom("abo-customer@example.com"),
		Entries: []OrderEntry{
			{
				Item:     aboItemID,
				Quantity: 1,
				Sender:   vendorAccount.ID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	// 5. Verify Order — Keycloak calls return errors (nil client) but are logged and skipped
	err = Db.VerifyOrderAndCreatePayments(orderID, 1)
	utils.CheckError(t, err)

	// 6. Customer record must have been created
	customer, err := Db.GetCustomerByEmail("abo-customer@example.com")
	utils.CheckError(t, err)
	require.NotNil(t, customer)

	// 7. Exactly one abonement must have been created for that customer
	abonements, err := Db.ListAbonementsByCustomer(customer.ID)
	utils.CheckError(t, err)
	require.Len(t, abonements, 1)

	abo := abonements[0]
	require.Equal(t, customer.ID, abo.CustomerID)
	require.Equal(t, aboItemID, abo.ItemID)
	require.Equal(t, "active", abo.Status)
	require.WithinDuration(t, time.Now(), abo.FromDate, 5*time.Second)
	require.WithinDuration(t, time.Now().AddDate(1, 0, 0), abo.ToDate, 5*time.Second)
}

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

const testItemDescription = "This is a test item description that is long enough"

func TestCreateItemResurrectsArchivedItem(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	pdfID, err := Db.CreatePDF(PDF{Path: "/tmp/resurrect.pdf"})
	require.NoError(t, err)
	licenseID, err := Db.CreateItem(Item{Name: "Resurrect License", Description: testItemDescription, Price: 100, Type: "license_item", IsLicenseItem: true})
	require.NoError(t, err)

	id, err := Db.CreateItem(Item{Name: "Resurrect Me", Description: testItemDescription, Price: 300, Type: "normal_item"})
	require.NoError(t, err)

	// The name of an active item cannot be reused
	_, err = Db.CreateItem(Item{Name: "Resurrect Me", Description: testItemDescription, Price: 300, Type: "normal_item"})
	require.ErrorIs(t, err, ErrItemNameTaken)

	require.NoError(t, Db.DeleteItem(id))

	// Creating an item with the name of an archived one brings that row back
	again, err := Db.CreateItem(Item{
		Name:          "Resurrect Me",
		Description:   "Neue Beschreibung, lang genug für den Validator",
		Price:         450,
		Type:          "issue",
		IsPDFItem:     true,
		PDF:           null.IntFrom(pdfID),
		LicenseItem:   null.IntFrom(int64(licenseID)),
		LicenseGroup:  null.StringFrom("resurrect_group"),
		ItemColor:     null.StringFrom("#123456"),
		ItemTextColor: null.StringFrom("#ffffff"),
	})
	require.NoError(t, err)
	require.Equal(t, id, again)

	item, err := Db.GetItem(id)
	require.NoError(t, err)
	require.False(t, item.Archived)
	require.Equal(t, 450, item.Price)
	require.Equal(t, "issue", item.Type)
	require.EqualValues(t, pdfID, item.PDF.Int64)
	require.EqualValues(t, licenseID, item.LicenseItem.Int64)
	require.Equal(t, "resurrect_group", item.LicenseGroup.String)

	// Resurrecting without PDF and license clears both
	require.NoError(t, Db.DeleteItem(id))
	_, err = Db.CreateItem(Item{Name: "Resurrect Me", Description: testItemDescription, Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	item, err = Db.GetItem(id)
	require.NoError(t, err)
	require.False(t, item.PDF.Valid)
	require.False(t, item.LicenseItem.Valid)
}

func TestTxQueryHelpers(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	vendorID, err := Db.CreateVendor(Vendor{FirstName: "Tx", LastName: "Vendor", Email: "tx@vendor.test", LicenseID: null.StringFrom("tv-tx-1")})
	require.NoError(t, err)
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	orga, err := Db.GetAccountByType("Orga")
	require.NoError(t, err)

	itemID, err := Db.CreateItem(Item{Name: "Tx Item", Description: testItemDescription, Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	disabledID, err := Db.CreateItem(Item{Name: "Tx Disabled", Description: testItemDescription, Price: 300, Type: "normal_item", Disabled: true})
	require.NoError(t, err)

	orderID, err := Db.CreateOrder(Order{Vendor: vendorID, Entries: []OrderEntry{{Item: itemID, Quantity: 1, Sender: orga.ID, Receiver: vendorAccount.ID}}})
	require.NoError(t, err)

	// Orders with disabled or missing items are refused
	_, err = Db.CreateOrder(Order{Vendor: vendorID, Entries: []OrderEntry{{Item: disabledID, Quantity: 1, Sender: orga.ID, Receiver: vendorAccount.ID}}})
	require.Error(t, err)
	_, err = Db.CreateOrder(Order{Vendor: vendorID, Entries: []OrderEntry{{Item: 999999, Quantity: 1, Sender: orga.ID, Receiver: vendorAccount.ID}}})
	require.Error(t, err)

	tx, err := Db.EntClient.Tx(context.Background())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	item, err := Db.GetItemTx(tx, itemID)
	require.NoError(t, err)
	require.Equal(t, "Tx Item", item.Name)
	// Orders keep referring to items that were disabled later
	disabled, err := Db.GetItemTx(tx, disabledID)
	require.NoError(t, err)
	require.True(t, disabled.Disabled)
	_, err = Db.GetItemTx(tx, 999999)
	require.Error(t, err)

	order, err := Db.GetOrderByIDTx(tx, orderID)
	require.NoError(t, err)
	require.Len(t, order.Entries, 1)
	require.Equal(t, 300, order.Entries[0].Price)
	_, err = Db.GetOrderByIDTx(tx, 999999)
	require.Error(t, err)
}

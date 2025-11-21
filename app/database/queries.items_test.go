package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestCreateAndUpdateDigitalItem ensures we can create a digital (PDF) item and update it
func TestCreateAndUpdateDigitalItem(t *testing.T) {
	// start with a clean DB for deterministic results
	Db.InitEmptyTestDb()

	// create a PDF resource to reference from the item
	pdf := PDF{
		Path:      "test_digital.pdf",
		Timestamp: time.Now(),
	}
	pdfID, err := Db.CreatePDF(pdf)
	require.NoError(t, err)
	require.True(t, pdfID > 0)

	// create an item that is a PDF (digital) item and references the pdfID
	it := Item{
		Name:        "digital-item-test",
		Description: "A test digital item",
		Price:       499,
		IsPDFItem:   true,
		PDF:         null.IntFrom(pdfID),
	}

	itemID, err := Db.CreateItem(it)
	require.NoError(t, err)
	require.True(t, itemID > 0)

	// fetch item and verify fields
	fetched, err := Db.GetItem(itemID)
	require.NoError(t, err)
	require.Equal(t, "digital-item-test", fetched.Name)
	require.Equal(t, 499, fetched.Price)
	require.Equal(t, true, fetched.IsPDFItem)
	require.True(t, fetched.PDF.Valid)
	require.Equal(t, int64(pdfID), fetched.PDF.ValueOrZero())

	// update the item: change name/price and clear the PDF (set IsPDFItem false)
	updated := fetched
	updated.Name = "digital-item-test-updated"
	updated.Price = 799
	updated.IsPDFItem = false
	updated.PDF = null.Int{} // clear

	err = Db.UpdateItem(itemID, updated)
	require.NoError(t, err)

	// re-fetch and verify update
	fetched2, err := Db.GetItem(itemID)
	require.NoError(t, err)
	require.Equal(t, "digital-item-test-updated", fetched2.Name)
	require.Equal(t, 799, fetched2.Price)
	require.Equal(t, false, fetched2.IsPDFItem)
	require.False(t, fetched2.PDF.Valid)
}

// TestCreateAndUpdateLicenseItem ensures we can create a license item and
// an item that depends on that license, then update the dependent item to
// remove the license reference.
func TestCreateAndUpdateLicenseItem(t *testing.T) {
	// start with a clean DB for deterministic results
	Db.InitEmptyTestDb()

	// create a license item
	license := Item{
		Name:          "digital-newspaper-license",
		Description:   "License for digital newspaper",
		Price:         100,
		IsLicenseItem: true,
		LicenseGroup:  null.StringFrom("digital_newspaper"),
	}
	licenseID, err := Db.CreateItem(license)
	require.NoError(t, err)
	require.True(t, licenseID > 0)

	// create an item that requires the license
	dep := Item{
		Name:        "newspaper-with-license",
		Description: "Newspaper requiring license",
		Price:       599,
		LicenseItem: null.IntFrom(int64(licenseID)),
	}
	depID, err := Db.CreateItem(dep)
	require.NoError(t, err)
	require.True(t, depID > 0)

	// fetch dependent item and verify license relation
	fetched, err := Db.GetItem(depID)
	require.NoError(t, err)
	require.True(t, fetched.LicenseItem.Valid)
	require.Equal(t, int64(licenseID), fetched.LicenseItem.ValueOrZero())

	// update: clear the license on the dependent item
	updated := fetched
	updated.LicenseItem = null.Int{}
	updated.Name = "newspaper-no-license"

	err = Db.UpdateItem(depID, updated)
	require.NoError(t, err)

	// re-fetch and verify license cleared
	fetched2, err := Db.GetItem(depID)
	require.NoError(t, err)
	require.False(t, fetched2.LicenseItem.Valid)
	require.Equal(t, "newspaper-no-license", fetched2.Name)
}

// TestUpdateItemWithLicense verifies updating an item that already has a
// LicenseItem works: switch license to another license and then clear it.
func TestUpdateItemWithLicense(t *testing.T) {
	Db.InitEmptyTestDb()

	// create two license items
	licA := Item{
		Name:          "license-A",
		Description:   "License A",
		Price:         50,
		IsLicenseItem: true,
	}
	licB := Item{
		Name:          "license-B",
		Description:   "License B",
		Price:         75,
		IsLicenseItem: true,
	}
	licAID, err := Db.CreateItem(licA)
	require.NoError(t, err)
	require.True(t, licAID > 0)
	licBID, err := Db.CreateItem(licB)
	require.NoError(t, err)
	require.True(t, licBID > 0)

	// create an item that references licA
	it := Item{
		Name:        "item-with-license",
		Description: "Item referencing license A",
		Price:       200,
		LicenseItem: null.IntFrom(int64(licAID)),
	}
	itID, err := Db.CreateItem(it)
	require.NoError(t, err)
	require.True(t, itID > 0)

	// verify initial license is A
	fetched, err := Db.GetItem(itID)
	require.NoError(t, err)
	require.True(t, fetched.LicenseItem.Valid)
	require.Equal(t, int64(licAID), fetched.LicenseItem.ValueOrZero())

	// update to point to licB
	updated := fetched
	updated.LicenseItem = null.IntFrom(int64(licBID))
	updated.Name = "item-with-license-updated"
	err = Db.UpdateItem(itID, updated)
	require.NoError(t, err)

	fetched2, err := Db.GetItem(itID)
	require.NoError(t, err)
	require.True(t, fetched2.LicenseItem.Valid)
	require.Equal(t, int64(licBID), fetched2.LicenseItem.ValueOrZero())
	require.Equal(t, "item-with-license-updated", fetched2.Name)

	// now clear the license
	updated2 := fetched2
	updated2.LicenseItem = null.Int{}
	updated2.Name = "item-without-license"
	err = Db.UpdateItem(itID, updated2)
	require.NoError(t, err)

	fetched3, err := Db.GetItem(itID)
	require.NoError(t, err)
	require.False(t, fetched3.LicenseItem.Valid)
	require.Equal(t, "item-without-license", fetched3.Name)
}

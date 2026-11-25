package database

import (
	"time"

	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// devItemIDByType returns the first item of the given type, or 0 if none exists
func (db *Database) devItemIDByType(itemType string) (int, error) {
	items, err := db.ListItemsWithDisabled(false, false)
	if err != nil {
		return 0, err
	}
	for _, it := range items {
		if it.Type == itemType {
			return it.ID, nil
		}
	}
	return 0, nil
}

// createDevItems creates the dev items in addition to the default ones
func (db *Database) createDevItems() (items devItems, err error) {
	for itemType, id := range map[string]*int{
		"issue":             &items.newspaper,
		"donation":          &items.donation,
		"transaction_costs": &items.transactionCosts,
	} {
		if *id, err = db.devItemIDByType(itemType); err != nil {
			return
		}
	}

	items.digitalLicense, err = db.CreateItem(Item{
		Name:          "Digitale Zeitung (Lizenz)",
		Description:   "Lizenz für digitale Zeitungsausgabe",
		Price:         50,
		IsLicenseItem: true,
		Image:         "img/demo_digital.jpg",
		Type:          "license_item",
	})
	if err != nil {
		log.Error("createDevItems: ", err)
		return
	}

	items.digitalNewspaper, err = db.CreateItem(Item{
		Name:         "Digitale Zeitung",
		Description:  "Digitale Zeitungsausgabe",
		Price:        300,
		LicenseItem:  null.IntFrom(int64(items.digitalLicense)),
		LicenseGroup: null.StringFrom("testedition"),
		Image:        "img/demo_digital.jpg",
		Type:         "issue",
		ItemOrder:    2,
	})
	if err != nil {
		log.Error("Dev newspaper creation failed ", zap.Error(err))
		return
	}

	items.calendar, err = db.CreateItem(Item{
		Name:          "Kalender",
		Description:   "Kalender für das Jahr 2024",
		Price:         800,
		Image:         "img/demo_kalender.jpg",
		Type:          "normal_item",
		ItemOrder:     1,
		ItemColor:     null.StringFrom("#FFD100"),
		ItemTextColor: null.StringFrom("#1A1A1A"),
	})
	if err != nil {
		log.Error("Dev calendar creation failed ", zap.Error(err))
		return
	}

	pdfLicenseID, err := db.CreateItem(Item{
		Name:          "Digitale Zeitung (Lizenz) 2",
		Description:   "Lizenz für digitale Zeitungsausgabe 2",
		Price:         50,
		IsLicenseItem: true,
		Image:         "img/demo_digital.jpg",
		Type:          "license_item",
	})
	if err != nil {
		log.Error("createDevItems: ", err)
		return
	}

	pdfID, err := db.CreatePDF(PDF{
		Path:      "test.pdf",
		Timestamp: time.Now(),
	})
	if err != nil {
		log.Error("createDevItems PDF creation failed: ", err)
		return
	}

	items.pdfNewspaper, err = db.CreateItem(Item{
		Name:         "Digitale Zeitung (PDF)",
		Description:  "Digitale Zeitungsausgabe mit PDF",
		Price:        300,
		LicenseItem:  null.IntFrom(int64(pdfLicenseID)),
		LicenseGroup: null.StringFrom("testedition"),
		Image:        "img/demo_digital.jpg",
		PDF:          null.IntFrom(pdfID),
		IsPDFItem:    true,
		Type:         "issue",
		ItemOrder:    3,
	})
	if err != nil {
		log.Error("Dev newspaper with PDF creation failed ", zap.Error(err))
		return
	}

	items.bag, err = db.CreateItem(Item{
		Name:          "Stofftasche",
		Description:   "Stofftasche mit Logo, dunkel eingefärbt",
		Price:         1200,
		Type:          "normal_item",
		ItemOrder:     4,
		ItemColor:     null.StringFrom("#1F3A5F"),
		ItemTextColor: null.StringFrom("#FFFFFF"),
	})
	if err != nil {
		log.Error("Dev bag creation failed ", zap.Error(err))
		return
	}

	// Temporarily unavailable, hidden from the shop but visible in the backoffice
	_, err = db.CreateItem(Item{
		Name:        "Buch: Geschichten von der Straße",
		Description: "Derzeit vergriffen",
		Price:       1500,
		Type:        "normal_item",
		Disabled:    true,
		ItemOrder:   5,
	})
	if err != nil {
		log.Error("Dev disabled item creation failed ", zap.Error(err))
	}
	return
}

// SeedDevArchivedItems adds the archived dev items to an already initialized
// database, selling the sold one via the vendor with the given license ID.
func (db *Database) SeedDevArchivedItems(vendorLicenseID string) error {
	vendor, err := db.GetVendorByLicenseID(vendorLicenseID)
	if err != nil {
		return err
	}
	return db.createDevArchivedItems([]int{vendor.ID})
}

// createDevArchivedItems creates items that are deleted (archived) afterwards.
// One of them is sold first, so payments, statistics and exports reference a
// deleted item — the case the product archive exists for.
func (db *Database) createDevArchivedItems(vendorIDs []int) (err error) {
	soldItemID, err := db.CreateItem(Item{
		Name:        "Kalender 2023",
		Description: "Kalender für das Jahr 2023 (ausverkauft)",
		Price:       700,
		Image:       "img/demo_kalender.jpg",
		Type:        "normal_item",
	})
	if err != nil {
		return
	}
	unsoldItemID, err := db.CreateItem(Item{
		Name:        "Postkartenset",
		Description: "Postkartenset (nicht mehr im Sortiment)",
		Price:       500,
		Type:        "normal_item",
	})
	if err != nil {
		return
	}

	if len(vendorIDs) > 0 {
		var buyerAccountID int
		buyerAccountID, err = db.GetAccountTypeID("UserAnon")
		if err != nil {
			return
		}
		var vendorAccount Account
		vendorAccount, err = db.GetAccountByVendorID(vendorIDs[0])
		if err != nil {
			return
		}
		var orderID int
		orderID, err = db.CreateOrder(Order{
			OrderCode: null.NewString("devOrderArchived", true),
			Vendor:    vendorIDs[0],
			Entries: []OrderEntry{
				{
					Item:     soldItemID,
					Quantity: 1,
					Sender:   buyerAccountID,
					Receiver: vendorAccount.ID,
					IsSale:   true,
				},
			},
		})
		if err != nil {
			return
		}
		err = db.VerifyOrderAndCreatePayments(orderID, 12346)
		if err != nil {
			return
		}
		err = db.backdateDevOrder(orderID, time.Now().AddDate(0, -9, 0))
		if err != nil {
			return
		}
	}

	if err = db.DeleteItem(soldItemID); err != nil {
		return
	}
	return db.DeleteItem(unsoldItemID)
}

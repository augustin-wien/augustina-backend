package database

import (
	"reflect"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// CreateDevData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	vendorIDs, err := db.createDevVendors()
	if err != nil {
		return err
	}
	_, err = db.createDevItems()
	if err != nil {
		return err
	}
	err = db.createDevOrdersAndPayments(vendorIDs)
	if err != nil {
		return err
	}
	return err
}

// CreateDevVendors creates test users for the application
func (db *Database) createDevVendors() (vendorIDs []int, err error) {
	vendor := Vendor{
		KeycloakID: "keycloakid1",
		URLID:      "www.augustin.or.at/fl-123",
		LicenseID:  null.NewString("fl-123", true),
		FirstName:  "firstname1",
		LastName:   "lastname1",
		Email:      "email1",
	}
	vendorID, err := db.CreateVendor(vendor)
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
	}
	vendorIDs = append(vendorIDs, vendorID)

	return
}

// CreateDevItems creates test items for the application
func (db *Database) createDevItems() (ids []int, err error) {

	digitalNewspaperLicense := Item{
		Name:          "Digitale Zeitung (Lizenz)",
		Description:   "Lizenz für digitale Zeitungsausgabe",
		Price:         50,
		Archived:      false,
		IsLicenseItem: true,
	}

	digitalNewspaperLicenseID, err := db.CreateItem(digitalNewspaperLicense)
	if err != nil {
		log.Error(err)
		return
	}

	digitalNewspaper := Item{
		Name:        "Digitale Zeitung",
		Description: "Digitale Zeitungsausgabe",
		Price:       300,
		LicenseItem: null.NewInt(int64(digitalNewspaperLicenseID), true),
		Archived:    false,
	}

	calendar := Item{
		Name:        "Kalender",
		Description: "Kalender für das Jahr 2024",
		Price:       800,
		Archived:    false,
	}

	// Create newspaper
	_, err = db.CreateItem(digitalNewspaper)
	if err != nil {
		log.Error("Dev newspaper creation failed ", zap.Error(err))
		return
	}

	// Create calendar
	_, err = db.CreateItem(calendar)
	if err != nil {
		pg := err.(*pgconn.PgError)
		if reflect.TypeOf(err) == reflect.TypeOf(&pgconn.PgError{}) {
			log.Info("Postgres details error are: ", pg.Detail)
		}
		log.Error("Dev newspaper creation failed ", zap.Error(err))
		return
	}

	return ids, err
}

//After initializing items and calling the function above, the database should look like this:
// item[0] = Newspaper
// item[1] = Donation
// item[2] = Transaction cost
// item[3] = Digital newspaper
// item[4] = Digital newspaper license
// item[5] = Calendar
//

// CreateDevOrdersAndPayments creates test orders and payments
// This function replicates what happens in CreateOrder handler
// User buys 2 newspapers (-600), 1 calendar (-800)
// Orga gets 2 licenses (100) and looses 27 transaction costs (-27)
// Vendor gets all sales (1600) and pays 2 licenses (-100)
func (db *Database) createDevOrdersAndPayments(vendorIDs []int) (err error) {

	buyerAccountID, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}

	orgaAccountID, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}

	// paypalAccountID, err := db.GetAccountTypeID("Paypal")
	// if err != nil {
	// 	return
	// }

	vendorAccount, err := db.GetAccountByVendorID(vendorIDs[0])
	if err != nil {
		return
	}

	items, err := db.ListItems(false)
	if err != nil {
		return
	}

	// Create order
	order := Order{
		OrderCode: null.NewString("devOrder1", true),
		Vendor:    vendorIDs[0],
		Entries: []OrderEntry{
			{
				Item:     items[2].ID, // Digital Newspaper
				Quantity: 2,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},

			// License for newspaper is paid to orga
			{
				Item:     items[1].ID, // Newspaper License
				Quantity: 2,
				Sender:   vendorAccount.ID,
				Receiver: orgaAccountID,
			},
			{
				Item:     items[3].ID, // Calendar
				Quantity: 1,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
			{
				Item:     items[1].ID, // Donation
				Quantity: 100,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
			},
		},
	}

	// Make order
	orderID, err := db.CreateOrder(order)
	if err != nil {
		return
	}

	// Create payments for order
	err = db.VerifyOrderAndCreatePayments(orderID, 12345)
	if err != nil {
		return
	}

	// Create transaction cost payments
	// err = db.CreatePayedOrderEntries(orderID, []OrderEntry{

	// 	// Vendor pays transaction costs
	// 	{
	// 		Item:     items[2].ID, // Transaction cost
	// 		Quantity: 27,          // 27 cents transaction cost
	// 		Sender:   vendorAccount.ID,
	// 		Receiver: paypalAccountID,
	// 	},

	// 	// Vendor gets transaction costs back from orga
	// 	{
	// 		Item:     items[2].ID, // Transaction cost
	// 		Quantity: 27,          // 27 cents transaction cost
	// 		Sender:   orgaAccountID,
	// 		Receiver: vendorAccount.ID,
	// 	},
	// })
	// if err != nil {
	// 	return
	// }

	return
}

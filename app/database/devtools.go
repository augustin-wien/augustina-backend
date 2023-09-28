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
	itemIDs, err := db.createDevItems()
	if err != nil {
		return err
	}
	err = db.createDevOrdersAndPayments(vendorIDs, itemIDs)
	if err != nil {
		return err
	}
	err = db.createDevSettings()
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

	newspaperLicense := Item{
		Name:        "Zeitung (Lizenz)",
		Description: "Lizenz für aktuelle Zeitungsausgabe",
		Price:       50,
		Archived:    false,
	}

	newspaperLicenseID, err := db.CreateItem(newspaperLicense)
	if err != nil {
		log.Error(err)
		return
	}

	newspaper := Item{
		Name:        "Zeitung",
		Description: "Aktuelle Zeitungsausgabe",
		Price:       300,
		LicenseItem: null.NewInt(int64(newspaperLicenseID), true),
		Archived:    false,
	}

	calendar := Item{
		Name:        "Kalender",
		Description: "Kalender für das Jahr 2024",
		Price:       800,
		Archived:    false,
	}

	donation := Item{
		Name:        "Spende",
		Description: "Spenden für das eigene Wohlbefinden",
		Price:       1,
		Archived:    false,
	}

	transactionCost := Item{
		Name:        "Transaktionskosten2",
		Description: "Transaktionskosten2",
		Price:       1,
		Archived:    false,
	}

	// Create newspaper
	newspaperID, err := db.CreateItem(newspaper)
	if err != nil {
		log.Error("Dev newspaper creation failed ", zap.Error(err))
		return
	}

	// Create calendar
	calendarID, err := db.CreateItem(calendar)
	if err != nil {
		pg := err.(*pgconn.PgError)
		if reflect.TypeOf(err) == reflect.TypeOf(&pgconn.PgError{}) {
			log.Info("Postgres details error are: ", pg.Detail)
		}
		log.Error("Dev newspaper creation failed ", zap.Error(err))
		return
	}

	// Create donation
	donationID, err := db.CreateItem(donation)
	if err != nil {
		log.Error("Dev donation creation failed ", zap.Error(err))
		return
	}

	// Create donation
	transactionCostID, err := db.CreateItem(transactionCost)
	if err != nil {
		log.Error(err)
		return
	}

	ids = append(ids, newspaperID, newspaperLicenseID, calendarID, donationID, transactionCostID)

	return ids, err
}

// CreateDevOrdersAndPayments creates test orders and payments
// This function replicates what happens in CreateOrder handler
// User buys 2 newspapers (-600), 1 calendar (-800), 200 donations (-200)
// Orga gets 2 licenses (100) and looses 27 transaction costs (-27)
// Vendor gets all sales (1600) and pays 2 licenses (-100)
func (db *Database) createDevOrdersAndPayments(vendorIDs []int, itemIDs []int) (err error) {

	buyerAccount, err := db.GetAccountByType("UserAnon")
	if err != nil {
		return
	}

	orgaAccount, err := db.GetAccountByType("Orga")
	if err != nil {
		return
	}

	paypalAccount, err := db.GetAccountByType("Paypal")
	if err != nil {
		return
	}

	vendorAccount, err := db.GetAccountByVendorID(vendorIDs[0])
	if err != nil {
		return
	}

	// Create order
	order := Order{
		OrderCode: null.NewString("devOrder1", true),
		Vendor:    vendorIDs[0],
		Entries: []OrderEntry{
			{
				Item:         itemIDs[0], // Newspaper
				Quantity:     2,
				Sender:       buyerAccount.ID,
				Receiver:     vendorAccount.ID,
				SenderName:   buyerAccount.Name,
				ReceiverName: vendorAccount.Name,
				IsSale:       true,
			},

			// License for newspaper is paid to orga
			{
				Item:         itemIDs[1], // Newspaper License
				Quantity:     2,
				Sender:       vendorAccount.ID,
				Receiver:     orgaAccount.ID,
				SenderName:   vendorAccount.Name,
				ReceiverName: orgaAccount.Name,
			},
			{
				Item:         itemIDs[2], // Calendar
				Quantity:     1,
				Sender:       buyerAccount.ID,
				Receiver:     vendorAccount.ID,
				SenderName:   buyerAccount.Name,
				ReceiverName: vendorAccount.Name,
				IsSale:       true,
			},
			{
				Item:         itemIDs[3], // Donation
				Quantity:     200,        // 2 Euros donation
				Sender:       buyerAccount.ID,
				Receiver:     vendorAccount.ID,
				SenderName:   buyerAccount.Name,
				ReceiverName: vendorAccount.Name,
				IsSale:       true,
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
	err = db.CreatePayedOrderEntries(orderID, []OrderEntry{

		// Vendor pays transaction costs
		{
			Item:         itemIDs[4], // Transaction cost
			Quantity:     27,         // 27 cents transaction cost
			Sender:       vendorAccount.ID,
			Receiver:     paypalAccount.ID,
			SenderName:   vendorAccount.Name,
			ReceiverName: paypalAccount.Name,
		},

		// Vendor gets transaction costs back from orga
		{
			Item:         itemIDs[4], // Transaction cost
			Quantity:     27,         // 27 cents transaction cost
			Sender:       orgaAccount.ID,
			Receiver:     vendorAccount.ID,
			SenderName:   orgaAccount.Name,
			ReceiverName: vendorAccount.Name,
		},
	})
	if err != nil {
		return
	}

	return
}

// CreateDevSettings creates test settings for the application
func (db *Database) createDevSettings() (err error) {
	settings := Settings{
		Color:                      "#FF0000",
		Logo:                       "/img/Augustin-Logo-Rechteck.jpg",
		MainItem:                   null.IntFrom(2),
		MaxOrderAmount:             5000,
		OrgaCoversTransactionCosts: true,
	}

	err = db.UpdateSettings(settings)
	if err != nil {
		log.Error("Dev settings creation failed ", zap.Error(err))
	}

	return err
}

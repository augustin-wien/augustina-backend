package database

import (
	"reflect"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	schema "github.com/augustin-wien/augustina-backend/ent/schema"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// CreateDevData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	vendorIDs, err := db.createDevVendors()
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevLocations(vendorIDs)
	if err != nil {
		log.Error("Dev data location creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevComments(vendorIDs)
	if err != nil {
		log.Error("Dev data comment creation failed ", zap.Error(err))
		return err
	}
	_, err = db.createDevItems()
	if err != nil {
		log.Error("Dev data item creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevOrdersAndPayments(vendorIDs)
	if err != nil {
		log.Error("Dev data order creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPayout(vendorIDs)
	if err != nil {
		log.Error("Dev data payout creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevCustomersAndAbonements()
	if err != nil {
		log.Error("Dev data customers/abonements creation failed ", zap.Error(err))
		return err
	}
	return err
}

// CreateDevVendors creates test users for the application
func (db *Database) createDevVendors() (vendorIDs []int, err error) {
	vendor := Vendor{
		KeycloakID: "keycloakid1",
		UrlID:      "www.augustin.or.at/fl-123",
		LicenseID:  null.NewString("fl-123", true),
		FirstName:  "firstname1",
		LastName:   "lastname1",
		Email:      "test_vendor@example.com",
	}
	vendorID, err := db.CreateVendor(vendor)
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
	}
	vendorIDs = append(vendorIDs, vendorID)

	vendor2 := Vendor{
		KeycloakID: "keycloakid2",
		UrlID:      "www.augustin.or.at/fl-234",
		LicenseID:  null.NewString("fl-234", true),
		FirstName:  "Recep",
		LastName:   "lastname2",
		Email:      "test_vendor2@example.com",
	}
	_, err = db.CreateVendor(vendor2)
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
	}

	return
}

func (db *Database) createDevLocations(vendorIDs []int) (err error) {
	if len(vendorIDs) == 0 {
		return nil
	}

	vendorID := vendorIDs[0]
	locations := []ent.Location{
		{
			Name:      "Morning Market",
			Address:   "Marktplatz 1",
			Longitude: 16.3725,
			Latitude:  48.2082,
			Zip:       "1010",
			WorkingTime: &schema.WorkingTime{
				Mode: "everyday",
				Everyday: []schema.TimeRange{
					{From: "08:00", To: "12:00"},
				},
			},
		},
		{
			Name:      "Evening Stand",
			Address:   "Kulturstraße 7",
			Longitude: 16.3790,
			Latitude:  48.2110,
			Zip:       "1020",
			WorkingTime: &schema.WorkingTime{
				Mode: "by_day",
				WeekDays: map[string][]schema.TimeRange{
					"mon": {{From: "09:00", To: "17:00"}},
					"tue": {{From: "09:00", To: "17:00"}},
					"wed": {{From: "09:00", To: "17:00"}},
					"thu": {{From: "09:00", To: "17:00"}},
					"fri": {{From: "09:00", To: "17:00"}},
					"sat": {{FullDay: true}},
				},
			},
		},
	}

	for _, location := range locations {
		err = db.CreateLocation(vendorID, location)
		if err != nil {
			log.Error("Dev data location creation failed ", zap.Error(err))
			return err
		}
	}

	return nil
}

func (db *Database) createDevComments(vendorIDs []int) (err error) {
	if len(vendorIDs) == 0 {
		return nil
	}

	now := time.Now()
	comments := []ent.Comment{
		{
			Comment:    "Demo vendor is active and ready for use.",
			Warning:    false,
			CreatedAt:  now.Add(-72 * time.Hour),
			ResolvedAt: now.Add(-72 * time.Hour),
		},
		{
			Comment:    "Please verify the demo vendor's contract details.",
			Warning:    true,
			CreatedAt:  now.Add(-24 * time.Hour),
			ResolvedAt: now,
		},
	}

	for _, comment := range comments {
		err = db.CreateVendorComment(vendorIDs[0], comment)
		if err != nil {
			log.Error("Dev data comment creation failed ", zap.Error(err))
			return err
		}
	}

	return nil
}

// CreateDevItems creates test items for the application
func (db *Database) createDevItems() (ids []int, err error) {

	digitalNewspaperLicense := Item{
		Name:          "Digitale Zeitung (Lizenz)",
		Description:   "Lizenz für digitale Zeitungsausgabe",
		Price:         50,
		Archived:      false,
		IsLicenseItem: true,
		Image:         "img/demo_digital.jpg",
		Type:          "license_item",
	}

	digitalNewspaperLicenseID, err := db.CreateItem(digitalNewspaperLicense)
	if err != nil {
		log.Error("createDevItems: ", err)
		return
	}

	digitalNewspaper := Item{
		Name:         "Digitale Zeitung",
		Description:  "Digitale Zeitungsausgabe",
		Price:        300,
		LicenseItem:  null.NewInt(int64(digitalNewspaperLicenseID), true),
		Archived:     false,
		LicenseGroup: null.NewString("testedition", true),
		Image:        "img/demo_digital.jpg",
		Type:         "issue",
	}

	calendar := Item{
		Name:        "Kalender",
		Description: "Kalender für das Jahr 2024",
		Price:       800,
		Archived:    false,
		Image:       "img/demo_kalender.jpg",
		Type:        "normal_item",
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

	// Create newspaper with PDF
	digitalNewspaperLicense2 := Item{
		Name:          "Digitale Zeitung (Lizenz) 2",
		Description:   "Lizenz für digitale Zeitungsausgabe 2",
		Price:         50,
		Archived:      false,
		IsLicenseItem: true,
		Image:         "img/demo_digital.jpg",
		Type:          "license_item",
	}

	digitalNewspaperLicenseID2, err := db.CreateItem(digitalNewspaperLicense2)
	if err != nil {
		log.Error("createDevItems: ", err)
		return
	}

	pdf := PDF{
		Path:      "test.pdf",
		Timestamp: time.Now(),
	}
	pdfID, err := db.CreatePDF(pdf)
	if err != nil {
		log.Error("createDevItems PDF creation failed: ", err)
		return
	}

	digitalNewspaperWithPDF := Item{
		Name:         "Digitale Zeitung (PDF)",
		Description:  "Digitale Zeitungsausgabe mit PDF",
		Price:        300,
		LicenseItem:  null.NewInt(int64(digitalNewspaperLicenseID2), true),
		Archived:     false,
		LicenseGroup: null.NewString("testedition", true),
		Image:        "img/demo_digital.jpg",
		PDF:          null.NewInt(int64(pdfID), true),
		IsPDFItem:    true,
		Type:         "issue",
	}

	_, err = db.CreateItem(digitalNewspaperWithPDF)
	if err != nil {
		log.Error("Dev newspaper with PDF creation failed ", zap.Error(err))
		return
	}

	return ids, err
}

// After initializing items and calling the function above, the database should look like this:
// item[0] = Newspaper
// item[1] = Donation
// item[2] = Transaction costs
// item[3] = Digital newspaper license
// item[4] = Digital newspaper
// item[5] = Calendar
// item[6] = Digital newspaper license 2
// item[7] = Digital newspaper (PDF)

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

	paypalAccountID, err := db.GetAccountTypeID("Paypal")
	if err != nil {
		return
	}

	vendorAccount, err := db.GetAccountByVendorID(vendorIDs[0])
	if err != nil {
		return
	}

	items, err := db.ListItems(false, false, false)
	if err != nil {
		return
	}

	// Create order
	order := Order{
		OrderCode: null.NewString("devOrder1", true),
		Vendor:    vendorIDs[0],
		Entries: []OrderEntry{
			{
				Item:     items[4].ID, // Digital Newspaper
				Quantity: 2,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},

			// License for newspaper is paid to orga
			{
				Item:     items[3].ID, // Newspaper License
				Quantity: 2,
				Sender:   vendorAccount.ID,
				Receiver: orgaAccountID,
			},
			{
				Item:     items[5].ID, // Calendar
				Quantity: 1,
				Sender:   buyerAccountID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
			{
				Item:     items[1].ID, // Donation
				Quantity: 50,
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
	err = db.CreatePayedOrderEntries(orderID, []OrderEntry{

		// Vendor pays transaction costs
		{
			Item:     items[2].ID, // Transaction cost
			Quantity: 27,          // 27 cents transaction cost
			Sender:   vendorAccount.ID,
			Receiver: paypalAccountID,
		},

		// Vendor gets transaction costs back from orga
		{
			Item:     items[2].ID, // Transaction cost
			Quantity: 27,          // 27 cents transaction cost
			Sender:   orgaAccountID,
			Receiver: vendorAccount.ID,
		},
	})
	if err != nil {
		return
	}

	return
}

// createDevPayout creates a payout for the first dev vendor using their existing sales payments.
func (db *Database) createDevPayout(vendorIDs []int) error {
	if len(vendorIDs) == 0 {
		return nil
	}

	vendor, err := db.GetVendor(vendorIDs[0])
	if err != nil {
		return err
	}

	vendorAccount, err := db.GetAccountByVendorID(vendorIDs[0])
	if err != nil {
		return err
	}

	payments, err := db.ListPaymentsForPayout(time.Now().AddDate(-1, 0, 0), time.Now().AddDate(0, 0, 1), vendor.LicenseID.String)
	if err != nil {
		return err
	}

	if len(payments) == 0 {
		return nil
	}

	total := 0
	for _, p := range payments {
		total += p.Amount
	}

	_, err = db.CreatePaymentPayout(vendor, vendorAccount.ID, "devtools", total, payments)
	return err
}

// createDevCustomersAndAbonements creates sample customers and abonements for development.
func (db *Database) createDevCustomersAndAbonements() error {
	// Use ListItemsWithDisabled so we find abonement items even when they start disabled
	items, err := db.ListItemsWithDisabled(false, false)
	if err != nil {
		return err
	}

	abonementItemID := 0
	for _, it := range items {
		if it.Type == "abonement" {
			abonementItemID = it.ID
			break
		}
	}

	customers := []Customer{
		{
			KeycloakID:    "dev-customer-keycloak-1",
			Email:         "anna.mueller@example.com",
			FirstName:     "Anna",
			LastName:      "Müller",
			LicenseGroups: []string{"digital_edition"},
		},
		{
			KeycloakID: "dev-customer-keycloak-2",
			Email:      "max.mustermann@example.com",
			FirstName:  "Max",
			LastName:   "Mustermann",
		},
		{
			KeycloakID: "dev-customer-keycloak-3",
			Email:      "eva.schmidt@example.com",
			FirstName:  "Eva",
			LastName:   "Schmidt",
		},
	}

	createdIDs := make([]int, 0, len(customers))
	for _, c := range customers {
		created, err := db.CreateCustomer(&c)
		if err != nil {
			log.Error("createDevCustomersAndAbonements: customer creation failed ", zap.Error(err))
			return err
		}
		createdIDs = append(createdIDs, created.ID)
	}

	if abonementItemID == 0 {
		return nil
	}

	now := time.Now()
	abonements := []Abonement{
		{
			CustomerID: createdIDs[0],
			ItemID:     abonementItemID,
			FromDate:   now.AddDate(-1, 0, 0),
			ToDate:     now.AddDate(0, 6, 0),
			Status:     "active",
		},
		{
			CustomerID: createdIDs[1],
			ItemID:     abonementItemID,
			FromDate:   now.AddDate(0, -3, 0),
			ToDate:     now.AddDate(0, 9, 0),
			Status:     "active",
		},
		{
			CustomerID: createdIDs[2],
			ItemID:     abonementItemID,
			FromDate:   now.AddDate(-2, 0, 0),
			ToDate:     now.AddDate(-1, 0, 0),
			Status:     "active",
		},
	}

	for _, a := range abonements {
		_, err := db.CreateAbonement(&a)
		if err != nil {
			log.Error("createDevCustomersAndAbonements: abonement creation failed ", zap.Error(err))
			return err
		}
	}

	return nil
}

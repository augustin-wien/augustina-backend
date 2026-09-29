package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	entorder "github.com/augustin-wien/augustina-backend/ent/order"
	entpayment "github.com/augustin-wien/augustina-backend/ent/payment"
	schema "github.com/augustin-wien/augustina-backend/ent/schema"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// The dev seed tries to cover every state the backoffice and the shop can show,
// so each screen has something to render without clicking data together first:
//
//   - vendors: active with full profile, open balance, negative balance and debt,
//     disabled, blocked, deleted and brand new without any activity
//   - locations: every working time mode, with and without vendor or telephone
//   - comments: warnings and notes, resolved and unresolved
//   - items: every type, disabled, colored and ordered, with PDF, archived
//   - orders: verified (card and PayPal) spread over the last months, pending,
//     invalidated, with customer email, synced to Odoo and failed Odoo sync
//   - payments: online sales, POS sales paid cash and by balance, payouts
//   - customers and abonements: active, expired, future, inactive, cancelled
//   - campaigns: running, scheduled, expired, disabled and without time frame
//   - PDF downloads: unused and downloaded links
//   - blocked IPs: active block, expired block and strikes only

// devVendors holds the IDs of the seeded vendors by role
type devVendors struct {
	active      int // fl-123: full profile, sales, payout, locations, comments
	openBalance int // fl-234: sales that were never paid out
	negative    int // fl-345: bought licenses without selling, owes debt
	disabled    int // fl-456
	blocked     int // fl-567
	deleted     int // fl-678
	fresh       int // fl-789: just registered, no activity at all
}

// devItems holds the IDs of the items the seed sells
type devItems struct {
	newspaper        int
	donation         int
	transactionCosts int
	digitalLicense   int
	digitalNewspaper int
	calendar         int
	pdfNewspaper     int
	bag              int
}

// CreateDevData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	vendors, err := db.createDevVendors()
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevLocations(vendors)
	if err != nil {
		log.Error("Dev data location creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevComments(vendors)
	if err != nil {
		log.Error("Dev data comment creation failed ", zap.Error(err))
		return err
	}
	items, err := db.createDevItems()
	if err != nil {
		log.Error("Dev data item creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevOrdersAndPayments(vendors, items)
	if err != nil {
		log.Error("Dev data order creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevArchivedItems([]int{vendors.active})
	if err != nil {
		log.Error("Dev data archived item creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPayout(vendors.active)
	if err != nil {
		log.Error("Dev data payout creation failed ", zap.Error(err))
		return err
	}
	// Created after the payout so the active vendor also has an open balance
	err = db.createDevRecentSales(vendors, items)
	if err != nil {
		log.Error("Dev data recent sales creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPOSSales(vendors, items)
	if err != nil {
		log.Error("Dev data POS sales creation failed ", zap.Error(err))
		return err
	}
	// Blocked and deleted vendors keep their history, so their state is set
	// only after their sales exist
	err = db.finishDevVendorStates(vendors)
	if err != nil {
		log.Error("Dev data vendor state update failed ", zap.Error(err))
		return err
	}
	err = db.createDevCustomersAndAbonements()
	if err != nil {
		log.Error("Dev data customers/abonements creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevCampaigns(items)
	if err != nil {
		log.Error("Dev data campaign creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPDFDownloads(items)
	if err != nil {
		log.Error("Dev data PDF download creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevBlockedIPs()
	if err != nil {
		log.Error("Dev data blocked IP creation failed ", zap.Error(err))
		return err
	}
	return nil
}

// createDevVendors creates one vendor per vendor state
func (db *Database) createDevVendors() (vendors devVendors, err error) {
	now := time.Now()
	create := func(v Vendor) (int, error) {
		if v.UrlID == "" {
			v.UrlID = "www.augustin.or.at/" + v.LicenseID.String
		}
		id, err := db.CreateVendor(v)
		if err != nil {
			log.Error("Dev data vendor creation failed ", zap.Error(err))
		}
		return id, err
	}

	if vendors.active, err = create(Vendor{
		KeycloakID:       "keycloakid1",
		LicenseID:        null.StringFrom("fl-123"),
		FirstName:        "firstname1",
		LastName:         "lastname1",
		Email:            "test_vendor@example.com",
		Language:         "de",
		Telephone:        "+43 660 123 45 67",
		RegistrationDate: now.AddDate(-3, 0, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-5, 0, 0).Format("2006-01-02"),
		OnlineMap:        true,
		HasSmartphone:    true,
		HasBankAccount:   true,
		AccountProofUrl:  null.StringFrom("https://example.com/account-proof/fl-123.pdf"),
	}); err != nil {
		return
	}
	if vendors.openBalance, err = create(Vendor{
		KeycloakID:       "keycloakid2",
		LicenseID:        null.StringFrom("fl-234"),
		FirstName:        "Recep",
		LastName:         "lastname2",
		Email:            "test_vendor2@example.com",
		Language:         "tr",
		RegistrationDate: now.AddDate(-1, -2, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-1, -2, 0).Format("2006-01-02"),
		OnlineMap:        true,
	}); err != nil {
		return
	}
	if vendors.negative, err = create(Vendor{
		KeycloakID:       "keycloakid3",
		LicenseID:        null.StringFrom("fl-345"),
		FirstName:        "Ioana",
		LastName:         "Popescu",
		Email:            "test_vendor3@example.com",
		Language:         "ro",
		Telephone:        "+43 699 987 65 43",
		RegistrationDate: now.AddDate(0, -8, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(0, -8, 0).Format("2006-01-02"),
		HasSmartphone:    true,
		Debt:             "25,00 € Vorschuss (Rückzahlung in Raten)",
	}); err != nil {
		return
	}
	if vendors.disabled, err = create(Vendor{
		KeycloakID:       "keycloakid4",
		LicenseID:        null.StringFrom("fl-456"),
		FirstName:        "Kofi",
		LastName:         "Mensah",
		Email:            "test_vendor4@example.com",
		Language:         "en",
		RegistrationDate: now.AddDate(-4, 0, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-4, 0, 0).Format("2006-01-02"),
		IsDisabled:       true,
	}); err != nil {
		return
	}
	if vendors.blocked, err = create(Vendor{
		KeycloakID:       "keycloakid5",
		LicenseID:        null.StringFrom("fl-567"),
		FirstName:        "Stefan",
		LastName:         "Novak",
		Email:            "test_vendor5@example.com",
		Language:         "de",
		RegistrationDate: now.AddDate(-2, 0, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-2, 0, 0).Format("2006-01-02"),
		HasSmartphone:    true,
		HasBankAccount:   true,
	}); err != nil {
		return
	}
	if vendors.deleted, err = create(Vendor{
		KeycloakID: "keycloakid6",
		LicenseID:  null.StringFrom("fl-678"),
		FirstName:  "Gerda",
		LastName:   "Weiss",
		Email:      "test_vendor6@example.com",
		Language:   "de",
	}); err != nil {
		return
	}
	// Only the fields the vendor creation form requires
	vendors.fresh, err = create(Vendor{
		KeycloakID: "keycloakid7",
		LicenseID:  null.StringFrom("fl-789"),
		FirstName:  "Amira",
		LastName:   "Haddad",
		Email:      "test_vendor7@example.com",
	})
	return
}

// finishDevVendorStates blocks and deletes the vendors meant to end up that way
func (db *Database) finishDevVendorStates(vendors devVendors) error {
	ctx := context.Background()
	err := db.EntClient.Vendor.UpdateOneID(vendors.blocked).
		SetIsblocked(true).
		SetBlockednote("Ausweis abgelaufen, bitte neuen Ausweis vorlegen").
		Exec(ctx)
	if err != nil {
		return err
	}
	return db.EntClient.Vendor.UpdateOneID(vendors.deleted).
		SetIsdeleted(true).
		Exec(ctx)
}

func (db *Database) createDevLocations(vendors devVendors) (err error) {
	byVendor := map[int][]ent.Location{
		vendors.active: {
			{
				Name:      "Morning Market",
				Address:   "Marktplatz 1",
				Longitude: 16.3725,
				Latitude:  48.2082,
				Zip:       "1010",
				Telephone: "+43 1 512 34 56",
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
						"wed": {{From: "09:00", To: "12:00"}, {From: "14:00", To: "18:00"}},
						"thu": {{From: "09:00", To: "17:00"}},
						"fri": {{From: "09:00", To: "17:00"}},
						"sat": {{FullDay: true}},
					},
				},
			},
		},
		vendors.openBalance: {
			{
				Name:        "Bahnhof Meidling",
				Address:     "Eichenstraße 25",
				Longitude:   16.3337,
				Latitude:    48.1745,
				Zip:         "1120",
				WorkingTime: &schema.WorkingTime{Mode: "whole_week", WholeWeek: true},
			},
		},
		vendors.negative: {
			{
				Name:      "Supermarkt Favoriten",
				Address:   "Favoritenstraße 120",
				Longitude: 16.3772,
				Latitude:  48.1823,
				Zip:       "1100",
				Telephone: "+43 699 987 65 43",
				WorkingTime: &schema.WorkingTime{
					Mode: "custom",
					WeekDays: map[string][]schema.TimeRange{
						"fri": {{From: "15:00", To: "19:00"}},
						"sat": {{From: "08:00", To: "13:00"}},
					},
				},
			},
		},
		vendors.disabled: {
			{
				Name:        "Stephansplatz",
				Address:     "Stephansplatz 3",
				Longitude:   16.3731,
				Latitude:    48.2085,
				Zip:         "1010",
				WorkingTime: &schema.WorkingTime{},
			},
		},
	}

	for vendorID, locations := range byVendor {
		for _, location := range locations {
			err = db.CreateLocation(vendorID, location)
			if err != nil {
				log.Error("Dev data location creation failed ", zap.Error(err))
				return err
			}
		}
	}

	// A location nobody is assigned to yet
	_, err = db.CreateStandaloneLocation(nil, ent.Location{
		Name:      "Westbahnhof (unbesetzt)",
		Address:   "Europaplatz 2",
		Longitude: 16.3378,
		Latitude:  48.1966,
		Zip:       "1150",
		WorkingTime: &schema.WorkingTime{
			Mode:     "everyday",
			Everyday: []schema.TimeRange{{From: "07:00", To: "10:00"}},
		},
	})
	return err
}

func (db *Database) createDevComments(vendors devVendors) (err error) {
	now := time.Now()
	var unresolved time.Time
	byVendor := map[int][]ent.Comment{
		vendors.active: {
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
			{
				Comment:    "Neue Fotos für den Ausweis mitbringen.",
				Warning:    false,
				CreatedAt:  now.Add(-2 * time.Hour),
				ResolvedAt: unresolved,
			},
		},
		vendors.negative: {
			{
				Comment:    "Vorschuss von 25 € ausgezahlt, Rückzahlung vereinbart.",
				Warning:    true,
				CreatedAt:  now.AddDate(0, -1, 0),
				ResolvedAt: unresolved,
			},
		},
		vendors.blocked: {
			{
				Comment:    "Ausweis abgelaufen, bis zur Verlängerung gesperrt.",
				Warning:    true,
				CreatedAt:  now.AddDate(0, 0, -3),
				ResolvedAt: unresolved,
			},
		},
	}

	for vendorID, comments := range byVendor {
		for _, comment := range comments {
			err = db.CreateVendorComment(vendorID, comment)
			if err != nil {
				log.Error("Dev data comment creation failed ", zap.Error(err))
				return err
			}
		}
	}
	return nil
}

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

// saleEntries returns the order entries of a QR code sale, including the
// license the vendor pays to the organisation for licensed items
func (db *Database) devSaleEntries(items devItems, buyer, vendorAccount, orga int, digital, calendars, newspapers, donation int) []OrderEntry {
	var entries []OrderEntry
	add := func(item, quantity, sender, receiver int, isSale bool) {
		if item == 0 || quantity == 0 {
			return
		}
		entries = append(entries, OrderEntry{
			Item:     item,
			Quantity: quantity,
			Sender:   sender,
			Receiver: receiver,
			IsSale:   isSale,
		})
	}
	add(items.digitalNewspaper, digital, buyer, vendorAccount, true)
	add(items.digitalLicense, digital, vendorAccount, orga, false)
	add(items.calendar, calendars, buyer, vendorAccount, true)
	add(items.newspaper, newspapers, buyer, vendorAccount, true)
	add(items.donation, donation, buyer, vendorAccount, false)
	return entries
}

// devSale describes one QR code sale of the seed
type devSale struct {
	vendor     int
	code       string
	daysAgo    int
	digital    int
	calendars  int
	newspapers int
	donation   int
	paypal     bool
}

// createDevSale creates, verifies and backdates a QR code sale, including the
// transaction costs the organisation covers
func (db *Database) createDevSale(items devItems, s devSale) (orderID int, err error) {
	buyer, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}
	provider, err := db.GetAccountTypeID("VivaWallet")
	if err != nil {
		return
	}
	transactionType := 0
	if s.paypal {
		if provider, err = db.GetAccountTypeID("Paypal"); err != nil {
			return
		}
		transactionType = config.Config.VivaWalletTransactionTypeIDPaypal
	}
	vendorAccount, err := db.GetAccountByVendorID(s.vendor)
	if err != nil {
		return
	}

	order := Order{
		OrderCode:     null.StringFrom(s.code),
		TransactionID: "dev-" + s.code,
		Vendor:        s.vendor,
		Entries:       db.devSaleEntries(items, buyer, vendorAccount.ID, orga, s.digital, s.calendars, s.newspapers, s.donation),
	}
	if orderID, err = db.CreateOrder(order); err != nil {
		return
	}
	if err = db.VerifyOrderAndCreatePayments(orderID, transactionType); err != nil {
		return
	}
	if items.transactionCosts != 0 {
		err = db.CreatePayedOrderEntries(orderID, []OrderEntry{
			// Vendor pays the payment provider's transaction costs...
			{Item: items.transactionCosts, Quantity: 27, Sender: vendorAccount.ID, Receiver: provider},
			// ...and gets them back from the organisation
			{Item: items.transactionCosts, Quantity: 27, Sender: orga, Receiver: vendorAccount.ID},
		})
		if err != nil {
			return
		}
	}
	err = db.backdateDevOrder(orderID, time.Now().AddDate(0, 0, -s.daysAgo))
	return
}

// backdateDevOrder moves an order and its payments into the past, so the
// statistics have more than a single day to show
func (db *Database) backdateDevOrder(orderID int, at time.Time) error {
	ctx := context.Background()
	at = at.UTC()
	err := db.EntClient.Order.UpdateOneID(orderID).SetTimestamp(at).Exec(ctx)
	if err != nil {
		return err
	}
	err = db.EntClient.Order.Update().
		Where(entorder.ID(orderID), entorder.Verified(true)).
		SetVerifiedAt(at.Add(2 * time.Minute)).
		Exec(ctx)
	if err != nil {
		return err
	}
	return db.EntClient.Payment.Update().
		Where(entpayment.OrderID(orderID)).
		SetTimestamp(at.Add(2 * time.Minute)).
		Exec(ctx)
}

// createDevOrdersAndPayments creates the sales history before the payout: verified
// orders over the last months plus the order states that never lead to payments
func (db *Database) createDevOrdersAndPayments(vendors devVendors, items devItems) (err error) {
	sales := []devSale{
		// The original dev order: 2 digital newspapers, 1 calendar and a donation
		{vendor: vendors.active, code: "devOrder1", daysAgo: 150, digital: 2, calendars: 1, donation: 50},
		{vendor: vendors.active, code: "devOrder2", daysAgo: 120, newspapers: 3},
		{vendor: vendors.active, code: "devOrder3", daysAgo: 95, digital: 1, donation: 100, paypal: true},
		{vendor: vendors.active, code: "devOrder4", daysAgo: 60, calendars: 2},
		{vendor: vendors.active, code: "devOrder5", daysAgo: 35, newspapers: 1, digital: 1, paypal: true},
		{vendor: vendors.active, code: "devOrder6", daysAgo: 14, newspapers: 2, donation: 20},
		{vendor: vendors.openBalance, code: "devOrder7", daysAgo: 80, newspapers: 2, calendars: 1},
		{vendor: vendors.openBalance, code: "devOrder8", daysAgo: 25, digital: 3, paypal: true},
		{vendor: vendors.disabled, code: "devOrder9", daysAgo: 170, newspapers: 4},
		{vendor: vendors.blocked, code: "devOrder10", daysAgo: 45, calendars: 1, donation: 30},
		{vendor: vendors.deleted, code: "devOrder11", daysAgo: 160, newspapers: 1},
	}
	var synced, failed int
	for _, s := range sales {
		var orderID int
		if orderID, err = db.createDevSale(items, s); err != nil {
			return
		}
		switch s.code {
		case "devOrder4":
			synced = orderID
		case "devOrder6":
			failed = orderID
		}
	}

	// Odoo export states
	if err = db.SetOdooSyncSuccess(synced); err != nil {
		return
	}
	if err = db.SetOdooSyncFailure(failed, errors.New("odoo: 502 Bad Gateway")); err != nil {
		return
	}

	buyer, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}
	vendorAccount, err := db.GetAccountByVendorID(vendors.active)
	if err != nil {
		return
	}

	// Abandoned checkout, given up on by the order reconciliation
	abandonedID, err := db.CreateOrder(Order{
		OrderCode: null.StringFrom("devOrderAbandoned"),
		Vendor:    vendors.active,
		Entries:   db.devSaleEntries(items, buyer, vendorAccount.ID, orga, 0, 1, 0, 0),
	})
	if err != nil {
		return
	}
	if err = db.backdateDevOrder(abandonedID, time.Now().AddDate(0, 0, -2)); err != nil {
		return
	}
	if err = db.InvalidateOrder(abandonedID); err != nil {
		return
	}

	// Checkout still in progress; the reconciliation invalidates it after a while
	_, err = db.CreateOrder(Order{
		OrderCode: null.StringFrom("devOrderPending"),
		Vendor:    vendors.openBalance,
		Entries:   db.devSaleEntries(items, buyer, vendorAccount.ID, orga, 0, 0, 2, 10),
	})
	return
}

// createDevRecentSales creates sales after the payout, so vendors have open balances
func (db *Database) createDevRecentSales(vendors devVendors, items devItems) (err error) {
	sales := []devSale{
		{vendor: vendors.active, code: "devOrderRecent1", daysAgo: 3, newspapers: 2, calendars: 1},
		{vendor: vendors.active, code: "devOrderRecent2", daysAgo: 1, digital: 1, donation: 50, paypal: true},
	}
	for _, s := range sales {
		if _, err = db.createDevSale(items, s); err != nil {
			return
		}
	}
	// Online PDF purchase: the customer gets the download link by email
	if _, err = db.createDevPDFSale(vendors.openBalance, items); err != nil {
		return
	}

	// A vendor who bought licenses without selling anything ends up with a
	// negative balance
	vendorAccount, err := db.GetAccountByVendorID(vendors.negative)
	if err != nil {
		return
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}
	_, err = db.CreatePayment(Payment{
		Sender:       vendorAccount.ID,
		Receiver:     orga,
		Amount:       150,
		AuthorizedBy: "devtools",
		Item:         null.IntFrom(int64(items.digitalLicense)),
		Quantity:     3,
		Price:        50,
	})
	return
}

// createDevPDFSale sells the PDF newspaper online with a customer email, which
// creates the PDF download link
func (db *Database) createDevPDFSale(vendorID int, items devItems) (orderID int, err error) {
	buyer, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}
	vendorAccount, err := db.GetAccountByVendorID(vendorID)
	if err != nil {
		return
	}
	orderID, err = db.CreateOrder(Order{
		OrderCode:     null.StringFrom("devOrderPDF"),
		TransactionID: "dev-devOrderPDF",
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom("pdf.kaeuferin@example.com"),
		Entries: []OrderEntry{{
			Item:     items.pdfNewspaper,
			Quantity: 1,
			Sender:   buyer,
			Receiver: vendorAccount.ID,
			IsSale:   true,
		}},
	})
	if err != nil {
		return
	}
	err = db.VerifyOrderAndCreatePayments(orderID, 0)
	return
}

// createDevPOSSales creates sales at the backoffice point of sale, paid in cash
// and with the vendor's balance
func (db *Database) createDevPOSSales(vendors devVendors, items devItems) error {
	cash, err := db.GetAccountTypeID("Cash")
	if err != nil {
		return err
	}
	backoffice, err := db.GetAccountTypeID("Backoffice")
	if err != nil {
		return err
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return err
	}

	posSale := func(vendorID int, balancePortion int, entries map[int]int) error {
		vendorAccount, err := db.GetAccountByVendorID(vendorID)
		if err != nil {
			return err
		}
		total := 0
		var payments []Payment
		for itemID, quantity := range entries {
			item, err := db.GetItem(itemID)
			if err != nil {
				return err
			}
			total += item.Price * quantity
			// Bookkeeping record, doesn't move money (see handlers.pos.go)
			payments = append(payments, Payment{
				Sender:       vendorAccount.ID,
				Receiver:     backoffice,
				Amount:       item.Price * quantity,
				AuthorizedBy: "devtools",
				IsSale:       true,
				IsPOS:        true,
				Item:         null.IntFrom(int64(itemID)),
				Quantity:     quantity,
				Price:        item.Price,
			})
		}
		if balancePortion > total {
			balancePortion = total
		}
		if balancePortion > 0 {
			payments = append(payments,
				Payment{Sender: vendorAccount.ID, Receiver: orga, Amount: balancePortion, AuthorizedBy: "devtools", IsPOS: true, Quantity: 1, Price: balancePortion},
				Payment{Sender: orga, Receiver: backoffice, Amount: balancePortion, AuthorizedBy: "devtools", IsPOS: true, Quantity: 1, Price: balancePortion},
			)
		}
		if cashPortion := total - balancePortion; cashPortion > 0 {
			payments = append(payments, Payment{Sender: cash, Receiver: backoffice, Amount: cashPortion, AuthorizedBy: "devtools", IsPOS: true, Quantity: 1, Price: cashPortion})
		}
		return db.CreatePayments(payments)
	}

	// Paid in cash
	if err = posSale(vendors.active, 0, map[int]int{items.newspaper: 10}); err != nil {
		return err
	}
	// Paid with part of the vendor's balance, the rest in cash
	return posSale(vendors.openBalance, 500, map[int]int{items.newspaper: 5, items.bag: 1})
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

// createDevPayout pays out everything the vendor has earned so far
func (db *Database) createDevPayout(vendorID int) error {
	vendor, err := db.GetVendor(vendorID)
	if err != nil {
		return err
	}

	vendorAccount, err := db.GetAccountByVendorID(vendorID)
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
	if err != nil {
		return err
	}
	return db.EntClient.Vendor.UpdateOneID(vendorID).
		SetLastpayout(time.Now().AddDate(0, 0, -7)).
		Exec(context.Background())
}

// createDevCustomersAndAbonements creates customers in every abonement state
func (db *Database) createDevCustomersAndAbonements() error {
	// Abonement items may start disabled, so look them up including disabled ones
	abonementItemID, err := db.devItemIDByType("abonement")
	if err != nil {
		return err
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
		{
			KeycloakID:    "dev-customer-keycloak-4",
			Email:         "jonas.berger@example.com",
			FirstName:     "Jonas",
			LastName:      "Berger",
			LicenseGroups: []string{"digital_edition", "testedition"},
		},
		{
			KeycloakID: "dev-customer-keycloak-5",
			Email:      "lena.gruber@example.com",
			FirstName:  "Lena",
			LastName:   "Gruber",
		},
		{
			KeycloakID: "dev-customer-keycloak-6",
			Email:      "paul.wagner@example.com",
			FirstName:  "Paul",
			LastName:   "Wagner",
		},
		// Registered without ever buying anything, name unknown
		{
			KeycloakID: "dev-customer-keycloak-7",
			Email:      "nur.email@example.com",
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
		// Running
		{CustomerID: createdIDs[0], FromDate: now.AddDate(-1, 0, 0), ToDate: now.AddDate(0, 6, 0), Status: "active"},
		{CustomerID: createdIDs[1], FromDate: now.AddDate(0, -3, 0), ToDate: now.AddDate(0, 9, 0), Status: "active"},
		// Still marked active, but ran out
		{CustomerID: createdIDs[2], FromDate: now.AddDate(-2, 0, 0), ToDate: now.AddDate(-1, 0, 0), Status: "active"},
		// Renewed: the old one ran out, the new one runs
		{CustomerID: createdIDs[3], FromDate: now.AddDate(-2, 0, 0), ToDate: now.AddDate(-1, 0, 0), Status: "inactive"},
		{CustomerID: createdIDs[3], FromDate: now.AddDate(-1, 0, 0), ToDate: now.AddDate(0, 0, 10), Status: "active"},
		// Starts in the future
		{CustomerID: createdIDs[4], FromDate: now.AddDate(0, 0, 14), ToDate: now.AddDate(1, 0, 14), Status: "active"},
		// Cancelled before running out
		{CustomerID: createdIDs[5], FromDate: now.AddDate(0, -6, 0), ToDate: now.AddDate(0, 6, 0), Status: "cancelled"},
	}

	for _, a := range abonements {
		a.ItemID = abonementItemID
		_, err := db.CreateAbonement(&a)
		if err != nil {
			log.Error("createDevCustomersAndAbonements: abonement creation failed ", zap.Error(err))
			return err
		}
	}

	return nil
}

// createDevCampaigns creates a campaign for every schedule state
func (db *Database) createDevCampaigns(items devItems) error {
	now := time.Now()
	at := func(days int) *time.Time {
		t := now.AddDate(0, 0, days)
		return &t
	}
	campaigns := []struct {
		campaign      ent.Campaign
		views, clicks int
	}{
		{ent.Campaign{Name: "Kalender-Aktion", ItemID: items.calendar, Title: "Der neue Kalender ist da!", Text: "Jetzt den Augustin-Kalender sichern und Verkäufer:innen unterstützen.", StartsAt: at(-10), EndsAt: at(20), Enabled: true}, 340, 27},
		{ent.Campaign{Name: "Dauerwerbung Digitale Zeitung", ItemID: items.digitalNewspaper, Title: "Lies uns auch digital", Text: "Die Zeitung als digitale Ausgabe, direkt nach dem Kauf.", Enabled: true}, 1250, 88},
		{ent.Campaign{Name: "Weihnachtsaktion", ItemID: items.bag, Title: "Das Geschenk mit Sinn", Text: "Die Stofftasche als Weihnachtsgeschenk.", StartsAt: at(30), EndsAt: at(60), Enabled: true}, 0, 0},
		{ent.Campaign{Name: "Sommeraktion (vorbei)", ItemID: items.newspaper, Title: "Sommerausgabe", Text: "Die Doppelnummer für den Sommer.", StartsAt: at(-90), EndsAt: at(-30), Enabled: true}, 2100, 64},
		{ent.Campaign{Name: "Entwurf Spendenaufruf", ItemID: items.donation, Title: "Spenden hilft", Text: "Noch nicht freigegeben.", Enabled: false}, 0, 0},
	}

	for _, c := range campaigns {
		if c.campaign.ItemID == 0 {
			continue
		}
		created, err := db.CreateCampaign(&c.campaign)
		if err != nil {
			return err
		}
		err = db.EntClient.Campaign.UpdateOneID(created.ID).
			SetViews(c.views).
			SetClicks(c.clicks).
			Exec(context.Background())
		if err != nil {
			return err
		}
	}
	return nil
}

// createDevPDFDownloads adds a used download link next to the unused one the PDF
// sale created
func (db *Database) createDevPDFDownloads(items devItems) error {
	item, err := db.GetItem(items.pdfNewspaper)
	if err != nil {
		return err
	}
	download, err := db.CreatePDFDownloadForItem(int(item.PDF.Int64), item.ID)
	if err != nil {
		return err
	}
	download.DownloadCount = 3
	download.LastDownload = time.Now().Add(-5 * time.Hour)
	download.ItemID = null.IntFrom(int64(item.ID))
	return db.UpdatePdfDownload(download)
}

// createDevBlockedIPs creates an active block, an expired block and an IP that
// only collected strikes
func (db *Database) createDevBlockedIPs() error {
	ctx := context.Background()
	now := time.Now()
	err := db.EntClient.BlockedIP.Create().
		SetIP("203.0.113.10").
		SetStrikes(0).
		SetBlockExpiresAt(now.Add(12 * time.Hour)).
		SetReason("too many invalid vendor IDs").
		Exec(ctx)
	if err != nil {
		return err
	}
	err = db.EntClient.BlockedIP.Create().
		SetIP("203.0.113.20").
		SetStrikes(0).
		SetBlockExpiresAt(now.AddDate(0, 0, -2)).
		SetReason("too many invalid vendor IDs").
		Exec(ctx)
	if err != nil {
		return err
	}
	return db.EntClient.BlockedIP.Create().
		SetIP("198.51.100.7").
		SetStrikes(2).
		Exec(ctx)
}

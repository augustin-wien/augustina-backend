package database

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// brokenDatabase returns a database whose connection is closed, so every query fails
func brokenDatabase(t *testing.T) *Database {
	t.Helper()
	pdb, err := sql.Open("postgres", "postgres://invalid@127.0.0.1:1/invalid?sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, pdb.Close())
	return &Database{
		EntClient: ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, pdb))),
		DB:        pdb,
	}
}

// TestQueriesReturnDatabaseErrors checks that the queries pass database
// failures on to the caller instead of swallowing them.
func TestQueriesReturnDatabaseErrors(t *testing.T) {
	db := brokenDatabase(t)
	now := time.Now()
	vendor := Vendor{ID: 1, LicenseID: null.StringFrom("dbfail-1"), Email: "dbfail@example.com"}
	item := Item{ID: 1, Name: "x", Description: "y", Price: 100, Type: "normal_item"}

	errOf := func(_ any, err error) error { return err }
	errOnly := func(err error) error { return err }

	cases := map[string]func() error{
		// Campaigns
		"ListCampaigns":       func() error { return errOf(db.ListCampaigns()) },
		"ListActiveCampaigns": func() error { return errOf(db.ListActiveCampaigns(now)) },
		"CreateCampaign":      func() error { return errOf(db.CreateCampaign(&ent.Campaign{Name: "c"})) },
		"UpdateCampaign":      func() error { return errOf(db.UpdateCampaign(1, &ent.Campaign{Name: "c"})) },
		"DeleteCampaign":      func() error { return db.DeleteCampaign(1) },
		"TrackCampaign":       func() error { return errOf(db.TrackCampaign(1, CampaignView, now)) },
		// Accounts
		"ListAccounts":                       func() error { return errOf(db.ListAccounts()) },
		"GetAccountByID":                     func() error { return errOf(db.GetAccountByID(1)) },
		"GetAccountTypeID":                   func() error { return errOf(db.GetAccountTypeID("Cash")) },
		"GetAccountByType":                   func() error { return errOf(db.GetAccountByType("Cash")) },
		"GetAccountByVendorID":               func() error { return errOf(db.GetAccountByVendorID(1)) },
		"UpdateAccountBalanceByOpenPayments": func() error { return errOf(db.UpdateAccountBalanceByOpenPayments(1)) },
		"RecalculateAllVendorBalances":       func() error { return db.RecalculateAllVendorBalances() },
		"CreateSpecialVendorAccount":         func() error { return errOf(db.CreateSpecialVendorAccount(vendor)) },
		// Abonements
		"CreateAbonement":           func() error { return errOf(db.CreateAbonement(&Abonement{CustomerID: 1})) },
		"GetAbonementByID":          func() error { return errOf(db.GetAbonementByID(1)) },
		"ListAbonements":            func() error { return errOf(db.ListAbonements()) },
		"ListAbonementsByCustomer":  func() error { return errOf(db.ListAbonementsByCustomer(1)) },
		"UpdateAbonement":           func() error { return errOf(db.UpdateAbonement(&Abonement{ID: 1})) },
		"DeleteAbonement":           func() error { return db.DeleteAbonement(1) },
		"GetActiveAbonementsByDate": func() error { return errOf(db.GetActiveAbonementsByDate(now)) },
		"GetAbonementsByDateRange":  func() error { return errOf(db.GetAbonementsByDateRange(now, now)) },
		// Orders
		"GetOrderEntries":              func() error { return errOf(db.GetOrderEntries(1)) },
		"DeleteOrderEntry":             func() error { return db.DeleteOrderEntry(1) },
		"GetOrders":                    func() error { return errOf(db.GetOrders()) },
		"GetUnverifiedOrders":          func() error { return errOf(db.GetUnverifiedOrders()) },
		"InvalidateOrder":              func() error { return db.InvalidateOrder(1) },
		"GetOrderStatistics":           func() error { return errOf(db.GetOrderStatistics(now, now)) },
		"GetOrderByID":                 func() error { return errOf(db.GetOrderByID(1)) },
		"GetOrderByOrderCode":          func() error { return errOf(db.GetOrderByOrderCode("1")) },
		"CreateOrder":                  func() error { return errOf(db.CreateOrder(Order{Vendor: 1})) },
		"DeleteOrder":                  func() error { return db.DeleteOrder(1) },
		"SetOrderTransactionID":        func() error { return db.SetOrderTransactionID(1, "tx") },
		"SetOrderPaymentFailure":       func() error { return errOf(db.SetOrderPaymentFailure("1", 1)) },
		"SetOdooSyncSuccess":           func() error { return db.SetOdooSyncSuccess(1) },
		"SetOdooSyncFailure":           func() error { return db.SetOdooSyncFailure(1, errors.New("x")) },
		"GetVerifiedOrders":            func() error { return errOf(db.GetVerifiedOrders(now, now)) },
		"VerifyOrderAndCreatePayments": func() error { return db.VerifyOrderAndCreatePayments(1, 1) },
		"CreatePayedOrderEntries":      func() error { return errOnly(db.CreatePayedOrderEntries(1, []OrderEntry{{Item: 1}})) },
		"ResendOrderMails":             func() error { return errOf(db.ResendOrderMails(1)) },
		// Mail templates
		"GetMailTemplateByName":         func() error { return errOf(db.GetMailTemplateByName("x")) },
		"ListMailTemplates":             func() error { return errOf(db.ListMailTemplates()) },
		"CreateOrUpdateMailTemplate":    func() error { return db.CreateOrUpdateMailTemplate("x", "s", "b") },
		"DeleteMailTemplate":            func() error { return db.DeleteMailTemplate("x") },
		"BuildEmailRequestFromTemplate": func() error { return errOf(db.BuildEmailRequestFromTemplate("x", []string{"a@example.com"}, nil)) },
		// Vendors
		"GetVendorByLicenseID":                func() error { return errOf(db.GetVendorByLicenseID("x")) },
		"RecordVendorIDScan":                  func() error { return errOf(db.RecordVendorIDScan("x")) },
		"ListVendors":                         func() error { return errOf(db.ListVendors()) },
		"ListVendorsWithDisabled":             func() error { return errOf(db.ListVendorsWithDisabled()) },
		"GetVendorByLicenseIDWithoutDisabled": func() error { return errOf(db.GetVendorByLicenseIDWithoutDisabled("x")) },
		"GetVendorByEmail":                    func() error { return errOf(db.GetVendorByEmail("x@example.com")) },
		"GetVendorSimple":                     func() error { return errOf(db.GetVendorSimple(1)) },
		"GetVendor":                           func() error { return errOf(db.GetVendor(1)) },
		"GetVendorWithBalanceUpdate":          func() error { return errOf(db.GetVendorWithBalanceUpdate(1)) },
		"CreateVendor":                        func() error { return errOf(db.CreateVendor(vendor)) },
		"UpdateVendor":                        func() error { return db.UpdateVendor(1, vendor) },
		"DeleteVendor":                        func() error { return db.DeleteVendor(1) },
		"GetVendorComments":                   func() error { return errOf(db.GetVendorComments(1)) },
		"CreateVendorComment":                 func() error { return db.CreateVendorComment(1, ent.Comment{Comment: "x"}) },
		"UpdateVendorComment":                 func() error { return db.UpdateVendorComment(1, ent.Comment{ID: 1, Comment: "x"}) },
		"DeleteVendorComment":                 func() error { return db.DeleteVendorComment(1, 1) },
		// Customers
		"CreateCustomer":                func() error { return errOf(db.CreateCustomer(&Customer{Email: "a@example.com"})) },
		"GetCustomerByID":               func() error { return errOf(db.GetCustomerByID(1)) },
		"GetCustomerByKeycloakID":       func() error { return errOf(db.GetCustomerByKeycloakID("x")) },
		"ListCustomers":                 func() error { return errOf(db.ListCustomers()) },
		"UpdateCustomer":                func() error { return errOf(db.UpdateCustomer(&Customer{ID: 1})) },
		"SetCustomerKeycloakID":         func() error { return db.SetCustomerKeycloakID(1, "x") },
		"DeleteCustomer":                func() error { return db.DeleteCustomer(1) },
		"GetCustomerByEmail":            func() error { return errOf(db.GetCustomerByEmail("a@example.com")) },
		"GetOrCreateCustomerByEmail":    func() error { return errOf(db.GetOrCreateCustomerByEmail("a@example.com", "x")) },
		"AddLicenseGroupToCustomer":     func() error { return errOf(db.AddLicenseGroupToCustomer(1, "x")) },
		"SubscribeCustomerToNewsletter": func() error { return db.SubscribeCustomerToNewsletter(1) },
		// Payments
		"ListPayments":            func() error { return errOf(db.ListPayments(now, now, "", false, false, false, false, false)) },
		"ListPaymentsForCustomer": func() error { return errOf(db.ListPaymentsForCustomer("a@example.com")) },
		"ListPaymentsForPayout":   func() error { return errOf(db.ListPaymentsForPayout(now, now, "")) },
		"GetPayment":              func() error { return errOf(db.GetPayment(1)) },
		"CreatePayment":           func() error { return errOf(db.CreatePayment(Payment{Sender: 1, Receiver: 2, Amount: 1})) },
		"CreatePayments":          func() error { return db.CreatePayments([]Payment{{Sender: 1, Receiver: 2, Amount: 1}}) },
		"CreatePaymentPayout":     func() error { return errOf(db.CreatePaymentPayout(vendor, 1, "admin", 100, nil)) },
		"ListPOSOrdersForVendor":  func() error { return errOf(db.ListPOSOrdersForVendor("x", 10)) },
		"ListAllPOSOrders":        func() error { return errOf(db.ListAllPOSOrders(now, now)) },
		"DeletePayment":           func() error { return db.DeletePayment(1) },
		"GetHelloWorld":           func() error { return errOf(db.GetHelloWorld()) },
		"RecordExternalPayment":   func() error { return db.RecordExternalPayment(ExternalPaymentUpdate{TransactionID: "tx"}) },
		"ListExternalPayments":    func() error { return errOf(db.ListExternalPayments(now, now)) },
		// Locations
		"GetLocationsByVendorID":   func() error { return errOf(db.GetLocationsByVendorID(1)) },
		"CreateLocation":           func() error { return db.CreateLocation(1, ent.Location{Name: "x"}) },
		"UpdateLocation":           func() error { return db.UpdateLocation(1, ent.Location{ID: 1, Name: "x"}) },
		"DeleteLocation":           func() error { return db.DeleteLocation(1, 1) },
		"GetVendorLocations":       func() error { return errOf(db.GetVendorLocations()) },
		"ListLocationsWithVendor":  func() error { return errOf(db.ListLocationsWithVendor()) },
		"CreateStandaloneLocation": func() error { return errOf(db.CreateStandaloneLocation(nil, ent.Location{Name: "x"})) },
		"UpdateLocationByID":       func() error { return errOnly(db.UpdateLocationByID(nil, ent.Location{ID: 1, Name: "x"})) },
		"DeleteLocationByID":       func() error { return db.DeleteLocationByID(1) },
		// Items
		"ListItems":                      func() error { return errOf(db.ListItems(false, false, false)) },
		"ListItemsWithDisabled":          func() error { return errOf(db.ListItemsWithDisabled(false, false)) },
		"ListLicenseGroups":              func() error { return errOf(db.ListLicenseGroups()) },
		"ListItemsShop":                  func() error { return errOf(db.ListItemsShop()) },
		"GetItemByName":                  func() error { return errOf(db.GetItemByName("x")) },
		"GetItem":                        func() error { return errOf(db.GetItem(1)) },
		"GetItemIncludingDisabled":       func() error { return errOf(db.GetItemIncludingDisabled(1)) },
		"GetItemByLicenseID":             func() error { _, _, err := db.GetItemByLicenseID(1); return err },
		"GetCurrentOnlineIssue":          func() error { _, _, err := db.GetCurrentOnlineIssue(); return err },
		"CreateItem":                     func() error { return errOf(db.CreateItem(item)) },
		"CreateOnlineIssueWithLicense":   func() error { _, _, err := db.CreateOnlineIssueWithLicense(item, 100); return err },
		"CreateAbonementItemWithLicense": func() error { _, _, err := db.CreateAbonementItemWithLicense(item, 100); return err },
		"UpdateItem":                     func() error { return db.UpdateItem(1, item) },
		"DeleteItem":                     func() error { return db.DeleteItem(1) },
		"DeleteItemChecked":              func() error { return db.DeleteItemChecked(1, true) },
		"ListArchivedItems":              func() error { return errOf(db.ListArchivedItems()) },
		"ListItemsIncludingArchived":     func() error { return errOf(db.ListItemsIncludingArchived(false, false)) },
		"GetItemName":                    func() error { return errOf(db.GetItemName(1)) },
		"GetItemIncludingArchived":       func() error { return errOf(db.GetItemIncludingArchived(1)) },
		"RestoreItem":                    func() error { return db.RestoreItem(1) },
		// Settings
		"GetSettings":                  func() error { return errOf(db.GetSettings()) },
		"UpdateSettings":               func() error { return db.UpdateSettings(&ent.Settings{ID: 1}) },
		"UpdateDBSettings":             func() error { return db.UpdateDBSettings(DBSettings{}) },
		"GetDBSettings":                func() error { return errOf(db.GetDBSettings()) },
		"GetAccountingReportSentMonth": func() error { return errOf(db.GetAccountingReportSentMonth()) },
		"SetAccountingReportSentMonth": func() error { return db.SetAccountingReportSentMonth("2026-01") },
		// Dev data
		"CreateDevData":        func() error { return db.CreateDevData() },
		"SeedDevArchivedItems": func() error { return db.SeedDevArchivedItems("x") },
		// PDFs
		"DeletePDF":                func() error { return db.DeletePDF() },
		"CreatePDF":                func() error { return errOf(db.CreatePDF(PDF{Path: "x"})) },
		"GetPDF":                   func() error { return errOf(db.GetPDF()) },
		"GetPDFByID":               func() error { return errOf(db.GetPDFByID(1)) },
		"CreatePDFDownloadForItem": func() error { return errOf(db.CreatePDFDownloadForItem(1, 1)) },
		"GetPDFDownload":           func() error { return errOf(db.GetPDFDownload("x")) },
		"UpdatePdfDownload":        func() error { return db.UpdatePdfDownload(PDFDownload{ID: 1}) },
		"GetPDFDownloadByOrderId":  func() error { return errOf(db.GetPDFDownloadByOrderId(1)) },
	}

	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			require.NotPanics(t, func() { err = call() })
			require.Error(t, err)
		})
	}
}

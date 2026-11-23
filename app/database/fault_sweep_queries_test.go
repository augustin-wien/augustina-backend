package database

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// sweepFixture is a vendor with an unverified order of one item
type sweepFixture struct {
	vendor        Vendor
	vendorAccount Account
	itemID        int
	orderID       int
	customerID    int
}

var sweepOrderCodes atomic.Int64

func newSweepFixture(t *testing.T, run int) sweepFixture {
	t.Helper()
	suffix := fmt.Sprintf("%s-%d", t.Name(), run)
	vendorID, err := Db.CreateVendor(Vendor{FirstName: "Sweep", LastName: "Vendor", Email: suffix + "@vendor.test", LicenseID: null.StringFrom(suffix)})
	require.NoError(t, err)
	vendor, err := Db.GetVendor(vendorID)
	require.NoError(t, err)
	account, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	buyer, err := Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{Name: "Item " + suffix, Description: testItemDescription, Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	orderID, err := Db.CreateOrder(Order{
		Vendor:        vendorID,
		OrderCode:     null.StringFrom(fmt.Sprintf("%d", 9000000000+sweepOrderCodes.Add(1))),
		CustomerEmail: null.StringFrom(suffix + "@customer.test"),
		Entries:       []OrderEntry{{Item: itemID, Quantity: 2, Sender: buyer, Receiver: account.ID, IsSale: true}},
	})
	require.NoError(t, err)
	customer, err := Db.CreateCustomer(&Customer{Email: suffix + "@customer.test", KeycloakID: "kc-" + suffix})
	require.NoError(t, err)
	return sweepFixture{vendor: vendor, vendorAccount: account, itemID: itemID, orderID: orderID, customerID: customer.ID}
}

func fixtureSetup(t *testing.T, run int) any { return newSweepFixture(t, run) }

// verifiedFixture also verifies the order, so the vendor has open payments
func verifiedFixture(t *testing.T, run int) any {
	f := newSweepFixture(t, run)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(f.orderID, 0))
	return f
}

func TestQueriesSurviveEachDatabaseFailure(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	mailer.Send = func(*mailer.EmailRequest) (bool, error) { return true, nil }
	require.NoError(t, Db.CreateOrUpdateMailTemplate("PDFLicenceItemTemplate.html", "Dein PDF", `<a href="{{.URL}}">PDF</a>`))

	errOnly := func(err error) error { return err }
	errOf := func(_ any, err error) error { return err }
	cases := map[string]struct {
		setup func(t *testing.T, run int) any
		fn    func(db *Database, data any) error
	}{
		"CreateOrder": {fixtureSetup, func(db *Database, d any) error {
			f := d.(sweepFixture)
			return errOf(db.CreateOrder(Order{Vendor: f.vendor.ID, Entries: []OrderEntry{{Item: f.itemID, Quantity: 1, Sender: f.vendorAccount.ID, Receiver: f.vendorAccount.ID}}}))
		}},
		"VerifyOrderAndCreatePayments": {fixtureSetup, func(db *Database, d any) error {
			return db.VerifyOrderAndCreatePayments(d.(sweepFixture).orderID, 48)
		}},
		"GetVendorWithBalanceUpdate": {verifiedFixture, func(db *Database, d any) error {
			return errOf(db.GetVendorWithBalanceUpdate(d.(sweepFixture).vendor.ID))
		}},
		"CreatePaymentPayout": {verifiedFixture, func(db *Database, d any) error {
			f := d.(sweepFixture)
			payments, err := Db.ListPaymentsForPayout(time.Time{}, time.Time{}, f.vendor.LicenseID.String)
			if err != nil {
				return err
			}
			return errOf(db.CreatePaymentPayout(f.vendor, f.vendorAccount.ID, "admin", 600, payments))
		}},
		"ListPayments": {verifiedFixture, func(db *Database, d any) error {
			return errOf(db.ListPayments(time.Time{}, time.Time{}, d.(sweepFixture).vendor.LicenseID.String, false, false, false, true, true))
		}},
		"ListPaymentsForCustomer": {verifiedFixture, func(db *Database, d any) error {
			return errOf(db.ListPaymentsForCustomer(d.(sweepFixture).vendor.Email))
		}},
		"InvalidateOrder": {fixtureSetup, func(db *Database, d any) error {
			return db.InvalidateOrder(d.(sweepFixture).orderID)
		}},
		"GetOrderStatistics": {fixtureSetup, func(db *Database, d any) error {
			return errOf(db.GetOrderStatistics(time.Time{}, time.Time{}))
		}},
		"ResendOrderMails": {func(t *testing.T, run int) any {
			f := newSweepFixture(t, run)
			pdfID, err := Db.CreatePDF(PDF{Path: fmt.Sprintf("/tmp/sweep-%d.pdf", sweepOrderCodes.Add(1)), Timestamp: time.Now()})
			require.NoError(t, err)
			license, err := Db.CreateItem(Item{Name: fmt.Sprintf("PDF License %s %d", t.Name(), run), Description: testItemDescription, Price: 100, Type: "license_item", IsLicenseItem: true})
			require.NoError(t, err)
			pdfItem, err := Db.CreateItem(Item{Name: fmt.Sprintf("PDF %s %d", t.Name(), run), Description: testItemDescription, Price: 200, Type: "issue", IsPDFItem: true, PDF: null.IntFrom(pdfID), LicenseItem: null.IntFrom(int64(license))})
			require.NoError(t, err)
			orderID, err := Db.CreateOrder(Order{
				Vendor:        f.vendor.ID,
				OrderCode:     null.StringFrom(fmt.Sprintf("%d", 9000000000+sweepOrderCodes.Add(1))),
				CustomerEmail: null.StringFrom(fmt.Sprintf("pdf-%d@customer.test", run)),
				Entries:       []OrderEntry{{Item: pdfItem, Quantity: 1, Sender: f.vendorAccount.ID, Receiver: f.vendorAccount.ID, IsSale: true}},
			})
			require.NoError(t, err)
			require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 0))
			f.orderID = orderID
			return f
		}, func(db *Database, d any) error {
			return errOf(db.ResendOrderMails(d.(sweepFixture).orderID))
		}},
		"UpdateItem": {fixtureSetup, func(db *Database, d any) error {
			f := d.(sweepFixture)
			return db.UpdateItem(f.itemID, Item{Name: fmt.Sprintf("Renamed %d", f.itemID), Description: testItemDescription, Price: 400, Type: "normal_item"})
		}},
		"DeleteAndRestoreItem": {fixtureSetup, func(db *Database, d any) error {
			f := d.(sweepFixture)
			if err := db.DeleteItemChecked(f.itemID, true); err != nil {
				return err
			}
			return db.RestoreItem(f.itemID)
		}},
		"CreateOnlineIssueWithLicense": {func(t *testing.T, run int) any { return run }, func(db *Database, d any) error {
			_, _, err := db.CreateOnlineIssueWithLicense(Item{Name: fmt.Sprintf("Sweep Issue %d", d.(int)), Description: testItemDescription, Price: 300, LicenseGroup: null.StringFrom("sweep")}, 100)
			return err
		}},
		"CreateVendorAndDelete": {func(t *testing.T, run int) any { return run }, func(db *Database, d any) error {
			id, err := db.CreateVendor(Vendor{FirstName: "New", LastName: "Vendor", Email: fmt.Sprintf("new-%d@vendor.test", d.(int)), LicenseID: null.StringFrom(fmt.Sprintf("sweep-new-%d", d.(int)))})
			if err != nil {
				return err
			}
			return db.DeleteVendor(id)
		}},
		"UpdateVendor": {fixtureSetup, func(db *Database, d any) error {
			f := d.(sweepFixture)
			v := f.vendor
			v.FirstName = "Changed"
			return db.UpdateVendor(v.ID, v)
		}},
		"ReleaseVendorLocations": {func(t *testing.T, run int) any {
			f := newSweepFixture(t, run)
			require.NoError(t, Db.CreateLocation(f.vendor.ID, ent.Location{Name: "Billa"}))
			return f
		}, func(db *Database, d any) error {
			return db.ReleaseVendorLocations(d.(sweepFixture).vendor.ID, d.(sweepFixture).vendor.ID%2 == 0)
		}},
		"RecordVendorIDScan": {fixtureSetup, func(db *Database, d any) error {
			return errOf(db.RecordVendorIDScan(d.(sweepFixture).vendor.LicenseID.String))
		}},
		"AddLicenseGroupAndNewsletter": {fixtureSetup, func(db *Database, d any) error {
			f := d.(sweepFixture)
			if _, err := db.AddLicenseGroupToCustomer(f.customerID, "sweep_group"); err != nil {
				return err
			}
			return db.SubscribeCustomerToNewsletter(f.customerID)
		}},
		"ListCustomers": {verifiedFixture, func(db *Database, d any) error {
			return errOf(db.ListCustomers())
		}},
		"RecordExternalPayment": {func(t *testing.T, run int) any { return run }, func(db *Database, d any) error {
			return db.RecordExternalPayment(ExternalPaymentUpdate{TransactionID: fmt.Sprintf("ext-%d", d.(int)), Amount: 500, StatusID: "F", PaidAt: time.Now()})
		}},
		"ListPOSOrdersForVendor": {func(t *testing.T, run int) any {
			f := newSweepFixture(t, run)
			cash, err := Db.GetAccountByType("Cash")
			require.NoError(t, err)
			require.NoError(t, Db.CreatePayments([]Payment{
				{Sender: f.vendorAccount.ID, Receiver: cash.ID, Amount: 300, IsPOS: true, IsSale: true, Item: null.IntFrom(int64(f.itemID)), Quantity: 1, Price: 300},
			}))
			return f
		}, func(db *Database, d any) error {
			return errOf(db.ListPOSOrdersForVendor(d.(sweepFixture).vendor.LicenseID.String, 10))
		}},
		"SendMailNow": {func(t *testing.T, run int) any { return run }, func(db *Database, d any) error {
			m, err := mailer.NewRequest([]string{"a@example.com"}, "s", "b")
			if err != nil {
				return err
			}
			return errOnly(db.SendMailNow(m, MailMeta{Kind: MailKindWelcome}))
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sweepFaults(t, tc.setup, tc.fn)
		})
	}
}

// TestListAllPOSOrdersSurvivesEachDatabaseFailure runs on fixed data: the query
// looks up every vendor's account, so new vendors per run would add calls
func TestListAllPOSOrdersSurvivesEachDatabaseFailure(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	f := newSweepFixture(t, 1)
	cash, err := Db.GetAccountByType("Cash")
	require.NoError(t, err)
	require.NoError(t, Db.CreatePayments([]Payment{
		{Sender: f.vendorAccount.ID, Receiver: cash.ID, Amount: 300, IsPOS: true, IsSale: true, Item: null.IntFrom(int64(f.itemID)), Quantity: 1, Price: 300},
	}))
	sweepFaults(t, func(*testing.T, int) any { return nil }, func(db *Database, _ any) error {
		_, err := db.ListAllPOSOrders(time.Time{}, time.Time{})
		return err
	})
}

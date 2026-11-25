package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestAccountLookups(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	cash, err := Db.GetAccountByType("Cash")
	require.NoError(t, err)

	byID, err := Db.GetAccountByID(cash.ID)
	require.NoError(t, err)
	require.Equal(t, "Cash", byID.Type)

	_, err = Db.GetAccountByID(999999)
	require.EqualError(t, err, "account does not exist")

	_, err = Db.GetAccountByVendorID(999999)
	require.Error(t, err)
}

func TestRecalculateAllVendorBalances(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	vendorID, err := Db.CreateVendor(Vendor{
		FirstName: "Balance",
		LastName:  "Vendor",
		Email:     "balance@vendor.test",
		LicenseID: null.StringFrom("tv-balance-1"),
	})
	require.NoError(t, err)
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	cash, err := Db.GetAccountByType("Cash")
	require.NoError(t, err)

	_, err = Db.CreatePayment(Payment{Sender: cash.ID, Receiver: vendorAccount.ID, Amount: 700, Quantity: 1, Price: 700})
	require.NoError(t, err)

	// Corrupt the stored balance, then recalculate it from the payments
	_, err = Db.EntClient.Account.UpdateOneID(vendorAccount.ID).SetBalance(1).Save(context.Background())
	require.NoError(t, err)

	require.NoError(t, Db.RecalculateAllVendorBalances())

	vendorAccount, err = Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	require.Equal(t, 700, vendorAccount.Balance)
}

func TestProcessAbonementForCustomer(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	aboItem, err := Db.CreateItem(Item{
		Name:         "Coverage Abo",
		Description:  "This is a test item description that is long enough",
		Price:        5000,
		Type:         "abonement",
		LicenseGroup: null.StringFrom("coverage_digital"),
	})
	require.NoError(t, err)
	otherAboItem, err := Db.CreateItem(Item{
		Name:         "Coverage Abo Expired",
		Description:  "This is a test item description that is long enough",
		Price:        5000,
		Type:         "abonement",
		LicenseGroup: null.StringFrom("coverage_expired"),
	})
	require.NoError(t, err)

	customer, err := Db.CreateCustomer(&Customer{
		KeycloakID:    "kc-coverage-abo",
		Email:         "coverage-abo@example.com",
		LicenseGroups: []string{"stale_group"},
	})
	require.NoError(t, err)

	now := time.Now()
	for _, a := range []Abonement{
		{CustomerID: customer.ID, ItemID: aboItem, FromDate: now.AddDate(0, -1, 0), ToDate: now.AddDate(0, 1, 0), Status: "active"},
		// Same license group twice must only be assigned once
		{CustomerID: customer.ID, ItemID: aboItem, FromDate: now.AddDate(0, -2, 0), ToDate: now.AddDate(0, 2, 0), Status: "active"},
		{CustomerID: customer.ID, ItemID: otherAboItem, FromDate: now.AddDate(-2, 0, 0), ToDate: now.AddDate(-1, 0, 0), Status: "active"},
		{CustomerID: customer.ID, ItemID: otherAboItem, FromDate: now.AddDate(0, -1, 0), ToDate: now.AddDate(0, 1, 0), Status: "cancelled"},
	} {
		_, err := Db.CreateAbonement(&a)
		require.NoError(t, err)
	}

	mock := &mockAssigner{}
	svc := newTestService(mock)
	require.NoError(t, svc.ProcessAbonementForCustomer(customer.ID, now))

	updated, err := Db.GetCustomerByID(customer.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"coverage_digital"}, updated.LicenseGroups)
	require.Equal(t, []assignCall{{"kc-coverage-abo", "coverage_digital"}}, mock.calls)

	require.Error(t, svc.ProcessAbonementForCustomer(999999, now))
}

func TestGetAbonementsByDateRangeAndList(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	customer, err := Db.CreateCustomer(&Customer{KeycloakID: "kc-range", Email: "range@example.com"})
	require.NoError(t, err)

	jan := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	inRange, err := Db.CreateAbonement(&Abonement{CustomerID: customer.ID, FromDate: jan, ToDate: jan.AddDate(0, 3, 0), Status: "active"})
	require.NoError(t, err)
	_, err = Db.CreateAbonement(&Abonement{CustomerID: customer.ID, FromDate: jan.AddDate(1, 0, 0), ToDate: jan.AddDate(1, 3, 0), Status: "active"})
	require.NoError(t, err)
	_, err = Db.CreateAbonement(&Abonement{CustomerID: customer.ID, FromDate: jan, ToDate: jan.AddDate(0, 3, 0), Status: "cancelled"})
	require.NoError(t, err)

	found, err := Db.GetAbonementsByDateRange(jan.AddDate(0, 2, 0), jan.AddDate(0, 5, 0))
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, inRange.ID, found[0].ID)

	all, err := Db.ListAbonements()
	require.NoError(t, err)
	require.Len(t, all, 3)
}

func TestGetOrderEntries(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	vendorID, err := Db.CreateVendor(Vendor{
		FirstName: "Entries",
		LastName:  "Vendor",
		Email:     "entries@vendor.test",
		LicenseID: null.StringFrom("tv-entries-1"),
	})
	require.NoError(t, err)
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	orga, err := Db.GetAccountByType("Orga")
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{
		Name:        "Entries Item",
		Description: "This is a test item description that is long enough",
		Price:       300,
		Type:        "normal_item",
	})
	require.NoError(t, err)

	orderID, err := Db.CreateOrder(Order{
		Vendor: vendorID,
		Entries: []OrderEntry{
			{Item: itemID, Quantity: 2, Price: 300, Sender: orga.ID, Receiver: vendorAccount.ID, IsSale: true},
		},
	})
	require.NoError(t, err)

	entries, err := Db.GetOrderEntries(orderID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, itemID, entries[0].Item)
	require.Equal(t, 2, entries[0].Quantity)

	tx, err := Db.EntClient.Tx(context.Background())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	txEntries, err := Db.GetOrderEntriesTx(tx, orderID)
	require.NoError(t, err)
	require.Equal(t, entries, txEntries)

	empty, err := Db.GetOrderEntries(999999)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestPDFDownloadTxHelpers(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	pdfID, err := Db.CreatePDF(PDF{Path: "/tmp/does-not-matter.pdf", Timestamp: time.Now()})
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{
		Name:        "PDF Tx Item",
		Description: "This is a test item description that is long enough",
		Price:       300,
		Type:        "issue",
		IsPDFItem:   true,
		PDF:         null.IntFrom(pdfID),
	})
	require.NoError(t, err)
	download, err := Db.CreatePDFDownloadForItem(int(pdfID), itemID)
	require.NoError(t, err)

	tx, err := Db.EntClient.Tx(context.Background())
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	got, err := Db.GetPDFDownloadTx(tx, download.LinkID)
	require.NoError(t, err)
	require.Equal(t, download.ID, got.ID)
	require.True(t, got.EmailSent)

	_, err = Db.GetPDFDownloadTx(tx, "unknown-link")
	require.Error(t, err)

	byOrder, err := Db.GetPDFDownloadByOrderIdTx(tx, 999999)
	require.NoError(t, err)
	require.Empty(t, byOrder)
}

func TestMailTemplateQueries(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	require.NoError(t, Db.CreateOrUpdateMailTemplate("zz-coverage", "Betreff", "Body"))
	require.NoError(t, Db.CreateOrUpdateMailTemplate("zz-coverage", "Betreff 2", "Body 2"))

	list, err := Db.ListMailTemplates()
	require.NoError(t, err)
	var found *MailTemplate
	for i := range list {
		if list[i].Name == "zz-coverage" {
			found = &list[i]
		}
	}
	require.NotNil(t, found)
	require.Equal(t, "Betreff 2", found.Subject)

	require.NoError(t, Db.DeleteMailTemplate("zz-coverage"))
	_, err = Db.GetMailTemplateByName("zz-coverage")
	require.Error(t, err)
}

func TestListLicenseGroupsSkipsDefaultAndArchived(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	before, err := Db.ListLicenseGroups()
	require.NoError(t, err)

	for _, it := range []Item{
		{Name: "LG Issue A", Type: "issue", LicenseGroup: null.StringFrom("lg_cov_a")},
		{Name: "LG Issue A2", Type: "online_issue", LicenseGroup: null.StringFrom("lg_cov_a")},
		{Name: "LG Default", Type: "issue", LicenseGroup: null.StringFrom("default")},
		{Name: "LG Normal", Type: "normal_item", LicenseGroup: null.StringFrom("lg_cov_normal")},
		{Name: "LG Archived", Type: "issue", LicenseGroup: null.StringFrom("lg_cov_archived"), Archived: true},
	} {
		it.Description = "This is a test item description that is long enough"
		it.Price = 100
		_, err := Db.CreateItem(it)
		require.NoError(t, err)
	}

	groups, err := Db.ListLicenseGroups()
	require.NoError(t, err)
	require.ElementsMatch(t, append(before, "lg_cov_a"), groups)
}

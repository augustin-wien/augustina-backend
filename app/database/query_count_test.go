package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// countingDriver counts the statements sent to the database
type countingDriver struct {
	dialect.Driver
	n *atomic.Int64
}

func (d *countingDriver) Exec(ctx context.Context, q string, args, v any) error {
	d.n.Add(1)
	return d.Driver.Exec(ctx, q, args, v)
}

func (d *countingDriver) Query(ctx context.Context, q string, args, v any) error {
	d.n.Add(1)
	return d.Driver.Query(ctx, q, args, v)
}

func (d *countingDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return d.BeginTx(ctx, nil)
}

func (d *countingDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	d.n.Add(1)
	return d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
}

// countQueries returns how many statements fn sends to the database
func countQueries(t *testing.T, fn func(db *Database) error) int64 {
	t.Helper()
	n := &atomic.Int64{}
	db := &Database{EntClient: ent.NewClient(ent.Driver(&countingDriver{Driver: entsql.OpenDB(dialect.Postgres, Db.DB), n: n})), DB: Db.DB}
	require.NoError(t, fn(db))
	return n.Load()
}

var countSeq atomic.Int64

// addPOSSale creates a vendor with a POS sale of a new item
func addPOSSale(t *testing.T) string {
	t.Helper()
	n := countSeq.Add(1)
	license := fmt.Sprintf("count-%d", n)
	vendorID, err := Db.CreateVendor(Vendor{FirstName: "Count", LastName: "Vendor", Email: license + "@vendor.test", LicenseID: null.StringFrom(license)})
	require.NoError(t, err)
	account, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	cash, err := Db.GetAccountByType("Cash")
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{Name: "Count item " + license, Description: "This is a test item description that is long enough", Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	require.NoError(t, Db.CreatePayments([]Payment{
		{Sender: account.ID, Receiver: cash.ID, Amount: 300, IsPOS: true, IsSale: true, Item: null.IntFrom(int64(itemID)), Quantity: 1, Price: 300},
	}))
	return license
}

// TestQueriesDoNotGrowWithTheData checks that listings send the same number of
// statements whether there is little or much data
func TestQueriesDoNotGrowWithTheData(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	listAll := func(db *Database) error {
		orders, err := db.ListAllPOSOrders(time.Time{}, time.Time{})
		if err == nil && len(orders) == 0 {
			err = fmt.Errorf("no POS orders")
		}
		return err
	}
	license := addPOSSale(t)
	forVendor := func(db *Database) error {
		_, err := db.ListPOSOrdersForVendor(license, 20)
		return err
	}
	vendorAccounts := func(db *Database) error {
		_, err := db.GetVendorAccountIDs()
		return err
	}

	fewAll, fewVendor, fewAccounts := countQueries(t, listAll), countQueries(t, forVendor), countQueries(t, vendorAccounts)
	for i := 0; i < 5; i++ {
		addPOSSale(t)
	}
	cash, err := Db.GetAccountByType("Cash")
	require.NoError(t, err)
	vendor, err := Db.GetVendorByLicenseID(license)
	require.NoError(t, err)
	account, err := Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		itemID, err := Db.CreateItem(Item{Name: fmt.Sprintf("Count extra %d", i), Description: "This is a test item description that is long enough", Price: 100, Type: "normal_item"})
		require.NoError(t, err)
		require.NoError(t, Db.CreatePayments([]Payment{
			{Sender: account.ID, Receiver: cash.ID, Amount: 100, IsPOS: true, IsSale: true, Item: null.IntFrom(int64(itemID)), Quantity: 1, Price: 100},
		}))
	}

	require.Equal(t, fewAll, countQueries(t, listAll), "ListAllPOSOrders")
	require.Equal(t, fewVendor, countQueries(t, forVendor), "ListPOSOrdersForVendor")
	require.Equal(t, fewAccounts, countQueries(t, vendorAccounts), "GetVendorAccountIDs")

	orders, err := Db.ListAllPOSOrders(time.Time{}, time.Time{})
	require.NoError(t, err)
	named := 0
	for _, o := range orders {
		if o.VendorLicenseID != "" && len(o.Items) > 0 && o.Items[0].ItemName != "" {
			named++
		}
	}
	require.Equal(t, len(orders), named, "every order has its vendor and item names")
}

func TestGetCustomersByIDs(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	var ids []int
	for i := 0; i < 3; i++ {
		c, err := Db.CreateCustomer(&Customer{Email: fmt.Sprintf("batch-%d@example.com", i)})
		require.NoError(t, err)
		ids = append(ids, c.ID)
	}
	customers, err := Db.GetCustomersByIDs(append(ids, 999999))
	require.NoError(t, err)
	require.Len(t, customers, 3)
	require.Equal(t, "batch-1@example.com", customers[ids[1]].Email)

	empty, err := Db.GetCustomersByIDs(nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	one := countQueries(t, func(db *Database) error { _, err := db.GetCustomersByIDs(ids[:1]); return err })
	all := countQueries(t, func(db *Database) error { _, err := db.GetCustomersByIDs(ids); return err })
	require.Equal(t, one, all)
}

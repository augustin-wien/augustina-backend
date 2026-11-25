package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

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

func (d *countingDriver) Tx(ctx context.Context) (dialect.Tx, error) { return d.BeginTx(ctx, nil) }

func (d *countingDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	d.n.Add(1)
	return d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
}

// countRequestQueries sends the request and returns how many statements it sent to the database
func countRequestQueries(t *testing.T, path string) int64 {
	t.Helper()
	n := &atomic.Int64{}
	orig := database.Db
	database.Db = database.Database{EntClient: ent.NewClient(ent.Driver(&countingDriver{Driver: entsql.OpenDB(dialect.Postgres, orig.DB), n: n})), DB: orig.DB, IsProduction: orig.IsProduction}
	defer func() { database.Db = orig }()
	utils.TestRequestWithAuth(t, r, "GET", path, nil, 200, adminUserToken)
	return n.Load()
}

var countSeq atomic.Int64

func addVendorWithAbonnent(t *testing.T) {
	t.Helper()
	n := countSeq.Add(1)
	_, err := database.Db.CreateVendor(database.Vendor{FirstName: "Count", LastName: "Vendor", Email: fmt.Sprintf("count-%d@vendor.test", n), LicenseID: null.StringFrom(fmt.Sprintf("count-%d", n))})
	require.NoError(t, err)
	c, err := database.Db.CreateCustomer(&database.Customer{Email: fmt.Sprintf("count-%d@customer.test", n)})
	require.NoError(t, err)
	_, err = database.Db.CreateAbonement(&database.Abonement{CustomerID: c.ID, FromDate: time.Now().AddDate(0, -1, 0), ToDate: time.Now().AddDate(0, 1, 0), Status: "active"})
	require.NoError(t, err)
}

// TestListingsDoNotQueryPerRow checks that the number of statements of these
// listings does not grow with the number of vendors and abonnents
func TestListingsDoNotQueryPerRow(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	addVendorWithAbonnent(t)
	paths := []string{"/api/vendors/statistics/", "/api/abonements/active/"}
	few := map[string]int64{}
	for _, p := range paths {
		few[p] = countRequestQueries(t, p)
	}
	for i := 0; i < 5; i++ {
		addVendorWithAbonnent(t)
	}
	for _, p := range paths {
		require.Equal(t, few[p], countRequestQueries(t, p), p)
	}

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/abonements/active/", nil, 200, adminUserToken)
	var active []ActiveAbonementWithCustomer
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &active))
	require.Len(t, active, 6)
	for _, a := range active {
		require.Equal(t, a.Abonement.CustomerID, a.Customer.ID)
	}

	recipients, err := activeAbonementRecipients()
	require.NoError(t, err)
	require.Len(t, recipients, 6)
}

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

var errInjected = errors.New("injected failure")

// failingMutations returns a database on the test database whose mutations of
// the given entity type fail
func failingMutations(entityType string) *Database {
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, Db.DB)))
	client.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if m.Type() == entityType {
				return nil, errInjected
			}
			return next.Mutate(ctx, m)
		})
	})
	return &Database{EntClient: client, DB: Db.DB}
}

// TestCreateDevDataStopsAtFirstError checks that a failing step aborts the
// dev data seed with its error
func TestCreateDevDataStopsAtFirstError(t *testing.T) {
	for _, entityType := range []string{
		"Vendor", "Account", "Location", "Comment", "Item", "Order", "OrderEntry", "Payment",
		"Customer", "Abonement", "Campaign", "PDFDownload", "BlockedIP",
	} {
		t.Run(entityType, func(t *testing.T) {
			require.NoError(t, Db.InitEmptyTestDb())
			err := failingMutations(entityType).CreateDevData()
			require.ErrorIs(t, err, errInjected)
		})
	}
}

// TestInitDataOnFreshDatabase runs the first-start initialization including
// the production steps and the demo data
func TestInitDataOnFreshDatabase(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	require.NoError(t, Db.UpdateDBSettings(DBSettings{IsInitialized: false}))

	origDemo := config.Config.CreateDemoData
	t.Cleanup(func() { config.Config.CreateDemoData = origDemo })
	config.Config.CreateDemoData = true

	prod := Db
	prod.IsProduction = true
	require.NoError(t, initData(&prod))

	dbSettings, err := Db.GetDBSettings()
	require.NoError(t, err)
	require.True(t, dbSettings.IsInitialized)
	vendors, err := Db.ListVendors()
	require.NoError(t, err)
	require.NotEmpty(t, vendors, "demo data is created")

	// A second start leaves the initialized database alone
	require.NoError(t, initData(&prod))
}

// TestInitDataLogsFailedSteps checks that failing setup steps do not stop the
// remaining initialization
func TestInitDataLogsFailedSteps(t *testing.T) {
	for _, entityType := range []string{"DBSettings", "Settings", "Account", "Item"} {
		t.Run(entityType, func(t *testing.T) {
			require.NoError(t, Db.InitEmptyTestDb())
			require.NoError(t, Db.UpdateDBSettings(DBSettings{IsInitialized: false}))
			db := failingMutations(entityType)
			db.IsProduction = true
			require.NotPanics(t, func() { _ = initData(db) })
		})
	}
}

func TestMultiStepQueriesRollBack(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	vendorID, err := Db.CreateVendor(Vendor{FirstName: "Roll", LastName: "Back", Email: "rollback@vendor.test", LicenseID: null.StringFrom("tv-rollback")})
	require.NoError(t, err)
	vendor, err := Db.GetVendor(vendorID)
	require.NoError(t, err)
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	buyer, err := Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	itemID, err := Db.CreateItem(Item{Name: "Rollback Item", Description: testItemDescription, Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	orderID, err := Db.CreateOrder(Order{Vendor: vendorID, Entries: []OrderEntry{{Item: itemID, Quantity: 1, Sender: buyer, Receiver: vendorAccount.ID, IsSale: true}}})
	require.NoError(t, err)

	require.ErrorIs(t, failingMutations("Payment").VerifyOrderAndCreatePayments(orderID, 0), errInjected)
	order, err := Db.GetOrderByID(orderID)
	require.NoError(t, err)
	require.False(t, order.Verified, "a failed verification is rolled back")

	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 0))

	payments, err := Db.ListPaymentsForPayout(time.Time{}, time.Time{}, "tv-rollback")
	require.NoError(t, err)
	require.NotEmpty(t, payments)
	_, err = failingMutations("Payment").CreatePaymentPayout(vendor, vendorAccount.ID, "admin", 300, payments)
	require.ErrorIs(t, err, errInjected)

	for _, entityType := range []string{"Item"} {
		_, _, err = failingMutations(entityType).CreateOnlineIssueWithLicense(Item{Name: "Rollback Issue", Description: testItemDescription, Price: 300, LicenseGroup: null.StringFrom("rb")}, 100)
		require.ErrorIs(t, err, errInjected)
	}
	_, err = Db.GetItemByName("Rollback Issue")
	require.Error(t, err, "nothing is left behind")
}

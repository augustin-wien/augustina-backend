package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestCreateDevDataArchivedItems ensures the seed contains archived items and
// that the sold one is still referenced by payments.
func TestCreateDevDataArchivedItems(t *testing.T) {
	Db.InitEmptyTestDb()
	require.NoError(t, Db.CreateDevData())

	archived, err := Db.ListArchivedItems()
	require.NoError(t, err)
	names := map[string]int{}
	for _, it := range archived {
		names[it.Name] = it.ID
	}
	require.Contains(t, names, "Kalender 2023")
	require.Contains(t, names, "Postkartenset")

	sales, err := Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false, false, false)
	require.NoError(t, err)
	found := false
	for _, p := range sales {
		if p.Item.Valid && int(p.Item.Int64) == names["Kalender 2023"] {
			found = true
		}
	}
	require.True(t, found, "archived item must have a sale payment")
}

func TestSeedDevArchivedItems(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	_, err := Db.CreateVendor(Vendor{
		FirstName: "Seed",
		LastName:  "Vendor",
		Email:     "seed@vendor.test",
		LicenseID: null.StringFrom("tv-seed-1"),
	})
	require.NoError(t, err)

	require.NoError(t, Db.SeedDevArchivedItems("tv-seed-1"))
	require.Error(t, Db.SeedDevArchivedItems("unknown-license"))

	archived, err := Db.ListArchivedItems()
	require.NoError(t, err)
	require.Len(t, archived, 2)

	payments, err := Db.ListPayments(time.Time{}, time.Time{}, "tv-seed-1", false, true, false, false, false)
	require.NoError(t, err)
	require.NotEmpty(t, payments)
}

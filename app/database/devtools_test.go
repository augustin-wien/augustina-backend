package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

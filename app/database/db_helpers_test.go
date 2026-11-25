package database

import (
	"context"
	"database/sql"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
)

func TestDatabaseMaintenanceHelpers(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	require.NoError(t, Db.CheckRolePermissions())

	broken := brokenDatabase(t)
	require.Error(t, broken.CheckRolePermissions())
	require.Error(t, broken.EmptyDatabase())

	// Closing a separate client leaves the shared pool usable
	extra := &Database{EntClient: ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, Db.DB)))}
	extra.CloseDbPool()
	(&Database{}).CloseDbPool()
}

func TestDebugDriverTransactions(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())
	ctx := context.Background()
	drv := NewDebugDriver(entsql.OpenDB(dialect.Postgres, Db.DB)).(*DebugDriver)

	tx, err := drv.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	require.NoError(t, err)
	var rows entsql.Rows
	require.NoError(t, tx.Query(ctx, "SELECT 1", []any{}, &rows))
	require.NoError(t, rows.Close())
	require.Error(t, tx.Exec(ctx, "SELECT * FROM no_such_table", []any{}, nil))
	require.NoError(t, tx.Rollback())

	tx, err = drv.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// A constraint violation is logged as a warning, other errors as errors
	require.Error(t, drv.Exec(ctx, "INSERT INTO settings (id) VALUES (1)", []any{}, nil))
	require.Error(t, drv.Exec(ctx, "SELECT * FROM no_such_table", []any{}, nil))

	brokenDrv := NewDebugDriver(entsql.OpenDB(dialect.Postgres, brokenDatabase(t).DB)).(*DebugDriver)
	_, err = brokenDrv.BeginTx(ctx, nil)
	require.Error(t, err)
	_, err = brokenDrv.Tx(ctx)
	require.Error(t, err)
}

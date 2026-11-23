package database

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
)

// faultDriver counts the statements, transaction starts and commits sent to the
// database and fails the one with number failAt
type faultDriver struct {
	dialect.Driver
	calls  *atomic.Int64
	failAt int64
}

func (d *faultDriver) hit() error {
	if d.calls.Add(1) == d.failAt {
		return errInjected
	}
	return nil
}

func (d *faultDriver) Exec(ctx context.Context, query string, args, v any) error {
	if err := d.hit(); err != nil {
		return err
	}
	return d.Driver.Exec(ctx, query, args, v)
}

func (d *faultDriver) Query(ctx context.Context, query string, args, v any) error {
	if err := d.hit(); err != nil {
		return err
	}
	return d.Driver.Query(ctx, query, args, v)
}

func (d *faultDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return d.BeginTx(ctx, nil)
}

func (d *faultDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	if err := d.hit(); err != nil {
		return nil, err
	}
	tx, err := d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &faultTx{Tx: tx, d: d}, nil
}

type faultTx struct {
	dialect.Tx
	d *faultDriver
}

func (t *faultTx) Exec(ctx context.Context, query string, args, v any) error {
	if err := t.d.hit(); err != nil {
		return err
	}
	return t.Tx.Exec(ctx, query, args, v)
}

func (t *faultTx) Query(ctx context.Context, query string, args, v any) error {
	if err := t.d.hit(); err != nil {
		return err
	}
	return t.Tx.Query(ctx, query, args, v)
}

func (t *faultTx) Commit() error {
	if err := t.d.hit(); err != nil {
		_ = t.Tx.Rollback()
		return err
	}
	return t.Tx.Commit()
}

// faultyDatabase returns a database on the test database whose statement number
// failAt fails, and the counter of statements sent so far
func faultyDatabase(failAt int64) (*Database, *atomic.Int64) {
	calls := &atomic.Int64{}
	drv := &faultDriver{Driver: entsql.OpenDB(dialect.Postgres, Db.DB), calls: calls, failAt: failAt}
	return &Database{EntClient: ent.NewClient(ent.Driver(drv)), DB: Db.DB}, calls
}

// sweepFaults runs fn once for every database call it makes, failing that call.
// fn must not panic, and once no call fails it must succeed. setup prepares fresh
// data for each run, using the working database.
func sweepFaults(t *testing.T, setup func(t *testing.T, run int) any, fn func(db *Database, data any) error) {
	t.Helper()
	for failAt := int64(1); ; failAt++ {
		require.Less(t, failAt, int64(500), "too many database calls")
		data := setup(t, int(failAt))
		db, calls := faultyDatabase(failAt)
		var err error
		require.NotPanics(t, func() { err = fn(db, data) }, "failing call %d", failAt)
		if calls.Load() < failAt {
			require.NoError(t, err, "without a failing call")
			return
		}
	}
}

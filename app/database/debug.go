package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	"github.com/augustin-wien/augustina-backend/utils"
)

type DebugDriver struct {
	dialect.Driver
}

func NewDebugDriver(d dialect.Driver) dialect.Driver {
	return &DebugDriver{d}
}

func (d *DebugDriver) Exec(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Driver.Exec(ctx, query, args, v)
	logQuery(ctx, "Exec", query, args, time.Since(start), err)
	return err
}

func (d *DebugDriver) Query(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Driver.Query(ctx, query, args, v)
	logQuery(ctx, "Query", query, args, time.Since(start), err)
	return err
}

func (d *DebugDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := d.Driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	return &DebugTx{tx}, nil
}

func (d *DebugDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	// Check if the underlying driver supports BeginTx
	if starter, ok := d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}); ok {
		tx, err := starter.BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}
		return &DebugTx{tx}, nil
	}
	// Fallback to Tx() if BeginTx is not supported, though opts are ignored
	return d.Tx(ctx)
}

type DebugTx struct {
	dialect.Tx
}

func (d *DebugTx) Exec(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Tx.Exec(ctx, query, args, v)
	logQuery(ctx, "Tx.Exec", query, args, time.Since(start), err)
	return err
}

func (d *DebugTx) Query(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Tx.Query(ctx, query, args, v)
	logQuery(ctx, "Tx.Query", query, args, time.Since(start), err)
	return err
}

func (d *DebugTx) Commit() error {
	return d.Tx.Commit()
}

func (d *DebugTx) Rollback() error {
	return d.Tx.Rollback()
}

func logQuery(ctx context.Context, op, query string, args any, duration time.Duration, err error) {
	logger := utils.LoggerFromContext(ctx)
	msg := fmt.Sprintf("driver.Query: query=%s args=%v", query, args)
	if err != nil {
		logger.Errorw(msg, "op", op, "duration", duration, "error", err)
	} else {
		logger.Infow(msg, "op", op, "duration", duration)
	}
}

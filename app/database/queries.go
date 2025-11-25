package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Helpers --------------------------------------------------------------------

// deferTx executes a transaction after a function returns
func DeferTx(tx pgx.Tx, err error) error {
	if p := recover(); p != nil {
		// Rollback the transaction if a panic occurred
		err = tx.Rollback(context.Background())
		if err != nil {
			log.Error("DeferTx rollback after panic failed: ", err)
		}
		// Re-throw the panic
		panic(p)
	} else if err != nil {
		// Rollback the transaction if an error occurred
		log.Error("deferTx: ", err)
		err = tx.Rollback(context.Background())
		if err != nil {
			log.Error("DeferTx rollback on error failed: ", err)
		}

	} else {
		// Commit the transaction if everything is successful
		err = tx.Commit(context.Background())
		if err != nil {
			log.Error("DeferTx commit failed: ", err)
		}
	}
	return err
}

// Generic --------------------------------------------------------------------

// GetHelloWorld returns the string "Hello, world!" from the database and should be used as a template for other queries
func (db *Database) GetHelloWorld() (string, error) {
	var greeting string
	err := db.Dbpool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		log.Error("QueryRow failed: %v\n", zap.Error(err))
		return "", err
	}
	return greeting, err
}

// NOTE: Item-related queries were moved to queries.items.go so the whole
// feature can be toggled. See that file for implementations which check
// config.Config.ItemsEnabled.

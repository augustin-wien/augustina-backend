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
		rollbackErr := tx.Rollback(context.Background())
		if rollbackErr != nil {
			log.Error("DeferTx rollback on error failed: ", rollbackErr)
		}
		return err

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

// GetHelloWorld returns the string "Hello, world!" from the database and checks connection
func (db *Database) GetHelloWorld() (string, error) {
	// Simple connectivity check using Ent
	_, err := db.EntClient.Settings.Query().Count(context.Background())
	if err != nil {
		log.Error("GetHelloWorld (connectivity check) failed: ", zap.Error(err))
		return "", err
	}
	// Return the expected string to maintain interface compatibility
	return "Hello, world!", nil
}

// NOTE: Item-related queries were moved to queries.items.go so the whole
// feature can be toggled. See that file for implementations which check
// config.Config.ItemsEnabled.

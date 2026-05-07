package database

import (
	"context"

	"go.uber.org/zap"
)

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

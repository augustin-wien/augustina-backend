package database

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
)

// GetHelloWorld returns the string "Hello, world!" from the database and should be used as a template for other queries
func (db *Database) GetHelloWorld() (string, error) {
	var greeting string
	err := db.Dbpool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		log.Error(os.Stderr, "QueryRow failed: %v\n", err)
		return "", err
	}
	return greeting, nil
}

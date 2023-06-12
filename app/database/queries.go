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

func (db *Database) GetSettings() (string, error) {
	var settings string
	err := db.Dbpool.QueryRow(context.Background(), `select '{"color":"red","logo":"/img/Augustin-Logo-Rechteck.jpg","price":3.14}'`).Scan(&settings)
	if err != nil {
		log.Error(os.Stderr, "QueryRow failed: %v\n", err)
		return "", err
	}
	return settings, nil
}

func (db *Database) GetVendorSettings() (string, error) {
	var settings string
	err := db.Dbpool.QueryRow(context.Background(), `select '{"credit":1.61,"qrcode":"/img/Augustin-QR-Code.png","id-number":"123456789"}'`).Scan(&settings)
	if err != nil {
		log.Error(os.Stderr, "QueryRow failed: %v\n", err)
		return "", err
	}
	return settings, nil
}

package database

import (
	"augustin/types"
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

// GetPayments returns the payments from the database
func (db *Database) GetPayments() ([]types.Payment, error) {
	var payments []types.Payment
	rows, err := db.Dbpool.Query(context.Background(), "select * from payments")
	if err != nil {
		return payments, err
	}
	for rows.Next() {
		var payment types.Payment
		err = rows.Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.Type, &payment.Amount)
		if err != nil {
			return payments, err
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

// Create new payment rows
func (db *Database) CreatePayments(payments []types.Payment) (err error) {

	// Create a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return err
	}

	// Handle transaction after function returns
	defer func() {
		if p := recover(); p != nil {
			// Rollback the transaction if a panic occurred
			_ = tx.Rollback(context.Background())
			// Re-throw the panic
			panic(p)
		} else if err != nil {
			// Rollback the transaction if an error occurred
			_ = tx.Rollback(context.Background())
		} else {
			// Commit the transaction if everything is successful
			err = tx.Commit(context.Background())
		}
	}()

	// Insert payments within the transaction
	for _, payment := range payments {
		_, err := tx.Exec(context.Background(), "insert into payments (timestamp, sender, receiver, type, amount) values ($1, $2, $3, $4, $5)", payment.Timestamp, payment.Sender, payment.Receiver, payment.Type, payment.Amount)
		if err != nil {
			return err
		}
	}
	return nil
}

package database

import (
	"augustin/structs"
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
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
func (db *Database) GetPayments() ([]structs.Payment, error) {
	var payments []structs.Payment
	rows, err := db.Dbpool.Query(context.Background(), "select * from payments")
	if err != nil {
		return payments, err
	}
	for rows.Next() {
		var payment structs.Payment
		err = rows.Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.Type, &payment.Amount)
		if err != nil {
			return payments, err
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

// Create payment type
func (db *Database) CreatePaymentType(pt structs.PaymentType)(id pgtype.Int4, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into PaymentTypes (Name) values ($1) RETURNING ID", pt.Name).Scan(&id)
	return id, err
}

// Create account
func (db *Database) CreateAccount(account structs.Account)(id pgtype.Int4, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into Accounts (Name) values ($1) RETURNING ID", account.Name).Scan(&id)
	return id, err
}


// Create multiple payments
func (db *Database) CreatePayments(payments []structs.Payment) (err error) {

	// Create a transaction to insert all payments at once
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
		_, err := tx.Exec(context.Background(), "insert into payments ( sender, receiver, type, amount) values ($1, $2, $3, $4)", payment.Sender, payment.Receiver, payment.Type, payment.Amount)
		if err != nil {
			return err
		}
	}
	return nil
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

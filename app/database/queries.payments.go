package database

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"gopkg.in/guregu/null.v4"
)

// Payments -------------------------------------------------------------------

// ListPayments returns the payments from the database
func (db *Database) ListPayments(minDate time.Time, maxDate time.Time, vendorLicenseID string, filterPayouts bool, filterSales bool, filterNoPayout bool) (payments []Payment, err error) {
	var rows pgx.Rows
	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return nil, err
	}

	defer func() { err = DeferTx(tx, err) }()
	// Create filters
	var filters []string
	var filterValues []any
	var vendor Vendor
	var vendorAccount Account
	var cashAccountID int
	if !minDate.IsZero() {
		filterValues = append(filterValues, minDate)
		filters = append(filters, "Timestamp >= $"+strconv.Itoa(len(filterValues)))
	}
	if !maxDate.IsZero() {
		filterValues = append(filterValues, maxDate)
		filters = append(filters, "Timestamp <= $"+strconv.Itoa(len(filterValues)))

	}
	if vendorLicenseID != "" {
		vendor, err = db.GetVendorByLicenseID(vendorLicenseID)
		if err != nil {
			return
		}
		vendorAccount, err = db.GetAccountByVendorID(vendor.ID)
		if err != nil {
			return
		}
		filterValues = append(filterValues, vendorAccount.ID)
		filters = append(filters, "(Sender = $"+strconv.Itoa(len(filterValues))+" OR Receiver = $"+strconv.Itoa(len(filterValues))+")")
	}
	if filterPayouts {
		cashAccountID, err = db.GetAccountTypeID("Cash")
		if err != nil {
			return
		}
		filterValues = append(filterValues, cashAccountID)
		filters = append(filters, "Receiver = $"+strconv.Itoa(len(filterValues)))
	}
	if filterNoPayout {
		// Does not have a payout and is not a payout (i.e. payment to cash)
		cashAccountID, err = db.GetAccountTypeID("Cash")
		if err != nil {
			return
		}
		filterValues = append(filterValues, cashAccountID)
		filters = append(filters, "Payout IS NULL AND Receiver != $"+strconv.Itoa(len(filterValues)))
	}
	if filterSales {
		filterValues = append(filterValues, true)
		filters = append(filters, "IsSale = $"+strconv.Itoa(len(filterValues)))
	}

	// Query based on parameters
	query := "SELECT Payment.ID, Payment.Timestamp, Sender, Receiver, SenderAccount.Name SenderName, ReceiverAccount.Name ReceiverName, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, null as IsPayoutFor, Item, Quantity, Price FROM Payment JOIN Account as SenderAccount ON SenderAccount.ID = Sender JOIN Account as ReceiverAccount ON ReceiverAccount.ID = Receiver"
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	// Order by timestamp
	query += " ORDER BY Payment.Timestamp"
	rows, err = tx.Query(context.Background(), query, filterValues...)
	if err != nil {
		log.Error("ListPayments: ", err)
		return payments, err
	}
	defer rows.Close()

	tmpPayments, err := pgx.CollectRows(rows, pgx.RowToStructByName[Payment])
	if err != nil {
		log.Error("ListPayments: ", err)
		return payments, err
	}
	for _, payment := range tmpPayments {

		subrows, err := tx.Query(context.Background(), "SELECT ID, Timestamp, Sender, null as SenderName, Receiver, null as ReceiverName, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, null as IsPayoutFor, Item, Quantity, Price FROM Payment WHERE Payout = $1 ORDER BY Timestamp", payment.ID)
		if err != nil {
			return payments, err
		}
		defer subrows.Close()
		tmpSubPayments, err := pgx.CollectRows(subrows, pgx.RowToStructByName[Payment])
		if err != nil {
			log.Error("ListPayments: ", err)
			return payments, err
		}
		payment.IsPayoutFor = append(payment.IsPayoutFor, tmpSubPayments...)
		payments = append(payments, payment)
	}

	return payments, nil
}

// ListPaymentsForPayout returns sales payments that have not been paid out yet
func (db *Database) ListPaymentsForPayout(minDate time.Time, maxDate time.Time, vendorLicenseID string) (payments []Payment, err error) {

	return db.ListPayments(minDate, maxDate, vendorLicenseID, false, false, true)
}

// GetPayment returns the payment with the given ID
func (db *Database) GetPayment(id int) (payment Payment, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Payment.ID, Payment.Timestamp, Sender, Receiver, SenderAccount.Name, ReceiverAccount.Name, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, Item, Quantity, Price FROM Payment JOIN Account as SenderAccount ON SenderAccount.ID = Sender JOIN Account as ReceiverAccount ON ReceiverAccount.ID = Receiver WHERE Payment.ID = $1", id).Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.SenderName, &payment.ReceiverName, &payment.Amount, &payment.AuthorizedBy, &payment.Order, &payment.OrderEntry, &payment.IsSale, &payment.Payout, &payment.Item, &payment.Quantity, &payment.Price)
	if err != nil {
		log.Error("GetPayment: ", err)
	}
	return
}

// CreatePayment creates a payment in an transaction
func createPaymentTx(tx pgx.Tx, payment Payment) (paymentID int, err error) {

	// Create payment
	err = tx.QueryRow(context.Background(), "INSERT INTO Payment (Sender, Receiver, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, Item, Quantity, Price) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING ID", payment.Sender, payment.Receiver, payment.Amount, payment.AuthorizedBy, payment.Order, payment.OrderEntry, payment.IsSale, payment.Payout, payment.Item, payment.Quantity, payment.Price).Scan(&paymentID)
	if err != nil {
		log.Error("createPaymentTx: query row ", err)
		return
	}

	// Update account balances
	err = updateAccountBalanceTx(tx, payment.Sender, -payment.Amount)
	if err != nil {
		log.Error("createPaymentTx: update sender ", err)
	}
	err = updateAccountBalanceTx(tx, payment.Receiver, payment.Amount)
	if err != nil {
		log.Error("createPaymentTx: update receiver", err)
	}
	return
}

// CreatePayment creates a payment and returns the payment ID
func (db *Database) CreatePayment(payment Payment) (paymentID int, err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return
	}
	defer func() { err = DeferTx(tx, err) }()

	paymentID, err = createPaymentTx(tx, payment)

	return
}

// CreatePayments creates multiple payments
func (db *Database) CreatePayments(payments []Payment) (err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("CreatePayments: ", err)
		return err
	}
	defer func() { err = DeferTx(tx, err) }()

	// Insert payments within the transaction
	for _, payment := range payments {
		_, err = createPaymentTx(tx, payment)
		if err != nil {
			log.Error("CreatePayments: ", err)
			return err
		}
	}
	return nil
}

// CreatePaymentPayout creates a payout for a range of payments
func (db *Database) CreatePaymentPayout(vendor Vendor, vendorAccountID int, authorizedBy string, amount int, payments []Payment) (paymentID int, err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}
	defer func() { err = DeferTx(tx, err) }()

	// Get cash account
	cashAccount, err := db.GetAccountByType("Cash")
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}

	payment := Payment{
		Sender:       vendorAccountID,
		Receiver:     cashAccount.ID,
		Amount:       amount,
		AuthorizedBy: authorizedBy,
	}

	// Insert payments within the transaction
	paymentID, err = createPaymentTx(tx, payment)
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}

	// Document that these payments have a payout
	for _, payment := range payments {
		_, err = tx.Exec(context.Background(), `
		UPDATE Payment
		SET Payout = $1
		WHERE ID = $2
		`, paymentID, payment.ID)
		if err != nil {
			log.Error("CreatePaymentPayout: ", err)
		}
	}

	// Update last payout date
	vendor.LastPayout = null.NewTime(time.Now(), true)
	err = db.UpdateVendor(vendor.ID, vendor)
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}

	return
}

// DeletePayment deletes a payment (should not be used in production)
func (db *Database) DeletePayment(paymentID int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM Payment
	WHERE ID = $1
	`, paymentID)
	if err != nil {
		log.Error("DeletePayment: ", err)
	}
	return
}

package database

import (
	"augustin/config"
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"gopkg.in/guregu/null.v4"
)

// Helpers --------------------------------------------------------------------

// deferTx executes a transaction after a function returns
func deferTx(tx pgx.Tx, err error) error {
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
		if err != nil {
			log.Error(err)
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

// Users ----------------------------------------------------------------------

// ListVendors returns all users from the database but not all fields for better overview
func (db *Database) ListVendors() (vendors []Vendor, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "SELECT vendor.ID, LicenseID, FirstName, LastName, LastPayout, Balance from Vendor JOIN account ON account.vendor = vendor.id ORDER BY LicenseID ASC")
	if err != nil {
		log.Error(err)
		return vendors, err
	}
	for rows.Next() {
		var vendor Vendor
		err = rows.Scan(&vendor.ID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.LastPayout, &vendor.Balance)
		if err != nil {
			log.Error(err)
			return vendors, err
		}
		vendors = append(vendors, vendor)
	}
	return vendors, nil
}

// GetVendorByLicenseID returns the vendor with the given licenseID
func (db *Database) GetVendorByLicenseID(licenseID string) (vendor Vendor, err error) {
	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE LicenseID = $1", licenseID).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap, &vendor.HasSmartphone, &vendor.HasBankAccount)
	if err != nil {
		log.Error("GetVendorByLicenseID: Couldn't get vendor ", licenseID, err)
		return vendor, err
	}

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error(err)
	}
	return vendor, err
}

// GetVendorByEmail returns the vendor with the given licenseID
func (db *Database) GetVendorByEmail(mail string) (vendor Vendor, err error) {
	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE Email = $1", mail).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap, &vendor.HasSmartphone, &vendor.HasBankAccount)
	if err != nil {
		log.Error("GetVendorByEmail: Couldn't get vendor ", mail, err)
		return vendor, err
	}

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error(err)
	}
	return vendor, err
}

// GetVendor returns the vendor with the given id
func (db *Database) GetVendor(vendorID int) (vendor Vendor, err error) {

	// Update Account balance by open payments
	_, err = db.UpdateAccountBalanceByOpenPayments(vendorID)
	if err != nil {
		log.Error(err)
	}

	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE ID = $1", vendorID).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap, &vendor.HasSmartphone, &vendor.HasBankAccount)
	if err != nil {
		log.Error("GetVendor: Couldn't get vendor ", vendorID, err)
		return vendor, err
	}
	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error(err)
	}
	return vendor, err
}

// CreateVendor creates a vendor and an associated account in the database
func (db *Database) CreateVendor(vendor Vendor) (vendorID int, err error) {

	// Create vendor
	err = db.Dbpool.QueryRow(context.Background(), "insert into Vendor (keycloakid, UrlID, LicenseID, FirstName, LastName, Email, LastPayout, IsDisabled, Longitude, Latitude, Address, PLZ, Location, WorkingTime, Language, Comment, Telephone, RegistrationDate, VendorSince, OnlineMap, HasSmartphone, HasBankAccount) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22) RETURNING ID", vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, vendor.IsDisabled, vendor.Longitude, vendor.Latitude, vendor.Address, vendor.PLZ, vendor.Location, vendor.WorkingTime, vendor.Language, vendor.Comment, vendor.Telephone, vendor.RegistrationDate, vendor.VendorSince, vendor.OnlineMap, vendor.HasSmartphone, vendor.HasBankAccount).Scan(&vendorID)
	if err != nil {
		log.Error(err)
		return
	}

	// Create vendor account
	_, err = db.Dbpool.Exec(context.Background(), "insert into Account (Name, Balance, Type, Vendor) values ($1, 0, $2, $3) RETURNING ID", vendor.LicenseID, "Vendor", vendorID)
	if err != nil {
		log.Error(err)
		return
	}

	return
}

// UpdateVendor updates a user in the database
func (db *Database) UpdateVendor(id int, vendor Vendor) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Vendor
	SET keycloakid = $1, UrlID = $2, LicenseID = $3, FirstName = $4, LastName = $5, Email = $6, LastPayout = $7, IsDisabled = $8, Longitude = $9, Latitude = $10, Address = $11, PLZ = $12, Location = $13, WorkingTime = $14, Language = $15, Comment = $16, Telephone = $17, RegistrationDate = $18, VendorSince = $19, OnlineMap = $20, HasSmartphone = $21, HasBankAccount = $22
	WHERE ID = $23
	`, vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, vendor.IsDisabled, vendor.Longitude, vendor.Latitude, vendor.Address, vendor.PLZ, vendor.Location, vendor.WorkingTime, vendor.Language, vendor.Comment, vendor.Telephone, vendor.RegistrationDate, vendor.VendorSince, vendor.OnlineMap, vendor.HasSmartphone, vendor.HasBankAccount, id)
	if err != nil {
		log.Error(err)
	}
	return
}

// DeleteVendor deletes a user in the database and the associated account
func (db *Database) DeleteVendor(vendorID int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM Vendor
	WHERE ID = $1
	`, vendorID)
	if err != nil {
		log.Error(err)
	}

	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM Account
	WHERE Vendor = $1
	`, vendorID)
	if err != nil {
		log.Error(err)
	}

	return
}

// Items ----------------------------------------------------------------------

// ListItems returns all items from the database
func (db *Database) ListItems(skipHiddenItems bool, skipLicenses bool) ([]Item, error) {
	var items []Item
	rows, err := db.Dbpool.Query(context.Background(), "SELECT * FROM Item ORDER BY ID ASC")
	if err != nil {
		log.Error(err)
		return items, err
	}
	for rows.Next() {
		var item Item
		err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem)
		if err != nil {
			log.Error(err)
			return items, err
		}

		// Hardcode check: Do not add default items with their config names TransactionCostsName and DonationName
		if skipHiddenItems && (item.Name == config.Config.TransactionCostsName || item.Name == config.Config.DonationName) {
			continue
		}

		if skipLicenses && item.IsLicenseItem {
			continue
		}

		items = append(items, item)

	}
	return items, nil
}

// GetItemByName returns the item with the given name
func (db *Database) GetItemByName(name string) (item Item, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Item WHERE Name = $1", name).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem)
	if err != nil {
		log.Error(err)
	}
	return
}

// GetItem returns the item with the given ID
func (db *Database) GetItem(id int) (item Item, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Item WHERE ID = $1", id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem)
	if err != nil {
		log.Error(err)
	}
	return
}

// CreateItem creates an item in the database
func (db *Database) CreateItem(item Item) (id int, err error) {
	// Check if the item name already exists
	var count int
	err = db.Dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM Item WHERE Name = $1", item.Name).Scan(&count)
	if err != nil {
		log.Error(err)
		return 0, err
	}
	if count > 0 {
		return 0, errors.New("Item with the same name already exists. Update it or delete it first")
	}

	// Insert the new item
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Item (Name, Description, Price, LicenseItem, Archived, IsLicenseItem) values ($1, $2, $3, $4, $5, $6) RETURNING ID", item.Name, item.Description, item.Price, item.LicenseItem, item.Archived, item.IsLicenseItem).Scan(&id)
	if err != nil {
		log.Error(err)
	}
	return id, err
}

// UpdateItem updates an item in the database
func (db *Database) UpdateItem(id int, item Item) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Item
	SET Name = $2, Description = $3, Price = $4, Image = $5, LicenseItem = $6, Archived = $7, IsLicenseItem = $8
	WHERE ID = $1
	`, id, item.Name, item.Description, item.Price, item.Image, item.LicenseItem, item.Archived, item.IsLicenseItem)
	if err != nil {
		log.Error(err)
	}
	return
}

// DeleteItem deletes an item in the database
func (db *Database) DeleteItem(id int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM Item
	WHERE ID = $1
	`, id)
	if err != nil {
		log.Error(err)
	}
	return
}

// Orders ---------------------------------------------------------------------

// GetOrderEntries returns all entries of an order
func (db *Database) GetOrderEntries(orderID int) (entries []OrderEntry, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "SELECT OrderEntry.ID, Item, Quantity, Price, Sender, Receiver, SenderAccount.Name, ReceiverAccount.Name, IsSale FROM OrderEntry JOIN Account as SenderAccount ON SenderAccount.ID = Sender JOIN Account as ReceiverAccount ON ReceiverAccount.ID = Receiver WHERE paymentorder = $1 ", orderID)
	if err != nil {
		log.Error(err)
		return
	}
	for rows.Next() {
		var entry OrderEntry
		err = rows.Scan(&entry.ID, &entry.Item, &entry.Quantity, &entry.Price, &entry.Sender, &entry.Receiver, &entry.SenderName, &entry.ReceiverName, &entry.IsSale)
		if err != nil {
			log.Error(err)
			return
		}
		entries = append(entries, entry)
	}
	return
}

// GetOrders returns all orders from the database
func (db *Database) GetOrders() (orders []Order, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "SELECT * FROM PaymentOrder")
	if err != nil {
		log.Error(err)
		return orders, err
	}
	for rows.Next() {
		var order Order
		err = rows.Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.TransactionTypeID, &order.Timestamp, &order.User, &order.Vendor)
		if err != nil {
			log.Error(err)
			return orders, err
		}
		// Add entries to order
		order.Entries, err = db.GetOrderEntries(order.ID)
		if err != nil {
			log.Error(err)
		}
		orders = append(orders, order)
	}
	return
}

// GetOrderByID returns Order by OrderID
func (db *Database) GetOrderByID(id int) (order Order, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE ID = $1", id).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.TransactionTypeID, &order.Timestamp, &order.User, &order.Vendor)
	if err != nil {
		log.Error(err)
		return
	}

	// Add entries to order
	order.Entries, err = db.GetOrderEntries(order.ID)
	if err != nil {
		log.Error(err)
	}

	return
}

// GetOrderByOrderCode returns Order by OrderCode
func (db *Database) GetOrderByOrderCode(OrderCode string) (order Order, err error) {

	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE OrderCode = $1", OrderCode).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.TransactionTypeID, &order.Timestamp, &order.User, &order.Vendor)
	if err != nil {
		log.Error(err)
		return
	}

	// Add items to order
	order.Entries, err = db.GetOrderEntries(order.ID)
	if err != nil {
		log.Error(err, " orderID: ", order.ID)
	}
	return
}

// CreateOrder creates an order in the database
// Processes OrderCode, vendor, and items (trinkgeld is an item)
func (db *Database) CreateOrder(order Order) (orderID int, err error) {

	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return
	}
	defer func() { err = deferTx(tx, err) }()

	err = tx.QueryRow(context.Background(), "INSERT INTO PaymentOrder (OrderCode, Vendor) values ($1, $2) RETURNING ID", order.OrderCode, order.Vendor).Scan(&orderID)
	if err != nil {
		log.Error(err)
		return
	}

	// Create order items
	for _, entry := range order.Entries {
		_, err = createOrderEntryTx(tx, orderID, entry)
		if err != nil {
			log.Error(err)
			return
		}
	}

	return
}

// createOrderEntryTx adds an entry to an order in an transaction
func createOrderEntryTx(tx pgx.Tx, orderID int, entry OrderEntry) (OrderEntry, error) {

	// Get current item price
	var item Item
	err := tx.QueryRow(context.Background(), "SELECT Price FROM Item WHERE ID = $1", entry.Item).Scan(&item.Price)
	if err != nil {
		log.Error(err)
		return entry, err
	}
	entry.Price = item.Price

	// Create order entry
	err = tx.QueryRow(context.Background(), "INSERT INTO OrderEntry (Item, Price, Quantity, PaymentOrder, Sender, Receiver, IsSale) values ($1, $2, $3, $4, $5, $6, $7) RETURNING ID", entry.Item, entry.Price, entry.Quantity, orderID, entry.Sender, entry.Receiver, entry.IsSale).Scan(&entry.ID)
	if err != nil {
		log.Error(err)
	}
	return entry, err
}

// createPaymentForOrderEntryTx creates a payment for an order entry
func createPaymentForOrderEntryTx(tx pgx.Tx, orderID int, entry OrderEntry, errorIfExists bool) (paymentID int, err error) {

	var count int
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM Payment WHERE OrderEntry = $1", entry.ID).Scan(&count)

	// If no payment exists for this entry, create one
	var payment Payment
	if count == 0 && !errorIfExists {
		payment = Payment{
			Sender:     entry.Sender,
			Receiver:   entry.Receiver,
			Amount:     entry.Price * entry.Quantity,
			Order:      null.NewInt(int64(orderID), true),
			OrderEntry: null.NewInt(int64(entry.ID), true),
			IsSale:     entry.IsSale,
			Item:       null.NewInt(int64(entry.Item), true),
			Quantity:   entry.Quantity,
			Price:      entry.Price,
		}
		paymentID, err = createPaymentTx(tx, payment)
	}

	return
}

// VerifyOrderAndCreatePayments sets payment order to verified and creates a payment for each order entry if it doesn't already exist
// This means if some payments have already been created with CreatePayedOrderEntries before verifying the order, they will be skipped
func (db *Database) VerifyOrderAndCreatePayments(orderID int, transactionTypeID int) (err error) {

	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { err = deferTx(tx, err) }()

	// Verify payment order
	_, err = tx.Exec(context.Background(), `
	UPDATE PaymentOrder
	SET Verified = True, TransactionTypeID = $1
	WHERE ID = $2
	`, transactionTypeID, orderID)
	if err != nil {
		log.Error(err)
	}

	// Get Paymentorder (including payments)
	order, err := db.GetOrderByID(orderID)

	// Create payments
	for _, entry := range order.Entries {
		_, err = createPaymentForOrderEntryTx(tx, orderID, entry, false)
	}

	return
}

// CreatePayedOrderEntries creates entries with a payment for an order
func (db *Database) CreatePayedOrderEntries(orderID int, entries []OrderEntry) (err error) {

	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() { err = deferTx(tx, err) }()

	// Create entries & associated payments
	for _, entry := range entries {
		entry, err = createOrderEntryTx(tx, orderID, entry)
		if err != nil {
			log.Error(err)
			return err
		}
		_, err = createPaymentForOrderEntryTx(tx, orderID, entry, false)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	return
}

// Payments -------------------------------------------------------------------

// ListPayments returns the payments from the database
func (db *Database) ListPayments(minDate time.Time, maxDate time.Time, vendorLicenseID string, filterPayouts bool, filterSales bool, filterNoPayout bool) (payments []Payment, err error) {
	var rows pgx.Rows

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
	query := "SELECT Payment.ID, Payment.Timestamp, Sender, Receiver, SenderAccount.Name, ReceiverAccount.Name, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, Item, Quantity, Price FROM Payment JOIN Account as SenderAccount ON SenderAccount.ID = Sender JOIN Account as ReceiverAccount ON ReceiverAccount.ID = Receiver"
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	// Order by timestamp
	query += " ORDER BY Payment.Timestamp"
	rows, err = db.Dbpool.Query(context.Background(), query, filterValues...)
	if err != nil {
		log.Error(err)
		return payments, err
	}

	// Scan rows
	for rows.Next() {
		var payment Payment
		err = rows.Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.SenderName, &payment.ReceiverName, &payment.Amount, &payment.AuthorizedBy, &payment.Order, &payment.OrderEntry, &payment.IsSale, &payment.Payout, &payment.Item, &payment.Quantity, &payment.Price)
		if err != nil {
			log.Error(err)
			return payments, err
		}

		// Add payout payments to main payment
		subrows, err := db.Dbpool.Query(context.Background(), "SELECT ID, Timestamp, Sender, Receiver, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, Item, Quantity, Price FROM Payment WHERE Payout = $1 ORDER BY Timestamp", payment.ID)
		if err != nil {
			log.Error(err)
			return payments, err
		}
		for subrows.Next() {
			var subpayment Payment
			err = subrows.Scan(&subpayment.ID, &subpayment.Timestamp, &subpayment.Sender, &subpayment.Receiver, &subpayment.Amount, &subpayment.AuthorizedBy, &subpayment.Order, &subpayment.OrderEntry, &subpayment.IsSale, &subpayment.Payout, &subpayment.Item, &subpayment.Quantity, &subpayment.Price)
			if err != nil {
				log.Error(err)
				return payments, err
			}
			payment.IsPayoutFor = append(payment.IsPayoutFor, subpayment)
		}

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
		log.Error(err)
	}
	return
}

// CreatePayment creates a payment in an transaction
func createPaymentTx(tx pgx.Tx, payment Payment) (paymentID int, err error) {

	// Validation
	if payment.Amount <= 0 {
		err = errors.New("Payment amount must be greater than 0")
		return
	}

	// Create payment
	err = tx.QueryRow(context.Background(), "INSERT INTO Payment (Sender, Receiver, Amount, AuthorizedBy, PaymentOrder, OrderEntry, IsSale, Payout, Item, Quantity, Price) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING ID", payment.Sender, payment.Receiver, payment.Amount, payment.AuthorizedBy, payment.Order, payment.OrderEntry, payment.IsSale, payment.Payout, payment.Item, payment.Quantity, payment.Price).Scan(&paymentID)
	if err != nil {
		log.Error(err)
		return
	}

	// Update account balances
	err = updateAccountBalanceTx(tx, payment.Sender, -payment.Amount)
	if err != nil {
		log.Error(err)
	}
	err = updateAccountBalanceTx(tx, payment.Receiver, payment.Amount)
	if err != nil {
		log.Error(err)
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
	defer func() { err = deferTx(tx, err) }()

	paymentID, err = createPaymentTx(tx, payment)

	return
}

// CreatePayments creates multiple payments
func (db *Database) CreatePayments(payments []Payment) (err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error(err)
		return err
	}
	defer func() { err = deferTx(tx, err) }()

	// Insert payments within the transaction
	for _, payment := range payments {
		_, err = createPaymentTx(tx, payment)
		if err != nil {
			log.Error(err)
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
		log.Error(err)
		return
	}
	defer func() { err = deferTx(tx, err) }()

	// Get cash account
	cashAccount, err := db.GetAccountByType("Cash")
	if err != nil {
		log.Error(err)
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
		log.Error(err)
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
			log.Error(err)
		}
	}

	// Update last payout date
	vendor.LastPayout = null.NewTime(time.Now(), true)
	err = db.UpdateVendor(vendor.ID, vendor)
	if err != nil {
		log.Error(err)
		return
	}

	return
}

// Accounts -------------------------------------------------------------------

// CreateAccount creates an account in the database
func (db *Database) CreateAccount(account Account) (id int, err error) {
	// TODO: Validate that User should only be filled if type is user_auth
	// Check if account.type = UserAuth
	// if account.Type == "UserAuth" && account.User.String == "" {
	// 	err = new (Error)

	// Define a slice of types, which should only exist once
	existOnceTypes := []string{"Cash", "Orga", "UserAnon", "VivaWallet", "Paypal"}

	// Check if an account of the specified type already exists
	if slices.Contains(existOnceTypes, account.Type) {
		var existingCount int
		err = db.Dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM Account WHERE Type = $1", account.Type).Scan(&existingCount)
		if err != nil {
			return 0, err
		}
		// If an account of the specified type already exists, return an error
		if existingCount > 0 {
			return 0, errors.New("An account of this type, which should exist only once, already exists: " + account.Type)
		}
	}

	// Insert the new account
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Account (Name, Type) values ($1, $2) RETURNING ID", account.Name, account.Type).Scan(&id)
	return id, err
}

// ListAccounts returns all accounts from the database
func (db *Database) ListAccounts() (accounts []Account, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "select * from Account")
	if err != nil {
		log.Error(err)
		return accounts, err
	}
	for rows.Next() {
		var account Account
		err = rows.Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
		if err != nil {
			log.Error(err)
			return accounts, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// GetAccountByID returns the account with the given ID
func (db *Database) GetAccountByID(id int) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE ID = $1", id).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "No rows in result set" {
			err = errors.New("account does not exist")
		}
		log.Error(err)
	}
	return
}

// GetOrCreateAccountByUserID returns the account with the given user
func (db *Database) GetOrCreateAccountByUserID(userID string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE UserID = $1", userID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "No rows in result set" {
			err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Account (Type, UserID) values ($1, $2) RETURNING *", "UserAuth", userID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
			log.Info("Created new account for user " + userID)
		} else {
			log.Error(err)
		}
	}
	return
}

// GetAccountByVendorID returns the account with the given vendor
func (db *Database) GetAccountByVendorID(vendorID int) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE Vendor = $1", vendorID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "No rows in result set" {
			err = errors.New("vendor does not exist or has no account")
		}
		log.Error(err, vendorID)
	}
	return
}

// Memory cache for account type id's
var accountTypeIDCache map[string]int

// GetAccountTypeID returns the (cached) ID of an account type
func (db *Database) GetAccountTypeID(accountType string) (accountTypeID int, err error) {
	if accountTypeIDCache == nil {
		accountTypeIDCache = make(map[string]int)
	}
	if accountTypeIDCache[accountType] != 0 {
		accountTypeID = accountTypeIDCache[accountType]
		return
	}
	err = db.Dbpool.QueryRow(context.Background(), "SELECT ID FROM Account WHERE Type = $1", accountType).Scan(&accountTypeID)
	if err != nil {
		log.Error(err)
		return
	}
	accountTypeIDCache[accountType] = accountTypeID
	return
}

// GetAccountByType returns the account with the given type
// Works only for types with a single entry in the database
func (db *Database) GetAccountByType(accountType string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE Type = $1", accountType).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		log.Error(err)
	}
	return
}

// updateAccountBalanceTx updates the balance of an account in an transaction
func updateAccountBalanceTx(tx pgx.Tx, id int, balanceDiff int) (err error) {

	var account Account

	// Lock account balance via "for update"
	// https://stackoverflow.com/a/45871295/19932351
	err = tx.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE ID = $1 for update", id).Scan(&account.Balance)
	if err != nil {
		log.Error(err)
	}
	newBalance := account.Balance + balanceDiff

	_, err = tx.Exec(context.Background(), `
	UPDATE Account
	SET Balance = $2
	WHERE ID = $1
	`, id, newBalance)
	if err != nil {
		log.Error(err)
	}

	return
}

// UpdateAccountBalanceByOpenPayments updates the balance of an account by summing up all open payments (i.e. payments without a payout)
func (db *Database) UpdateAccountBalanceByOpenPayments(vendorID int) (payoutAmount int, err error) {

	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return
	}

	// Provide defer func to commit or rollback transaction after function returns
	defer func() { err = deferTx(tx, err) }()

	// Get account
	vendorAccount, err := db.GetAccountByVendorID(vendorID)
	if err != nil {
		return
	}

	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE ID = $1", vendorAccount.ID).Scan(&vendorAccount.Balance)
	if err != nil {
		log.Error(err)
	}
	log.Info("UpdateAccountBalanceByOpenPayments: Balance of account where Vendor = " + strconv.Itoa(vendorID) + " is " + strconv.Itoa(vendorAccount.Balance))

	var openPaymentsReceiverSum int
	err = db.Dbpool.QueryRow(context.Background(), "SELECT COALESCE(SUM(Amount), 0) FROM Payment WHERE Payout IS NULL AND Paymentorder IS NOT NULL AND Receiver = $1", vendorAccount.ID).Scan(&openPaymentsReceiverSum)
	if err != nil {
		log.Error(err)
	}

	// Get open payments where vendor is sender
	var openPaymentsSenderSum int
	err = db.Dbpool.QueryRow(context.Background(), "SELECT COALESCE(SUM(Amount), 0) FROM Payment WHERE Payout IS NULL AND Paymentorder IS NOT NULL AND Sender = $1", vendorAccount.ID).Scan(&openPaymentsSenderSum)
	if err != nil {
		log.Error(err)
	}

	// Calculate new balance
	openPaymentsSum := openPaymentsReceiverSum - openPaymentsSenderSum

	_, err = tx.Exec(context.Background(), "UPDATE Account SET Balance = $1 WHERE ID = $2", openPaymentsSum, vendorAccount.ID)
	if err != nil {
		log.Error(err)
	}

	log.Info("UpdateAccountBalanceByOpenPayments: Updated balance of account " + strconv.Itoa(vendorID) + " from " + strconv.Itoa(vendorAccount.Balance) + " to " + strconv.Itoa(openPaymentsSum) + ", openPaymentsReceiverSum = " + strconv.Itoa(openPaymentsReceiverSum) + ", openPaymentsSenderSum = " + strconv.Itoa(openPaymentsSenderSum) + ")")

	return openPaymentsSum, err
}

// Settings (singleton) -------------------------------------------------------

// InitiateSettings creates default settings if they don't exist
func (db *Database) InitiateSettings() (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	INSERT INTO Settings (ID) VALUES (1)
	ON CONFLICT (ID) DO NOTHING;
	`)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

// GetSettings returns the settings from the database
func (db *Database) GetSettings() (Settings, error) {
	var settings Settings
	err := db.Dbpool.QueryRow(context.Background(), `
	SELECT Settings.ID, Color, FontColor, Logo, MainItem, MaxOrderAmount, OrgaCoversTransactionCosts, Name, Price, Description, Image from Settings LEFT JOIN Item ON Item.ID = MainItem LIMIT 1
	`).Scan(&settings.ID, &settings.Color, &settings.FontColor, &settings.Logo, &settings.MainItem, &settings.MaxOrderAmount, &settings.OrgaCoversTransactionCosts, &settings.MainItemName, &settings.MainItemPrice, &settings.MainItemDescription, &settings.MainItemImage)
	if err != nil {
		log.Error(err)
	}
	return settings, err
}

// UpdateSettings updates the settings in the database
func (db *Database) UpdateSettings(settings Settings) (err error) {

	_, err = db.Dbpool.Query(context.Background(), `
	UPDATE Settings
	SET Color = $1, FontColor = $2, Logo = $3, MainItem = $4, MaxOrderAmount = $5, OrgaCoversTransactionCosts = $6
	WHERE ID = 1
	`, settings.Color, settings.FontColor, settings.Logo, settings.MainItem, settings.MaxOrderAmount, settings.OrgaCoversTransactionCosts)

	if err != nil {
		log.Error(err)
	}

	return err
}

// DBSettings -----------------------------------------------------------------

// InitiateDBSettings creates default settings if they don't exist
func (db *Database) InitiateDBSettings() (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	INSERT INTO DBSettings (ID, isInitialized) VALUES (1, false)
	ON CONFLICT (ID) DO NOTHING;
	`)
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

// UpdateDBSettings updates the settings in the database
func (db *Database) UpdateDBSettings(dbsettings DBSettings) (err error) {
	_, err = db.Dbpool.Query(context.Background(), `
	UPDATE DBSettings
	SET isInitialized = $1
	WHERE ID = 1
	`, dbsettings.IsInitialized)

	if err != nil {
		log.Error(err)
	}

	return err
}

// GetDBSettings returns the settings from the database
func (db *Database) GetDBSettings() (DBSettings, error) {
	var dbsettings DBSettings
	err := db.Dbpool.QueryRow(context.Background(), `
	SELECT * from DBSettings LIMIT 1
	`).Scan(&dbsettings.ID, &dbsettings.IsInitialized)
	if err != nil {
		log.Error(err)
	}
	return dbsettings, err
}

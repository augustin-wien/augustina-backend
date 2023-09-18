package database

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

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

// ListVendors returns all users from the database
func (db *Database) ListVendors() (vendors []Vendor, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "SELECT vendor.ID, keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout, Balance from Vendor JOIN account ON account.vendor = vendor.id")
	if err != nil {
		log.Error(err)
		return vendors, err
	}
	for rows.Next() {
		var vendor Vendor
		err = rows.Scan(&vendor.ID, &vendor.KeycloakID, &vendor.URLID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.Balance)
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
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE LicenseID = $1", licenseID).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.URLID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout)
	if err != nil {
		log.Error(err)
	}

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error(err)
	}
	return
}

// CreateVendor creates a vendor and an associated account in the database
func (db *Database) CreateVendor(vendor Vendor) (vendorID int32, err error) {

	// Create vendor
	err = db.Dbpool.QueryRow(context.Background(), "insert into Vendor (keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout) values ($1, $2, $3, $4, $5, $6, $7) RETURNING ID", vendor.KeycloakID, vendor.URLID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout).Scan(&vendorID)
	if err != nil {
		log.Error(err)
	}

	// Create vendor account
	_, err = db.Dbpool.Exec(context.Background(), "insert into Account (Balance, Type, Vendor) values (0, 'Vendor', $1) RETURNING ID", vendorID)
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
	SET keycloakid = $1, urlid = $2, LicenseID = $3, FirstName = $4, LastName = $5, Email = $6, LastPayout = $7
	WHERE ID = $8
	`, vendor.KeycloakID, vendor.URLID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, id)
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
func (db *Database) ListItems() ([]Item, error) {
	var items []Item
	rows, err := db.Dbpool.Query(context.Background(), "SELECT * FROM Item")
	if err != nil {
		log.Error(err)
		return items, err
	}
	for rows.Next() {
		var item Item
		err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived)
		if err != nil {
			log.Error(err)
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}

// GetItem returns the item with the given ID
func (db *Database) GetItem(id int) (item Item, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Item WHERE ID = $1", id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived)
	if err != nil {
		log.Error(err)
	}
	return
}

// CreateItem creates an item in the database
func (db *Database) CreateItem(item Item) (id int32, err error) {
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
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Item (Name, Description, Price, LicenseItem, Archived) values ($1, $2, $3, $4, $5) RETURNING ID", item.Name, item.Description, item.Price, item.LicenseItem, item.Archived).Scan(&id)
	if err != nil {
		log.Error(err)
	}
	return id, err
}

// UpdateItem updates an item in the database
func (db *Database) UpdateItem(id int, item Item) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Item
	SET Name = $2, Description = $3, Price = $4, Image = $5, LicenseItem = $6, Archived = $7
	WHERE ID = $1
	`, id, item.Name, item.Description, item.Price, item.Image, item.LicenseItem, item.Archived)
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
	rows, err := db.Dbpool.Query(context.Background(), "SELECT ID, Item, Quantity, Price, Sender, Receiver FROM OrderEntry WHERE paymentorder = $1", orderID)
	if err != nil {
		log.Error(err)
		return
	}
	for rows.Next() {
		var entry OrderEntry
		err = rows.Scan(&entry.ID, &entry.Item, &entry.Quantity, &entry.Price, &entry.Sender, &entry.Receiver)
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
		err = rows.Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.Timestamp, &order.User, &order.Vendor)
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
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE ID = $1", id).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.Timestamp, &order.User, &order.Vendor)
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

	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE OrderCode = $1", OrderCode).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.Timestamp, &order.User, &order.Vendor)
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
// TODO: This should be a transaction
// Processes OrderCode, vendor, and items (trinkgeld is an item)
func (db *Database) CreateOrder(order Order) (orderID int, err error) {

	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO PaymentOrder (OrderCode, Vendor) values ($1, $2) RETURNING ID", order.OrderCode, order.Vendor).Scan(&orderID)
	if err != nil {
		log.Error(err)
		return
	}

	// Create order items
	for _, entry := range order.Entries {

		// Get current item price
		var item Item
		err = db.Dbpool.QueryRow(context.Background(), "SELECT Price FROM Item WHERE ID = $1", entry.Item).Scan(&item.Price)
		if err != nil {
			log.Error(err)
			return
		}

		// Create order item
		_, err = db.Dbpool.Exec(context.Background(), "INSERT INTO OrderEntry (Item, Price, Quantity, PaymentOrder, Sender, Receiver) values ($1, $2, $3, $4, $5, $6)", entry.Item, item.Price, entry.Quantity, orderID, entry.Sender, entry.Receiver)
		if err != nil {
			log.Error(err)
			return orderID, err
		}
	}

	return
}

// VerifyOrderAndCreatePayments verifies payment order, update it in the database, and create payments
func (db *Database) VerifyOrderAndCreatePayments(id int) (err error) {

	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return err
	}

	// Execute transaction after function returns
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

	// Verify payment order
	_, err = tx.Exec(context.Background(), `
	UPDATE PaymentOrder
	SET Verified = True
	WHERE ID = $1
	`, id)
	if err != nil {
		log.Error(err)
	}

	// Get Paymentorder (including payments)
	order, err := db.GetOrderByID(id)

	// Create payments
	for _, oi := range order.Entries {
		_, err := tx.Exec(context.Background(), "INSERT INTO Payment (Sender, Receiver, Amount, OrderEntry, PaymentOrder) values ($1, $2, $3, $4, $5)", oi.Sender, oi.Receiver, oi.Price*oi.Quantity, oi.ID, order.ID)
		if err != nil {
			return err
		}
	}

	return
}

// Payments -------------------------------------------------------------------

// ListPayments returns the payments from the database
func (db *Database) ListPayments() ([]Payment, error) {
	var payments []Payment
	rows, err := db.Dbpool.Query(context.Background(), "select * from payment")
	if err != nil {
		log.Error(err)
		return payments, err
	}
	for rows.Next() {
		var payment Payment
		err = rows.Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.Amount, &payment.AuthorizedBy, &payment.Order, &payment.OrderEntry)
		if err != nil {
			return payments, err
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

// CreatePayment creates a payment and returns the payment ID
func (db *Database) CreatePayment(payment Payment) (paymentID int, err error) {

	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Payment (Sender, Receiver, Amount) values ($1, $2, $3) RETURNING ID", payment.Sender, payment.Receiver, payment.Amount).Scan(&paymentID)
	if err != nil {
		log.Error(err)
		return
	}

	return
}

// CreatePayments creates multiple payments in a transcation
func (db *Database) CreatePayments(payments []Payment) (err error) {

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
		_, err := tx.Exec(context.Background(), "INSERT INTO Payment (Sender, Receiver, Amount) values ($1, $2, $3)", payment.Sender, payment.Receiver, payment.Amount)
		if err != nil {
			return err
		}
	}
	return nil
}

// Accounts -------------------------------------------------------------------

// CreateAccount creates an account in the database
func (db *Database) CreateAccount(account Account) (id int, err error) {
	// TODO: Validate that User should only be filled if type is user_auth
	// Check if account.type = UserAuth
	// if account.Type == "UserAuth" && account.User.String == "" {
	// 	err = new (Error)

	// Define a slice of types, which should only exist once
	existOnceTypes := []string{"Cash", "Orga", "UserAnon"}

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
		if err.Error() == "no rows in result set" {
			err = errors.New("account does not exist")
		}
		log.Error(err)
	}
	return
}

// GetAccountByUser returns the account with the given user
func (db *Database) GetAccountByUser(user string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE User = $1", user).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "no rows in result set" {
			err = errors.New("user does not exist or has no account")
		}
		log.Error(err)
	}
	return
}

// GetAccountByVendorID returns the account with the given vendor
func (db *Database) GetAccountByVendorID(vendorID int) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE Vendor = $1", vendorID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "no rows in result set" {
			err = errors.New("vendor does not exist or has no account")
		}
		log.Error(err)
	}
	return
}

// GetAccountByType returns the account with the given type
// Works only for types with a single entry in the database
// TODO: This could easily be cached
func (db *Database) GetAccountByType(accountType string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE Type = $1", accountType).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		log.Error(err)
	}
	return
}

// UpdateAccountBalance updates the balance of an account in the database
func (db *Database) UpdateAccountBalance(id int, balance int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Account
	SET Balance = $2
	WHERE ID = $1
	`, id, balance)
	if err != nil {
		log.Error(err)
	}

	// DEBUG
	account, err := db.GetAccountByID(id)
	log.Info("Updated balance for account ", id, " is ", account.Balance)

	return
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
	SELECT * from Settings LIMIT 1
	`).Scan(&settings.ID, &settings.Color, &settings.Logo, &settings.MainItem, &settings.RefundFees)
	if err != nil {
		log.Error(err)
	}
	return settings, err
}

// UpdateSettings updates the settings in the database
func (db *Database) UpdateSettings(settings Settings) (err error) {

	_, err = db.Dbpool.Query(context.Background(), `
	UPDATE Settings
	SET Color = $1, Logo = $2, MainItem = $3, RefundFees = $4
	WHERE ID = 1
	`, settings.Color, settings.Logo, settings.MainItem, settings.RefundFees)

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

package database

import (
	"augustin/config"
	"augustin/keycloak"
	"augustin/mailer"
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
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

// Users ----------------------------------------------------------------------

// ListVendors returns all users from the database but not all fields for better overview
func (db *Database) ListVendors() (vendors []Vendor, err error) {
	rows, err := db.Dbpool.Query(context.Background(), `
		SELECT vendor.ID, LicenseID, FirstName, LastName, LastPayout, Balance 
		FROM Vendor 
		JOIN Account ON Account.vendor = Vendor.id 
		WHERE Account.Type = 'Vendor' 
		ORDER BY LicenseID ASC
	`)
	if err != nil {
		log.Error("ListVendors", err)
		return vendors, err
	}
	defer rows.Close()

	for rows.Next() {
		var vendor Vendor
		err = rows.Scan(&vendor.ID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.LastPayout, &vendor.Balance)
		if err != nil {
			log.Error("ListVendors", err)
			return vendors, err
		}
		vendors = append(vendors, vendor)
	}

	return vendors, nil
}

// GetVendorByLicenseID returns the vendor with the given licenseID
func (db *Database) GetVendorByLicenseID(licenseID string) (vendor Vendor, err error) {
	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE LicenseID = $1", licenseID).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap,
		&vendor.HasSmartphone, &vendor.HasBankAccount, &vendor.IsDeleted, &vendor.AccountProofUrl)
	if err != nil {
		log.Info("GetVendorByLicenseID: Couldn't get vendor: ", licenseID, err)
		return vendor, err
	}

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendorByLicenseID: couldn't get balance: ", err)
	}
	return vendor, err
}

// GetVendorByEmail returns the vendor with the given licenseID
func (db *Database) GetVendorByEmail(mail string) (vendor Vendor, err error) {
	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE Email = $1", mail).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap, &vendor.HasSmartphone, &vendor.HasBankAccount, &vendor.IsDeleted, &vendor.AccountProofUrl)
	if err != nil {
		log.Error("GetVendorByEmail: Couldn't get vendor ", mail, err)
		return vendor, err
	}

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendorByEmail: Couldn't get balance: ", err)
	}
	return vendor, err
}

// GetVendor returns the vendor with the given id
func (db *Database) GetVendor(vendorID int) (vendor Vendor, err error) {
	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE ID = $1", vendorID).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap, &vendor.HasSmartphone, &vendor.HasBankAccount, &vendor.IsDeleted, &vendor.AccountProofUrl)
	if err != nil {
		log.Error("GetVendor: Couldn't get vendor ", vendorID, err)
		return vendor, err
	}
	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendor: ", err)
	}
	return vendor, err
}

// GetVendorWithBalanceUpdate returns the vendor with the given id
func (db *Database) GetVendorWithBalanceUpdate(vendorID int) (vendor Vendor, err error) {

	// Update Account balance by open payments
	_, err = db.UpdateAccountBalanceByOpenPayments(vendorID)
	if err != nil {
		log.Error("GetVendor: ", err)
	}

	// Get vendor data
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Vendor WHERE ID = $1", vendorID).Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.IsDisabled, &vendor.Longitude, &vendor.Latitude, &vendor.Address, &vendor.PLZ, &vendor.Location, &vendor.WorkingTime, &vendor.Language, &vendor.Comment, &vendor.Telephone, &vendor.RegistrationDate, &vendor.VendorSince, &vendor.OnlineMap, &vendor.HasSmartphone, &vendor.HasBankAccount, &vendor.IsDeleted, &vendor.AccountProofUrl)
	if err != nil {
		log.Error("GetVendor: Couldn't get vendor ", vendorID, err)
		return vendor, err
	}
	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendor: ", err)
	}
	return vendor, err
}

// CreateVendor creates a vendor and an associated account in the database
func (db *Database) CreateVendor(vendor Vendor) (vendorID int, err error) {

	// Create vendor
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Vendor (Keycloakid, UrlID, LicenseID, FirstName, LastName, Email, LastPayout, IsDisabled, Longitude, Latitude, Address, PLZ, Location, WorkingTime, Language, Comment, Telephone, RegistrationDate, VendorSince, OnlineMap, HasSmartphone, HasBankAccount) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22) RETURNING ID", vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, vendor.IsDisabled, vendor.Longitude, vendor.Latitude, vendor.Address, vendor.PLZ, vendor.Location, vendor.WorkingTime, vendor.Language, vendor.Comment, vendor.Telephone, vendor.RegistrationDate, vendor.VendorSince, vendor.OnlineMap, vendor.HasSmartphone, vendor.HasBankAccount).Scan(&vendorID)
	if err != nil {
		log.Errorf("CreateVendor: create vendor %s %+v", vendor.Email, err)
		return
	}

	// Create vendor account
	_, err = db.Dbpool.Exec(context.Background(), "INSERT INTO Account (Name, Balance, Type, Vendor) values ($1, 0, $2, $3) RETURNING ID", vendor.LicenseID, "Vendor", vendorID)
	if err != nil {
		log.Error("CreateVendor: create vendor account %s %+v", vendor.Email, err)
		return
	}

	return
}

// UpdateVendor updates a vendor in the database
func (db *Database) UpdateVendor(id int, vendor Vendor) (err error) {
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("UpdateVendor: Failed to begin transaction: ", err)
		return err
	}
	defer func() {
		if err := tx.Rollback(context.Background()); err != nil && err != sql.ErrTxDone {
			log.Error("tx.Rollback failed: %v", err)
		}
	}()

	// Update the Vendor table
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Vendor
	SET keycloakid = $1, UrlID = $2, LicenseID = $3, FirstName = $4, LastName = $5, Email = $6, LastPayout = $7, IsDisabled = $8, Longitude = $9, Latitude = $10, Address = $11, PLZ = $12, Location = $13, WorkingTime = $14, Language = $15, Comment = $16, Telephone = $17, RegistrationDate = $18, VendorSince = $19, OnlineMap = $20, HasSmartphone = $21, HasBankAccount = $22, AccountProofUrl = $23
	WHERE ID = $24
	`, vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, vendor.IsDisabled, vendor.Longitude, vendor.Latitude, vendor.Address, vendor.PLZ, vendor.Location, vendor.WorkingTime, vendor.Language, vendor.Comment, vendor.Telephone, vendor.RegistrationDate, vendor.VendorSince, vendor.OnlineMap, vendor.HasSmartphone, vendor.HasBankAccount, vendor.AccountProofUrl, id)
	if err != nil {
		log.Error("UpdateVendor: Failed to update Vendor: ", err)
		return err
	}

	// Commit transaction
	if err = tx.Commit(context.Background()); err != nil {
		log.Error("UpdateVendor: Failed to commit transaction: ", err)
		return err
	}

	return nil
}

// DeleteVendor deletes a user in the database and the associated account
func (db *Database) DeleteVendor(vendorID int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM Account
	WHERE Vendor = $1
	`, vendorID)
	if err != nil {
		log.Error("DeleteVendor: ", err)
	}

	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM Vendor
	WHERE ID = $1
	`, vendorID)
	if err != nil {
		log.Error("DeleteVendor: ", err)
	}

	return
}

// Items ----------------------------------------------------------------------

// ListItems returns all items from the database
func (db *Database) ListItems(skipHiddenItems bool, skipLicenses bool) ([]Item, error) {
	var items []Item
	rows, err := db.Dbpool.Query(context.Background(), "SELECT * FROM Item ORDER BY ItemOrder DESC")
	if err != nil {
		log.Error("ListItems: ", err)
		return items, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem, &item.LicenseGroup, &item.IsPDFItem, &item.PDF, &item.ItemOrder, &item.ItemColor, &item.ItemTextColor)
		if err != nil {
			log.Error("ListItems: ", err)
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
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Item WHERE Name = $1", name).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem, &item.LicenseGroup, &item.IsPDFItem, &item.PDF, &item.ItemOrder, &item.ItemColor, &item.ItemTextColor)
	if err != nil {
		log.Error("GetItemByName: ", err)
	}
	return
}

// GetItem returns the item with the given ID
func (db *Database) GetItem(id int) (item Item, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Item WHERE ID = $1", id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem, &item.LicenseGroup, &item.IsPDFItem, &item.PDF, &item.ItemOrder, &item.ItemColor, &item.ItemTextColor)
	if err != nil {
		log.Error("GetItem: failed in Getitem() ", err)
	}
	return
}

// GetItemTx returns the item with the given ID
func (db *Database) GetItemTx(tx pgx.Tx, id int) (item Item, err error) {
	err = tx.QueryRow(context.Background(), "SELECT * FROM Item WHERE ID = $1", id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image, &item.LicenseItem, &item.Archived, &item.IsLicenseItem, &item.LicenseGroup, &item.IsPDFItem, &item.PDF, &item.ItemOrder, &item.ItemColor, &item.ItemTextColor)
	if err != nil {
		log.Error("GetItem: failed in GetItemTx() ", err)
	}
	return
}

// CreateItem creates an item in the database
func (db *Database) CreateItem(item Item) (id int, err error) {
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: Opening DBPool failed", err)
		return 0, err
	}

	defer func() { err = DeferTx(tx, err) }()
	// Check if the item name already exists
	var count int
	err = db.Dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM Item WHERE Name = $1", item.Name).Scan(&count)
	if err != nil {
		log.Error("CreateItem: failed to select item ", err)
		return 0, err
	}
	if count > 0 {
		return 0, errors.New("Item with the same name already exists. Update it or delete it first")
	}

	// Insert the new item
	err = db.Dbpool.QueryRow(context.Background(), `
	INSERT INTO Item
	(Name, Description, Price, Image, LicenseItem, Archived, IsLicenseItem, LicenseGroup, IsPDFItem, PDF)
	values ($1, $2, $3, '', $4, $5, $6, $7, $8, NULL)
	RETURNING ID
	`, item.Name, item.Description, item.Price, item.LicenseItem, item.Archived, item.IsLicenseItem, item.LicenseGroup, item.IsPDFItem).Scan(&id)
	if err != nil {
		log.Error("CreateItem: failed to insert item ", err)
	}
	return id, err
}

// UpdateItem updates an item in the database
func (db *Database) UpdateItem(id int, item Item) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Item
	SET Name = $2, Description = $3, Price = $4, Image = $5, LicenseItem = $6, Archived = $7, IsLicenseItem = $8, LicenseGroup = $9, IsPDFItem = $10, PDF = $11, ItemOrder = $12, ItemColor = $13, ItemTextColor = $14
	WHERE ID = $1
	`, id, item.Name, item.Description, item.Price, item.Image, item.LicenseItem, item.Archived, item.IsLicenseItem, item.LicenseGroup, item.IsPDFItem, item.PDF, item.ItemOrder, item.ItemColor, item.ItemTextColor)
	if err != nil {
		log.Error("DB UpdateItem: ", err)
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
		log.Error("DeleteItem: ", err)
	}
	return
}

// Orders ---------------------------------------------------------------------

// GetOrderEntries returns all entries of an order
func (db *Database) GetOrderEntries(orderID int) (entries []OrderEntry, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "SELECT OrderEntry.ID, Item, Quantity, Price, Sender, Receiver, SenderAccount.Name, ReceiverAccount.Name, IsSale FROM OrderEntry JOIN Account as SenderAccount ON SenderAccount.ID = Sender JOIN Account as ReceiverAccount ON ReceiverAccount.ID = Receiver WHERE paymentorder = $1 ", orderID)
	if err != nil {
		log.Error("GetOrderEntries: ", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var entry OrderEntry
		err = rows.Scan(&entry.ID, &entry.Item, &entry.Quantity, &entry.Price, &entry.Sender, &entry.Receiver, &entry.SenderName, &entry.ReceiverName, &entry.IsSale)
		if err != nil {
			log.Error("GetOrderEntries: ", err)
			return
		}
		entries = append(entries, entry)
	}
	return
}
func (db *Database) GetOrderEntriesTx(tx pgx.Tx, orderID int) (entries []OrderEntry, err error) {
	rows, err := tx.Query(context.Background(), "SELECT OrderEntry.ID, Item, Quantity, Price, Sender, Receiver, SenderAccount.Name, ReceiverAccount.Name, IsSale FROM OrderEntry JOIN Account as SenderAccount ON SenderAccount.ID = Sender JOIN Account as ReceiverAccount ON ReceiverAccount.ID = Receiver WHERE paymentorder = $1 ", orderID)
	if err != nil {
		log.Error("GetOrderEntriesTx: ", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry OrderEntry
		err = rows.Scan(&entry.ID, &entry.Item, &entry.Quantity, &entry.Price, &entry.Sender, &entry.Receiver, &entry.SenderName, &entry.ReceiverName, &entry.IsSale)
		if err != nil {
			log.Error("GetOrderEntriesTx: ", err)
			return
		}
		entries = append(entries, entry)
	}
	return
}

// DeleteOrderEntry deletes an entry in the database
func (db *Database) DeleteOrderEntry(id int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM OrderEntry
	WHERE ID = $1
	`, id)
	if err != nil {
		log.Error("DeleteOrderEntry: ", err)
	}
	return
}

// GetOrders returns all orders from the database
func (db *Database) GetOrders() (orders []Order, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "SELECT *, null as entries FROM PaymentOrder")
	if err != nil {
		log.Error("GetOrders: ", err)
		return orders, err
	}
	defer rows.Close()
	tmpOrders, err := pgx.CollectRows(rows, pgx.RowToStructByName[Order])
	if err != nil {
		log.Error("GetOrders: failed to collect rows: ", err)
		return orders, err
	}
	for _, order := range tmpOrders {
		// Add entries to order
		order.Entries, err = db.GetOrderEntries(order.ID)
		if err != nil {
			log.Error("GetOrders: failed to get order entries: ", err)
		}
		orders = append(orders, order)
	}

	return
}

// GetOrderByID returns Order by OrderID
func (db *Database) GetOrderByID(id int) (order Order, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE ID = $1", id).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.TransactionTypeID, &order.Timestamp, &order.User, &order.Vendor, &order.CustomerEmail)
	if err != nil {
		log.Error("GetOrderByID: ", err)
		return
	}

	// Add entries to order
	order.Entries, err = db.GetOrderEntries(order.ID)
	if err != nil {
		log.Error("GetOrderByID failed to add entries: ", err)
	}

	return
}

// GetOrderByIDTx returns Order by OrderID
func (db *Database) GetOrderByIDTx(tx pgx.Tx, id int) (order Order, err error) {
	err = tx.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE ID = $1", id).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.TransactionTypeID, &order.Timestamp, &order.User, &order.Vendor, &order.CustomerEmail)
	if err != nil {
		log.Error("GetOrderByIDTx: ", err)
		return
	}

	// Add entries to order
	order.Entries, err = db.GetOrderEntriesTx(tx, order.ID)
	if err != nil {
		log.Error("GetOrderByIDTx failed to add entries: ", err)
	}

	return
}

// GetOrderByOrderCode returns Order by OrderCode
func (db *Database) GetOrderByOrderCode(OrderCode string) (order Order, err error) {

	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE OrderCode = $1", OrderCode).Scan(&order.ID, &order.OrderCode, &order.TransactionID, &order.Verified, &order.TransactionTypeID, &order.Timestamp, &order.User, &order.Vendor, &order.CustomerEmail)
	if err != nil {
		log.Error("GetOrderByOrderCode: ", err)
		return
	}

	// Add items to order
	order.Entries, err = db.GetOrderEntries(order.ID)
	if err != nil {
		log.Error("GetOrderByOrderCode: ", err, " orderID: ", order.ID)
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
	defer func() {
		err = DeferTx(tx, err)
		if err != nil {
			log.Error("CreateOrder: reached defer error ", err)
		}
	}()

	err = tx.QueryRow(context.Background(), "INSERT INTO PaymentOrder (OrderCode, Vendor, CustomerEmail) values ($1, $2, $3) RETURNING ID", order.OrderCode, order.Vendor, order.CustomerEmail).Scan(&orderID)
	if err != nil {
		log.Error("CreateOrder failed: ", err)
		return
	}

	// Create order items
	for _, entry := range order.Entries {
		_, err = createOrderEntryTx(tx, orderID, entry)
		if err != nil {
			log.Errorf("CreateOrder create order entries: %+v %+v", entry, err)
			return
		}
	}

	return
}

// DeleteOrder deletes an order in the database
func (db *Database) DeleteOrder(id int) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	DELETE FROM PaymentOrder
	WHERE ID = $1
	`, id)
	if err != nil {
		log.Error("DeleteOrder: ", err)
	}
	return
}

// createOrderEntryTx adds an entry to an order in an transaction
func createOrderEntryTx(tx pgx.Tx, orderID int, entry OrderEntry) (OrderEntry, error) {

	// Get current item price
	var item Item
	err := tx.QueryRow(context.Background(), "SELECT Price FROM Item WHERE ID = $1", entry.Item).Scan(&item.Price)
	if err != nil {
		log.Error("createOrderEntryTx: query row", err)
		return entry, err
	}
	entry.Price = item.Price

	// Create order entry
	err = tx.QueryRow(context.Background(), "INSERT INTO OrderEntry (Item, Price, Quantity, PaymentOrder, Sender, Receiver, IsSale) values ($1, $2, $3, $4, $5, $6, $7) RETURNING ID", entry.Item, entry.Price, entry.Quantity, orderID, entry.Sender, entry.Receiver, entry.IsSale).Scan(&entry.ID)
	if err != nil {
		log.Error("createOrderEntryTx: insert ", err)
	}
	return entry, err
}

// createPaymentForOrderEntryTx creates a payment for an order entry
func createPaymentForOrderEntryTx(tx pgx.Tx, orderID int, entry OrderEntry, errorIfExists bool) (paymentID int, err error) {

	// Check if payment already exists for this entry
	var count int
	err = tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM Payment WHERE OrderEntry = $1", entry.ID).Scan(&count)
	if err != nil {
		log.Error("createPaymentForOrderEntryTx: ", err)
		return
	}

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
		log.Error("VerifyOrderAndCreatePayments: Opening DBPool failed", err)
		return err
	}

	defer func() { err = DeferTx(tx, err) }()
	// Verify payment order
	_, err = tx.Exec(context.Background(), `
	UPDATE PaymentOrder
	SET Verified = True, TransactionTypeID = $1
	WHERE ID = $2
	`, transactionTypeID, orderID)
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: update payment order", orderID, err)
	}

	// Get Paymentorder (including payments)
	order, err := db.GetOrderByIDTx(tx, orderID)
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: get order by id", orderID, err)
		return err
	}
	if order.CustomerEmail.Valid && order.CustomerEmail.String != "" {
		for _, entry := range order.Entries {
			item, err := db.GetItemTx(tx, entry.Item)
			if err != nil {
				log.Error("VerifyOrderAndCreatePayments: failed to get item: ", orderID, err)
			}
			if item.LicenseItem.Valid {

				if !item.IsPDFItem {
					// add customer to licenseItemGroup

					customer, err := keycloak.KeycloakClient.GetOrCreateUser(order.CustomerEmail.String)
					if err != nil {
						log.Error("VerifyOrderAndCreatePayments: failed to create keycloak customer: ", orderID, err)
					}
					// add customer to customer group
					err = keycloak.KeycloakClient.AssignGroup(customer, "customer")
					if err != nil {
						log.Error("VerifyOrderAndCreatePayments: failed to assign customer to group: ", orderID, err)
					}
					err = keycloak.KeycloakClient.AssignDigitalLicenseGroup(customer, item.LicenseGroup.String)
					if err != nil {
						log.Error("VerifyOrderAndCreatePayments: failed to assign customer to license group: ", orderID, err)
					}
					// Send email with link to the license Item
					templateData := struct {
						URL string
					}{
						URL: config.Config.OnlinePaperUrl,
					}
					receivers := []string{order.CustomerEmail.String}
					mail, err := mailer.NewRequestFromTemplate(receivers, "A new newspaper has been purchased", "digitalLicenceItemTemplate.html", templateData)
					if err != nil {
						log.Error("VerifyOrderAndCreatePayments: failed to create mail: ", orderID, err)
					}
					success, err := mail.SendEmail()
					if err != nil || !success {
						log.Error("VerifyOrderAndCreatePayments: failed to send mail: ", orderID, err)
					}
				} else {
					// Generate download link and send it to the
					if !item.PDF.Valid {
						log.Error("VerifyOrderAndCreatePayments: item has no pdf: oder id: ", orderID, "itemid: ", item.ID, err)
					}
					pdf_id := item.PDF.ValueOrZero()
					// TODO: check if pdf exists
					pdf, err := db.GetPDFByID(pdf_id)
					if err != nil {
						log.Error("VerifyOrderAndCreatePayments: failed to get pdf: orderid", orderID, "item", item.ID, err)
					}
					// Check if link already created for Download

					pdfDownload, err := db.GetPDFDownloadByOrderIdAndItemTx(tx, orderID, item.ID)

					if err != nil {
						log.Debug("VerifyOrderAndCreatePayments:create pdf download: ", orderID, item.ID, err)
						pdfDownload, err = db.CreatePDFDownload(tx, pdf, orderID, item.ID)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to create pdf download: ", orderID, err)
						}
					}

					if !pdfDownload.EmailSent {
						url := config.Config.FrontendURL + "/pdf/" + pdfDownload.LinkID
						templateData := struct {
							URL string
						}{
							URL: url,
						}
						receivers := []string{order.CustomerEmail.String}
						mail, err := mailer.NewRequestFromTemplate(receivers, "Deine Zeitung ist bereit zum Download", "PDFLicenceItemTemplate.html", templateData)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to create mail: ", orderID, err)
						}
						success, err := mail.SendEmail()
						if err != nil || !success {
							log.Error("VerifyOrderAndCreatePayments: failed to send mail: ", orderID, err)
						}
						pdfDownload.EmailSent = true
						pdfDownload.OrderID = null.IntFrom(int64(orderID))
						pdfDownload.ItemID = null.IntFrom(int64(item.ID))
						err = db.UpdatePdfDownloadTx(tx, pdfDownload)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments; failed to update pdfdownload ", err)
						}
					}

				}

			}
		}
	}
	// Create payments
	for _, entry := range order.Entries {
		_, err = createPaymentForOrderEntryTx(tx, orderID, entry, false)
		if err != nil {
			log.Error("VerifyOrderAndCreatePayments: create payments for order entry: ", orderID, err)
			return err
		}
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
	defer func() { err = DeferTx(tx, err) }()

	// Create entries & associated payments
	for _, entry := range entries {
		entry, err = createOrderEntryTx(tx, orderID, entry)
		if err != nil {
			log.Error("CreatePayedOrderEntries: create order entry", err)
			return err
		}
		_, err = createPaymentForOrderEntryTx(tx, orderID, entry, false)
		if err != nil {
			log.Error("CreatePayedOrderEntries: create payment for order entry", err)
			return err
		}
	}

	return
}

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

// Accounts -------------------------------------------------------------------

// CreateSingleAccount creates an account in the database
func (db *Database) CreateSpecialVendorAccount(vendor Vendor) (vendorID int, err error) {

	// Create a new vendor account
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Vendor (Keycloakid, UrlID, LicenseID, FirstName, LastName, Email, LastPayout, IsDisabled, Longitude, Latitude, Address, PLZ, Location, WorkingTime, Language, Comment, Telephone, RegistrationDate, VendorSince, OnlineMap, HasSmartphone, HasBankAccount) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22) RETURNING ID", "", "", vendor.LicenseID, "", "", vendor.Email, time.Now(), false, 0, 0, "", "", "", "", "", "", "", "", "", false, false, false).Scan(&vendorID)
	if err != nil {
		log.Errorf("CreateSpecialVendor: create vendor %s %+v", vendor.Email, err)
		return
	}

	_, err = db.Dbpool.Exec(context.Background(), "INSERT INTO Account (Name, Balance, Type, Vendor) values ($1, 0, $2, $3) RETURNING ID", vendor.LicenseID, vendor.LicenseID, vendorID)
	if err != nil {
		log.Error("CreateSpecialVendor: create vendor account %s %+v", vendor.Email, err)
		return
	}

	return vendorID, err
}

// ListAccounts returns all accounts from the database
func (db *Database) ListAccounts() (accounts []Account, err error) {
	rows, err := db.Dbpool.Query(context.Background(), "select * from Account")
	if err != nil {
		log.Error("ListAccounts: ", err)
		return accounts, err
	}
	defer rows.Close()
	for rows.Next() {
		var account Account
		err = rows.Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
		if err != nil {
			log.Error("ListAccounts: ", err)
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
		log.Error("GetAccountByID: ", err)
	}
	return
}

// GetOrCreateAccountByUserID returns the account with the given user
func (db *Database) GetOrCreateAccountByUserID(userID string) (account Account, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM Account WHERE UserID = $1", userID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
	if err != nil {
		if err.Error() == "no rows in result set" {
			err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO Account (Type, UserID) values ($1, $2) RETURNING *", "UserAuth", userID).Scan(&account.ID, &account.Name, &account.Balance, &account.Type, &account.User, &account.Vendor)
			log.Info("Created new account for user " + userID)
		} else {
			log.Error("GetOrCreateAccountByUserID: ", err)
		}
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
		log.Error("GetAccountByVendorID: ", err, vendorID)
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
		log.Error("GetAccountTypeID: ", accountType, err)
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
		log.Error("GetAccountByType: ", err)
	}
	return
}

// updateAccountBalanceTx updates the balance of an account in an transaction
func updateAccountBalanceTx(tx pgx.Tx, id int, balanceDiff int) (err error) {

	var account Account

	// Lock account balance via "for update"
	// https://stackoverflow.com/a/45871295/19932351
	err = tx.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE ID = $1", id).Scan(&account.Balance)
	if err != nil {
		log.Error("updateAccountBalanceTx: query row", err)
	}
	newBalance := account.Balance + balanceDiff

	_, err = tx.Exec(context.Background(), `
	UPDATE Account
	SET Balance = $2
	WHERE ID = $1
	`, id, newBalance)
	if err != nil {
		log.Error("updateAccountBalanceTx: update ", err)
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
	defer func() { err = DeferTx(tx, err) }()

	// Get account
	vendorAccount, err := db.GetAccountByVendorID(vendorID)
	if err != nil {
		return
	}

	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE ID = $1", vendorAccount.ID).Scan(&vendorAccount.Balance)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}
	log.Info("UpdateAccountBalanceByOpenPayments: Balance of account where Vendor = " + strconv.Itoa(vendorID) + " is " + strconv.Itoa(vendorAccount.Balance))

	var openPaymentsReceiverSum int
	err = db.Dbpool.QueryRow(context.Background(), "SELECT COALESCE(SUM(Amount), 0) FROM Payment WHERE Payout IS NULL AND Paymentorder IS NOT NULL AND Receiver = $1", vendorAccount.ID).Scan(&openPaymentsReceiverSum)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}

	// Get open payments where vendor is sender
	var openPaymentsSenderSum int
	err = db.Dbpool.QueryRow(context.Background(), "SELECT COALESCE(SUM(Amount), 0) FROM Payment WHERE Payout IS NULL AND Paymentorder IS NOT NULL AND Sender = $1", vendorAccount.ID).Scan(&openPaymentsSenderSum)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
	}

	// Calculate new balance
	openPaymentsSum := openPaymentsReceiverSum - openPaymentsSenderSum

	_, err = tx.Exec(context.Background(), "UPDATE Account SET Balance = $1 WHERE ID = $2", openPaymentsSum, vendorAccount.ID)
	if err != nil {
		log.Error("UpdateAccountBalanceByOpenPayments: ", err)
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
		log.Error("InitiateSettings: ", err)
		return err
	}
	return err
}

// GetSettings returns the settings from the database
func (db *Database) GetSettings() (Settings, error) {
	var settings Settings
	err := db.Dbpool.QueryRow(context.Background(), `
	SELECT Settings.ID, Color, FontColor, Logo, MainItem, MaxOrderAmount, OrgaCoversTransactionCosts, Name, Price, Description, Image, WebshopIsClosed, VendorNotFoundHelpUrl, MaintainanceModeHelpUrl, VendorEmailPostfix, NewspaperName, QRCodeUrl, QRCodeLogoImgUrl, AGBUrl, MapCenterLat, MapCenterLong, UseVendorLicenseIdInShop, Favicon, QrCodeSettings  from Settings LEFT JOIN Item ON Item.ID = MainItem LIMIT 1
	`).Scan(&settings.ID, &settings.Color, &settings.FontColor,
		&settings.Logo, &settings.MainItem, &settings.MaxOrderAmount,
		&settings.OrgaCoversTransactionCosts, &settings.MainItemName,
		&settings.MainItemPrice, &settings.MainItemDescription, &settings.MainItemImage,
		&settings.WebshopIsClosed, &settings.VendorNotFoundHelpUrl,
		&settings.MaintainanceModeHelpUrl, &settings.VendorEmailPostfix,
		&settings.NewspaperName, &settings.QRCodeUrl, &settings.QRCodeLogoImgUrl,
		&settings.AGBUrl, &settings.MapCenterLat, &settings.MapCenterLong,
		&settings.UseVendorLicenseIdInShop,
		&settings.Favicon, &settings.QRCodeSettings,
	)
	if err != nil {
		log.Error("GetSettings: ", err)
	}
	return settings, err
}

// UpdateSettings updates the settings in the database
func (db *Database) UpdateSettings(settings Settings) (err error) {

	tx, err := db.Dbpool.Begin(context.Background())
	defer func() { err = DeferTx(tx, err) }()
	if err != nil {
		log.Error("UpdateSettings failed to access db pool: ", err)
		return err
	}

	_, err = tx.Exec(context.Background(), `
	UPDATE Settings
	SET Color = $1, FontColor = $2, Logo = $3, MainItem = $4, MaxOrderAmount = $5, OrgaCoversTransactionCosts = $6, WebshopIsClosed = $7, VendorNotFoundHelpUrl = $8, MaintainanceModeHelpUrl = $9, VendorEmailPostfix = $10, NewspaperName = $11, QRCodeUrl = $12, QRCodeLogoImgUrl = $13, AGBUrl = $14, MapCenterLat = $15, MapCenterLong = $16, UseVendorLicenseIdInShop = $17, Favicon = $18, QrCodeSettings = $19
	WHERE ID = 1`,
		settings.Color, settings.FontColor, settings.Logo,
		settings.MainItem, settings.MaxOrderAmount,
		settings.OrgaCoversTransactionCosts, settings.WebshopIsClosed,
		settings.VendorNotFoundHelpUrl, settings.MaintainanceModeHelpUrl, settings.VendorEmailPostfix,
		settings.NewspaperName, settings.QRCodeUrl,
		settings.QRCodeLogoImgUrl, settings.AGBUrl, settings.MapCenterLat, settings.MapCenterLong,
		settings.UseVendorLicenseIdInShop,
		&settings.Favicon, &settings.QRCodeSettings,
	)
	if err != nil {
		log.Error("db UpdateSettings: ", err)
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
		log.Error("InitiateDBSettings: ", err)
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
		log.Error("UpdateDBSettings: ", err)
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
		log.Error("GetDBSettings: ", err)
	}
	return dbsettings, err
}

// Online Map -----------------------------------------------------------------

// LocationData is used to return the location data of a vendor for the online map
type LocationData struct {
	ID        int         `json:"id"`
	FirstName string      `json:"firstName"`
	LicenseID null.String `json:"licenseID"`
	Longitude float64     `json:"longitude"`
	Latitude  float64     `json:"latitude"`
}

// GetVendorLocations returns a list of all longitudes and latitudes given by the vendors table
func (db *Database) GetVendorLocations() (locationData []LocationData, err error) {
	rows, err := db.Dbpool.Query(context.Background(), `
	SELECT vendor.ID, LicenseID, FirstName, Longitude, Latitude 
	from Vendor 
	JOIN Account ON Account.Vendor = Vendor.id 
	WHERE Account.Type = 'Vendor'
	`)
	if err != nil {
		log.Error("GetVendorLocations: ", err)
		return locationData, err
	}
	for rows.Next() {
		var nextLocationData LocationData
		err = rows.Scan(&nextLocationData.ID, &nextLocationData.LicenseID, &nextLocationData.FirstName, &nextLocationData.Longitude, &nextLocationData.Latitude)
		if err != nil {
			log.Error("GetVendorLocations: ", err)
			return locationData, err
		}
		locationData = append(locationData, nextLocationData)
	}
	return locationData, nil
}

// DeletePDF removes pdfs if their creation date is older than 6 weeks
func (db *Database) DeletePDF() (err error) {
	deleteInterval := config.Config.IntervalToDeletePDFsInWeeks
	log.Info("DeletePDF entered: ", deleteInterval)
	_, err = db.Dbpool.Exec(context.Background(), "DELETE FROM PDF WHERE timestamp < NOW() - $1 * INTERVAL '1 week'", deleteInterval)
	if err != nil {
		log.Error("DeletePDF: ", err)
		return err
	}
	return err
}

// CreatePDF creates an instance of the PDF with given path and timestamp into the database
func (db *Database) CreatePDF(pdf PDF) (pdfId int64, err error) {

	// CreatePDF creates an instance of the PDF with given path and timestamp into the database
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO PDF (Path, Timestamp) values ($1, $2) RETURNING ID", pdf.Path, pdf.Timestamp).Scan(&pdf.ID)
	if err != nil {
		log.Error("CreatePDF: failed to add to database ", err)
	}
	if err != nil {
		log.Error("CreatePDF: ", err)
	}
	log.Info("Created new PDF with ID: ", pdf.ID, " and path: ", pdf.Path)
	return int64(pdf.ID), err
}

// GetPDF returns the latest PDF from the database
func (db *Database) GetPDF() (pdf PDF, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDF ORDER BY ID DESC LIMIT 1").Scan(&pdf.ID, &pdf.Path, &pdf.Timestamp)
	if err != nil {
		log.Error("GetPDF: ", err)
	}
	return pdf, err
}

// GetPDFByID returns the PDF with the given ID
func (db *Database) GetPDFByID(id int64) (pdf PDF, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDF WHERE ID = $1", id).Scan(&pdf.ID, &pdf.Path, &pdf.Timestamp)
	if err != nil {
		log.Error("GetPDFByID: failed for id:", id, err)
	}
	return pdf, err
}

// CreatePDFDownload creates an instance of the PDFDownload with given linkID and timestamp into the database
func (db *Database) CreatePDFDownload(tx pgx.Tx, pdf PDF, orderId, itemId int) (pdfDownload PDFDownload, err error) {
	// generate download id
	linkID := uuid.New()
	pdfDownload = PDFDownload{
		LinkID:        linkID.String(),
		PDF:           pdf.ID,
		Timestamp:     time.Now(),
		EmailSent:     false,
		LastDownload:  time.Time{},
		DownloadCount: 0,
		OrderID:       null.IntFrom(int64(orderId)),
		ItemID:        null.IntFrom(int64(itemId)),
	}

	// CreatePDF creates an instance of the PDF with given path and timestamp into the database
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO PDFDownload (LinkID, PDF, Timestamp, OrderId, ItemId) values ($1, $2, $3, $4, $5) RETURNING ID", pdfDownload.LinkID, pdfDownload.PDF, pdfDownload.Timestamp, pdfDownload.OrderID, pdfDownload.ItemID).Scan(&pdfDownload.ID)
	if err != nil {
		log.Error("CreatePDFDownload: ", err)
	}
	return
}

// GetPDFDownload returns the latest PDFDownload from the database
func (db *Database) GetPDFDownload(linkID string) (pdfDownload PDFDownload, err error) {
	if len(linkID) == 0 {
		return pdfDownload, errors.New("linkID is empty")
	}
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDFDownload WHERE LinkID = $1", linkID).Scan(&pdfDownload.ID, &pdfDownload.PDF, &pdfDownload.LinkID, &pdfDownload.Timestamp, &pdfDownload.EmailSent, &pdfDownload.OrderID, &pdfDownload.LastDownload, &pdfDownload.DownloadCount, &pdfDownload.ItemID)
	if err != nil {
		log.Error("GetPDFDownload: ", linkID, "err: ", err)
	}
	return pdfDownload, err
}

// GetPDFDownload returns the latest PDFDownload from the database
func (db *Database) GetPDFDownloadTx(tx pgx.Tx, linkID string) (pdfDownload PDFDownload, err error) {
	if len(linkID) == 0 {
		return pdfDownload, errors.New("linkID is empty")
	}
	err = tx.QueryRow(context.Background(), "SELECT * FROM PDFDownload WHERE LinkID = $1", linkID).Scan(&pdfDownload.ID, &pdfDownload.PDF, &pdfDownload.LinkID, &pdfDownload.Timestamp, &pdfDownload.EmailSent, &pdfDownload.OrderID, &pdfDownload.LastDownload, &pdfDownload.DownloadCount, &pdfDownload.ItemID)
	if err != nil {
		log.Error("GetPDFDownload: ", linkID, "err: ", err)
	}
	return pdfDownload, err
}

// DeletePDFDownload removes pdfs if their creation date is older than 6 weeks
func (db *Database) DeletePDFDownload() (err error) {
	// Get interval from config
	deleteInterval := config.Config.IntervalToDeletePDFsInWeeks
	_, err = db.Dbpool.Exec(context.Background(), "DELETE FROM PDFDownload WHERE timestamp < NOW() - $1 * INTERVAL '1 week'", deleteInterval)
	if err != nil {
		log.Error("DeletePDFDownload: ", err)
		return err
	}
	return err
}

func (db *Database) UpdatePdfDownloadTx(tx pgx.Tx, pdfDownload PDFDownload) (err error) {
	_, err = tx.Exec(context.Background(), `
	UPDATE PDFDownload SET PDF = $1, LinkID = $2, Timestamp = $3, EmailSent = $4, OrderID = $5, LastDownload = $6, DownloadCount = $7, ItemId = $8 WHERE ID = $9`,
		pdfDownload.PDF, pdfDownload.LinkID, pdfDownload.Timestamp, pdfDownload.EmailSent, pdfDownload.OrderID, pdfDownload.LastDownload, pdfDownload.DownloadCount, pdfDownload.ItemID, pdfDownload.ID)
	if err != nil {
		log.Error("UpdatePdfDownload: ", err)
	}
	return err
}

func (db *Database) UpdatePdfDownload(pdfDownload PDFDownload) (err error) {
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("UpdatePdfDownload: failed to start transaction ", err)
		return
	}
	defer func() {
		err = DeferTx(tx, err)
		if err != nil {

			log.Error("UpdatePdfDownload: reached defer error ", err)
		}
	}()
	log.Info("UpdatePdfDownload: pdfDownload: ", pdfDownload)
	return db.UpdatePdfDownloadTx(tx, pdfDownload)
}

func (db *Database) GetPDFDownloadByOrderIdTx(tx pgx.Tx, order int) (pdfDownload []PDFDownload, err error) {
	rows, err := tx.Query(context.Background(), "SELECT * FROM PDFDownload WHERE OrderId = $1", order)
	if err != nil {
		log.Error("GetPDFDownloadByOrderIdTx: ", err)
		return pdfDownload, err
	}
	defer rows.Close()
	for rows.Next() {
		var nextPdfDownload PDFDownload
		err = rows.Scan(&nextPdfDownload.ID, &nextPdfDownload.PDF, &nextPdfDownload.LinkID, &nextPdfDownload.Timestamp, &nextPdfDownload.EmailSent, &nextPdfDownload.OrderID, &nextPdfDownload.LastDownload, &nextPdfDownload.DownloadCount, &nextPdfDownload.ItemID)
		if err != nil {
			log.Error("GetPDFDownloadByOrderIdTx: ", err)
			return pdfDownload, err
		}
		pdfDownload = append(pdfDownload, nextPdfDownload)
	}
	return pdfDownload, nil
}

func (db *Database) GetPDFDownloadByOrderId(order int) (pdfDownload []PDFDownload, err error) {
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		return
	}
	defer func() {
		err = DeferTx(tx, err)
		if err != nil {

			log.Error("GetPDFDownloadByOrderId: reached defer error ", err)
		}
	}()
	return db.GetPDFDownloadByOrderIdTx(tx, order)
}

func (db *Database) GetPDFDownloadByOrderIdAndItemTx(tx pgx.Tx, order int, item int) (pdfDownload PDFDownload, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PDFDownload WHERE OrderId = $1 AND ItemId = $2", order, item).Scan(&pdfDownload.ID, &pdfDownload.PDF, &pdfDownload.LinkID, &pdfDownload.Timestamp, &pdfDownload.EmailSent, &pdfDownload.OrderID, &pdfDownload.LastDownload, &pdfDownload.DownloadCount, &pdfDownload.ItemID)
	return pdfDownload, err
}

package database

import (
	"context"

	"go.uber.org/zap"
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
	rows, err := db.Dbpool.Query(context.Background(), "select vendor.ID, keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout, Balance from Vendor JOIN account ON account.vendor = vendor.id")
	if err != nil {
		log.Error(err)
		return vendors, err
	}
	for rows.Next() {
		var vendor Vendor
		err = rows.Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.Balance)
		if err != nil {
			log.Error(err)
			return vendors, err
		}
		vendors = append(vendors, vendor)
	}
	return vendors, nil
}

// CreateVendor creates a vendor and an associated account in the database
func (db *Database) CreateVendor(vendor Vendor) (vendorID int32, err error) {

	// Create vendor
	err = db.Dbpool.QueryRow(context.Background(), "insert into Vendor (keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout) values ($1, $2, $3, $4, $5, $6, $7) RETURNING ID", vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout).Scan(&vendorID)
	if err != nil {
		log.Error(err)
	}

	// Create vendor account
	_, err = db.Dbpool.Exec(context.Background(), "insert into Account (Balance, Type, Vendor) values (0, 'vendor', $1) RETURNING ID", vendorID)
	if err != nil {
		log.Error(err)
		return
	}

	return
}

// UpdateVendor Updates a user in the database
func (db *Database) UpdateVendor(id int, vendor Vendor) (err error) {
	log.Info("updating")
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Vendor
	SET keycloakid = $1, urlid = $2, LicenseID = $3, FirstName = $4, LastName = $5, Email = $6, LastPayout = $7
	WHERE ID = $8
	`, vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, id)
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

func (db *Database) ListItems() ([]Item, error) {
	var items []Item
	rows, err := db.Dbpool.Query(context.Background(), "SELECT * FROM Item")
	if err != nil {
		log.Error(err)
		return items, err
	}
	for rows.Next() {
		var item Item
		err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Image)
		if err != nil {
			log.Error(err)
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (db *Database) CreateItem(item Item) (id int32, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into Item (Name, Description, Price) values ($1, $2, $3) RETURNING ID", item.Name, item.Description, item.Price).Scan(&id)
	return id, err
}

func (db *Database) UpdateItem(id int, item Item) (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Item
	SET Name = $2, Price = $3, Image = $4
	WHERE ID = $1
	`, id, item.Name, item.Price, item.Image)
	if err != nil {
		log.Error(err)
	}
	return
}

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

// PaymentOrders --------------------------------------------------------------

// GetOrder returns Order by TransactionID
func (db *Database) GetPaymentOrderByTransactionID(TransactionID string) (order PaymentOrder, err error) {

	err = db.Dbpool.QueryRow(context.Background(), "SELECT * FROM PaymentOrder WHERE TransactionID = $1", TransactionID).Scan(&order.ID, &order.TransactionID, &order.Verified, &order.Timestamp, &order.Vendor)

	if err != nil {
		log.Error(err)
		return
	}

	// Add items to order
	rows, err := db.Dbpool.Query(context.Background(), "SELECT ID, Item, Quantity, Price FROM OrderItem WHERE PaymentOrder = $1", order.ID)
	if err != nil {
		log.Error(err)
		return
	}
	for rows.Next() {
		var orderItem PaymentOrderItem
		err = rows.Scan(&orderItem.ID, &orderItem.ItemID, &orderItem.Quantity, &orderItem.Price)
		if err != nil {
			log.Error(err)
			return
		}
		order.OrderItems = append(order.OrderItems, orderItem)
	}

	return
}

// Create Payment Order
// Processes transactionID, vendor, and items (trinkgeld is an item)
func (db *Database) CreatePaymentOrder(order PaymentOrder) (orderID int, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "INSERT INTO PaymentOrder (TransactionID, Vendor) values ($1, $2, $3, $4, $5, $6, $7) RETURNING ID", order.TransactionID, order.Vendor).Scan(&orderID)
	if err != nil {
		log.Error(err)
	}

	// Create order items
	for _, orderItem := range order.OrderItems {

		// Get current item price
		var item Item
		err = db.Dbpool.QueryRow(context.Background(), "SELECT Price FROM Item WHERE ID = $1", orderItem.ItemID).Scan(&item.Price)
		if err != nil {
			log.Error(err)
			return
		}

		// Create order item
		_, err = db.Dbpool.Exec(context.Background(), "INSERT INTO OrderItem (Item, Price, Quantity, PaymentOrder) values ($1, $2, $3, $4)", orderItem.ItemID, item.Price, orderItem.Quantity, orderID)
		if err != nil {
			log.Error(err)
			return
		}
	}

	return
}

// // ListVendors returns all users from the database
// func (db *Database) ListOrders() (Orders []Order, err error) {
// 	rows, err := db.Dbpool.Query(context.Background(), "select Order.ID, keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout, Balance from Order JOIN account ON account.Order = Order.id")
// 	if err != nil {
// 		log.Error(err)
// 		return Orders, err
// 	}
// 	for rows.Next() {
// 		var Order Order
// 		err = rows.Scan(&Order.ID, &Order.KeycloakID, &Order.UrlID, &Order.LicenseID, &Order.FirstName, &Order.LastName, &Order.Email, &Order.LastPayout, &Order.Balance)
// 		if err != nil {
// 			log.Error(err)
// 			return Orders, err
// 		}
// 		Orders = append(Orders, Order)
// 	}
// 	return Orders, nil
// }

// // CreateOrder creates a Order and an associated account in the database
// func (db *Database) CreateOrder(Order Order) (OrderID int32, err error) {



// // UpdateOrder Updates a user in the database
// func (db *Database) UpdateOrder(id int, Order Order) (err error) {
// 	log.Info("updating")
// 	_, err = db.Dbpool.Exec(context.Background(), `
// 	UPDATE Order
// 	SET keycloakid = $1, urlid = $2, LicenseID = $3, FirstName = $4, LastName = $5, Email = $6, LastPayout = $7
// 	WHERE ID = $8
// 	`, Order.KeycloakID, Order.UrlID, Order.LicenseID, Order.FirstName, Order.LastName, Order.Email, Order.LastPayout, id)
// 	if err != nil {
// 		log.Error(err)
// 	}

// 	return
// }

// // DeleteOrder deletes a user in the database and the associated account
// func (db *Database) DeleteOrder(OrderID int) (err error) {
// 	_, err = db.Dbpool.Exec(context.Background(), `
// 	DELETE FROM Order
// 	WHERE ID = $1
// 	`, OrderID)
// 	if err != nil {
// 		log.Error(err)
// 	}

// 	_, err = db.Dbpool.Exec(context.Background(), `
// 	DELETE FROM Account
// 	WHERE Order = $1
// 	`, OrderID)
// 	if err != nil {
// 		log.Error(err)
// 	}

// 	return
// }

// Payments -------------------------------------------------------------------

// GetPayments returns the payments from the database
func (db *Database) ListPayments() ([]Payment, error) {
	var payments []Payment
	rows, err := db.Dbpool.Query(context.Background(), "select * from payment")
	if err != nil {
		log.Error(err)
		return payments, err
	}
	for rows.Next() {
		var payment Payment
		err = rows.Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.Amount, &payment.AuthorizedBy, &payment.PaymentOrderID, &payment.OrderItemID)
		if err != nil {
			return payments, err
		}
		payments = append(payments, payment)
	}
	return payments, nil
}


// Create multiple payments
func (db *Database) CreatePayments(payments []Payment) (err error) {

	log.Info("CreatePayments called")

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


func (db *Database) CreateAccount(account Account) (id int32, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into Account (Name) values ($1) RETURNING ID", account.Name).Scan(&id)
	return id, err
}


// Settings (singleton) -------------------------------------------------------

// Create default settings if they don't exist
func (db *Database) InitiateSettings() (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	INSERT INTO Settings (ID) VALUES (1);
	`)
	if err != nil {
		log.Error(err)
	}
	return
}

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

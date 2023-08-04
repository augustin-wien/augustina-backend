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
	rows, err := db.Dbpool.Query(context.Background(), "select vendor.ID, keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout, Account, Balance from Vendor JOIN account ON account = account.id")
	if err != nil {
		log.Error(err)
		return vendors, err
	}
	for rows.Next() {
		var vendor Vendor
		err = rows.Scan(&vendor.ID, &vendor.KeycloakID, &vendor.UrlID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.Email, &vendor.LastPayout, &vendor.Account, &vendor.Balance)
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
	// Create vendor account
	var accountID int32
	err = db.Dbpool.QueryRow(context.Background(), "insert into Account (Balance) values (0) RETURNING ID").Scan(&accountID)
	if err != nil {
		log.Error(err)
		return
	}

	// Create vendor
	err = db.Dbpool.QueryRow(context.Background(), "insert into Vendor (keycloakid, urlid, LicenseID, FirstName, LastName, Email, LastPayout, Account) values ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING ID", vendor.KeycloakID, vendor.UrlID, vendor.LicenseID, vendor.FirstName, vendor.LastName, vendor.Email, vendor.LastPayout, accountID).Scan(&vendorID)
	if err != nil {
		log.Error(err)
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
	WHERE ID = (SELECT Account FROM Vendor WHERE ID = $1)
	`, vendorID)
	if err != nil {
		log.Error(err)
	}

	return
}


// Payments -------------------------------------------------------------------

// GetPayments returns the payments from the database
func (db *Database) GetPayments() ([]Payment, error) {
	var payments []Payment
	rows, err := db.Dbpool.Query(context.Background(), "select * from payment")
	if err != nil {
		log.Error("GetPayments failed", zap.Error(err))
		return payments, err
	}
	for rows.Next() {
		var payment Payment
		err = rows.Scan(&payment.ID, &payment.Timestamp, &payment.Sender, &payment.Receiver, &payment.Type, &payment.Amount, &payment.AuthorizedBy, &payment.Item, &payment.PaymentBatch)
		if err != nil {
			return payments, err
		}
		payments = append(payments, payment)
	}
	return payments, nil
}

// Create payment type
func (db *Database) CreatePaymentType(pt PaymentType) (id int32, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into PaymentType (Name) values ($1) RETURNING ID", pt.Name).Scan(&id)
	return id, err
}

// Create account
func (db *Database) CreateAccount(account Account) (id int32, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into Account (Name) values ($1) RETURNING ID", account.Name).Scan(&id)
	return id, err
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
		_, err := tx.Exec(context.Background(), "INSERT INTO Payment ( Sender, Receiver, Type, Amount) values ($1, $2, $3, $4)", payment.Sender, payment.Receiver, payment.Type, payment.Amount)
		if err != nil {
			return err
		}
	}
	return nil
}



func (db *Database) ListItems() ([]Item, error) {
	var items []Item
	rows, err := db.Dbpool.Query(context.Background(), "select * from items")
	if err != nil {
		return items, err
	}
	for rows.Next() {
		var item Item
		err = rows.Scan(&item.ID, &item.Name, &item.Price)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, nil
}
func (db *Database) CreateItem(item Item) (id int32, err error) {
	err = db.Dbpool.QueryRow(context.Background(), "insert into Item (Name, Price) values ($1, $2) RETURNING ID", item.Name, item.Price).Scan(&id)
	return id, err
}
func (db *Database) UpdateItem(item Item) (err error) {

	// TODO: Throw error if is_editable = False
	_, err = db.Dbpool.Exec(context.Background(), `
	UPDATE Item
	SET Name = $2, Price = $3, Image = $4
	WHERE ID = $1
	`, item.ID, item.Name, item.Price, item.Image)

	if err != nil {
		panic(err)
	}

	// Set items
	// for _, item := range settings.Items {
	// 	_, err = db.Dbpool.Query(context.Background(), `
	// 	INSERT INTO Items (Name, Price) VALUES ($1, $2)
	// 	ON CONFLICT (Name)
	// 	DO UPDATE SET Name = $1, Price = $2
	// 	`, item.Name, item.Price)
	// 	if err != nil {
	// 		log.Errorf("SetSettings failed: %v\n", err)
	// 	}
	// }
	return err
}

func (db *Database) GetSettings() (Settings, error) {
	var settings Settings
	err := db.Dbpool.QueryRow(context.Background(), `select * from Settings LIMIT 1`).Scan(&settings.ID, &settings.Color, &settings.Logo)
	if err != nil {
		log.Error("GetSettings failed", zap.Error(err))
	}
	return settings, err
}

func (db *Database) UpdateSettings(settings Settings) (err error) {

	_, err = db.Dbpool.Query(context.Background(), `
	INSERT INTO Settings (Color, Logo) VALUES ($1, $2)
	ON CONFLICT (ID)
	DO UPDATE SET Color = $1, Logo = $2
	`, settings.Color, settings.Logo)

	if err != nil {
		log.Error("SetSettings failed:", zap.Error(err))
	}

	// Set items
	// for _, item := range settings.Items {
	// 	_, err = db.Dbpool.Query(context.Background(), `
	// 	INSERT INTO Items (Name, Price) VALUES ($1, $2)
	// 	ON CONFLICT (Name)
	// 	DO UPDATE SET Name = $1, Price = $2
	// 	`, item.Name, item.Price)
	// 	if err != nil {
	// 		log.Errorf("SetSettings failed: %v\n", err)
	// 	}
	// }
	return err
}

func (db *Database) GetVendorSettings() (string, error) {
	var settings string
	err := db.Dbpool.QueryRow(context.Background(), `select '{"credit":1.61,"qrcode":"/img/Augustin-QR-Code.png","idnumber":"123456789"}'`).Scan(&settings)
	if err != nil {
		log.Error("QueryRow failed:", zap.Error(err))
		return "", err
	}
	return settings, nil
}

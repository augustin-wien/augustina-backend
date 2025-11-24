package database

import (
	"context"
	"errors"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

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

	// Get current item price and disabled flag
	var item Item
	err := tx.QueryRow(context.Background(), "SELECT Price, Disabled FROM Item WHERE ID = $1", entry.Item).Scan(&item.Price, &item.Disabled)
	if err != nil {
		log.Error("createOrderEntryTx: query row", err)
		return entry, err
	}
	if item.Disabled {
		log.Debug("createOrderEntryTx: item is disabled", zap.Int("item_id", entry.Item))
		return entry, errors.New("item is disabled")
	}
	entry.Price = item.Price

	// Debug: log sender/receiver and ensure accounts exist in this transaction
	log.Debug("createOrderEntryTx: inserting order entry", zap.Int("orderID", orderID), zap.Int("item", entry.Item), zap.Int("sender", entry.Sender), zap.Int("receiver", entry.Receiver))
	var tmp int
	err = tx.QueryRow(context.Background(), "SELECT ID FROM Account WHERE ID = $1", entry.Sender).Scan(&tmp)
	if err != nil {
		log.Error("createOrderEntryTx: sender account lookup failed", zap.Int("sender", entry.Sender), zap.Error(err))
	}
	err = tx.QueryRow(context.Background(), "SELECT ID FROM Account WHERE ID = $1", entry.Receiver).Scan(&tmp)
	if err != nil {
		log.Error("createOrderEntryTx: receiver account lookup failed", zap.Int("receiver", entry.Receiver), zap.Error(err))
	}

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
		log.Error("createPaymentForOrderEntryTx: count query failed", zap.Int("entryID", entry.ID), zap.Error(err))
		return
	}
	log.Debug("createPaymentForOrderEntryTx: payment count for entry", zap.Int("entryID", entry.ID), zap.Int("count", count))

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
		log.Debug("createPaymentForOrderEntryTx: creating payment", zap.Int("orderID", orderID), zap.Int("entryID", entry.ID), zap.Int("sender", payment.Sender), zap.Int("receiver", payment.Receiver), zap.Int("amount", payment.Amount))
		paymentID, err = createPaymentTx(tx, payment)
	}

	return
}

// VerifyOrderAndCreatePayments sets payment order to verified and creates a payment for each order entry if it doesn't already exist
// This means if some payments have already been created with CreatePayedOrderEntries before verifying the order, they will be skipped
func (db *Database) VerifyOrderAndCreatePayments(orderID int, transactionTypeID int) (err error) {
	log.Info("VerifyOrderAndCreatePayments: Verifying order ", orderID)
	// Start a transaction
	tx, err := db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: Opening DBPool failed", err)
		return err
	}
	defer func() { err = DeferTx(tx, err) }()
	// Read current verified state before updating so we can detect a transition
	var alreadyVerified bool
	err = tx.QueryRow(context.Background(), "SELECT Verified FROM PaymentOrder WHERE ID = $1", orderID).Scan(&alreadyVerified)
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: read payment order verified flag", orderID, err)
		// continue — we'll still try to update and proceed
	}

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

	if !alreadyVerified && order.CustomerEmail.Valid && order.CustomerEmail.String != "" {

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
						URL   string
						EMAIL string
					}{
						URL:   config.Config.OnlinePaperUrl,
						EMAIL: order.CustomerEmail.String,
					}

					receivers := []string{order.CustomerEmail.String}
					mail, err := db.BuildEmailRequestFromTemplate("digitalLicenceItemTemplate.html", receivers, templateData)
					if err != nil {
						log.Error("VerifyOrderAndCreatePayments: failed to create mail: ", orderID, err)
					} else if mail != nil {
						// use subject from DB template (do not override)
						go func() {
							success, err := mail.SendEmail()
							if err != nil || !success {
								log.Error("VerifyOrderAndCreatePayments: failed to send mail: ", orderID, err)
							}
						}()
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
							URL   string
							EMAIL string
						}{
							URL:   url,
							EMAIL: order.CustomerEmail.String,
						}
						receivers := []string{order.CustomerEmail.String}
						mail, err := db.BuildEmailRequestFromTemplate("PDFLicenceItemTemplate.html", receivers, templateData)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to create mail: ", orderID, err)
						} else if mail != nil {
							// use subject from DB template (do not override)
							go func() {
								success, err := mail.SendEmail()
								if err != nil || !success {
									log.Error("VerifyOrderAndCreatePayments: failed to send mail: ", orderID, err)
								}
							}()
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
	log.Info("VerifyOrderAndCreatePayments: Creating payments for order ", orderID)
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

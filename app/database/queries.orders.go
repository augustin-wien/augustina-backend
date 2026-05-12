package database

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/ent/order"
	"github.com/augustin-wien/augustina-backend/ent/orderentry"
	"github.com/augustin-wien/augustina-backend/ent/payment"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// orderLocks stores a mutex per order ID to serialize payment creation
// for a given order and avoid concurrent duplicate creation.
var orderLocks sync.Map // map[int]*sync.Mutex

// lockOrder acquires a mutex for the given orderID and returns an unlock function.
func lockOrder(orderID int) func() {
	v, _ := orderLocks.LoadOrStore(orderID, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return func() {
		m.Unlock()
	}
}

// Orders ---------------------------------------------------------------------

// GetOrderEntries returns all entries of an order
func (db *Database) GetOrderEntries(orderID int) (entries []OrderEntry, err error) {
	res, err := db.EntClient.OrderEntry.Query().
		Where(orderentry.OrderID(orderID)).
		WithSender().
		WithReceiver().
		All(context.Background())
	if err != nil {
		log.Error("GetOrderEntries: ", err)
		return
	}
	for _, e := range res {
		entries = append(entries, convertOrderEntry(e))
	}
	return
}
func (db *Database) GetOrderEntriesTx(tx *ent.Tx, orderID int) (entries []OrderEntry, err error) {
	res, err := tx.OrderEntry.Query().
		Where(orderentry.OrderID(orderID)).
		WithSender().
		WithReceiver().
		All(context.Background())

	if err != nil {
		log.Error("GetOrderEntriesTx: ", err)
		return
	}

	for _, e := range res {
		entries = append(entries, convertOrderEntry(e))
	}

	return
}

// DeleteOrderEntry deletes an entry in the database
func (db *Database) DeleteOrderEntry(id int) (err error) {
	err = db.EntClient.OrderEntry.DeleteOneID(id).Exec(context.Background())
	if err != nil {
		log.Error("DeleteOrderEntry: ", err)
	}
	return
}

// GetOrders returns all orders from the database
func (db *Database) GetOrders() (orders []Order, err error) {
	res, err := db.EntClient.Order.Query().
		Order(ent.Desc(order.FieldTimestamp)).
		WithEntries(func(q *ent.OrderEntryQuery) {
			q.WithSender().WithReceiver()
		}).
		All(context.Background())

	if err != nil {
		log.Error("GetOrders: ", err)
		return nil, err
	}

	for _, o := range res {
		orders = append(orders, convertOrder(o))
	}

	return
}

// GetUnverifiedOrders returns all unverified orders from the database
func (db *Database) GetUnverifiedOrders() (orders []Order, err error) {
	res, err := db.EntClient.Order.Query().
		Where(order.Verified(false)).
		Order(ent.Desc(order.FieldTimestamp)).
		WithEntries(func(q *ent.OrderEntryQuery) {
			q.WithSender().WithReceiver()
		}).
		All(context.Background())

	if err != nil {
		log.Error("GetUnverifiedOrders: ", err)
		return nil, err
	}

	for _, o := range res {
		orders = append(orders, convertOrder(o))
	}

	return
}

// GetOrderByID returns Order by OrderID
func (db *Database) GetOrderByID(id int) (o Order, err error) {
	res, err := db.EntClient.Order.Query().
		Where(order.ID(id)).
		WithEntries(func(q *ent.OrderEntryQuery) {
			q.WithSender().WithReceiver()
		}).
		Only(context.Background())

	if err != nil {
		log.Error("GetOrderByID: ", err)
		return
	}

	return convertOrder(res), nil
}

// GetOrderByIDTx returns Order by OrderID
func (db *Database) GetOrderByIDTx(tx *ent.Tx, id int) (o Order, err error) {
	res, err := tx.Order.Query().
		Where(order.ID(id)).
		WithEntries(func(q *ent.OrderEntryQuery) {
			q.WithSender().WithReceiver()
		}).
		Only(context.Background())

	if err != nil {
		log.Error("GetOrderByIDTx: ", err)
		return
	}

	return convertOrder(res), nil
}

// GetOrderByOrderCode returns Order by OrderCode
func (db *Database) GetOrderByOrderCode(OrderCode string) (o Order, err error) {
	if OrderCode == "" {
		err = errors.New("GetOrderByOrderCode: order code is empty")
		return
	}
	res, err := db.EntClient.Order.Query().
		Where(order.OrderCode(OrderCode)). // Assuming order.OrderCode works as predicate
		WithEntries(func(q *ent.OrderEntryQuery) {
			q.WithSender().WithReceiver()
		}).
		Only(context.Background())

	if err != nil {
		log.Error("GetOrderByOrderCode: failed to get order", err, " orderCode: ", OrderCode)
		return
	}

	return convertOrder(res), nil
}

// CreateOrder creates an order in the database
// Processes OrderCode, vendor, and items (trinkgeld is an item)
func (db *Database) CreateOrder(o Order) (orderID int, err error) {

	// Start a transaction
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	tCreate := tx.Order.Create().
		SetVendorID(o.Vendor).
		SetVerified(o.Verified).
		SetTransactionTypeID(o.TransactionTypeID).
		SetTimestamp(time.Now().UTC())

	if o.TransactionID != "" {
		tCreate.SetTransactionID(o.TransactionID)
	} else {
		tCreate.SetTransactionID("manual-" + time.Now().Format(time.RFC3339Nano))
	}

	if o.VerifiedAt.Valid {
		tCreate.SetVerifiedAt(o.VerifiedAt.Time)
	}

	if o.OrderCode.Valid {
		tCreate.SetOrderCode(o.OrderCode.String)
	}
	if o.CustomerEmail.Valid {
		tCreate.SetCustomerEmail(o.CustomerEmail.String)
	}

	oRes, err := tCreate.Save(context.Background())
	if err != nil {
		log.Error("CreateOrder failed: ", err)
		return
	}
	orderID = oRes.ID

	// Create order items
	for _, entry := range o.Entries {
		_, err = createOrderEntryTx(tx, orderID, entry)
		if err != nil {
			log.Errorf("CreateOrder create order entries: %+v %+v", entry, err)
			return
		}
	}

	err = tx.Commit()
	return
}

// DeleteOrder deletes an order in the database
func (db *Database) DeleteOrder(id int) (err error) {
	err = db.EntClient.Order.DeleteOneID(id).Exec(context.Background())
	if err != nil {
		log.Error("DeleteOrder: ", err)
	}
	return
}

// createOrderEntryTx adds an entry to an order in an transaction
func createOrderEntryTx(tx *ent.Tx, orderID int, entry OrderEntry) (OrderEntry, error) {

	// Get current item price and disabled flag
	itemRes, err := tx.Item.Get(context.Background(), entry.Item)
	if err != nil {
		log.Error("createOrderEntryTx: query row", err)
		return entry, err
	}
	if itemRes.Disabled {
		log.Debug("createOrderEntryTx: item is disabled", zap.Int("item_id", entry.Item))
		return entry, errors.New("item is disabled")
	}
	entry.Price = int(math.Round(itemRes.Price))

	// Create order entry
	oeRes, err := tx.OrderEntry.Create().
		SetItemID(entry.Item).
		SetPrice(entry.Price).
		SetQuantity(entry.Quantity).
		SetOrderID(orderID).
		SetSenderID(entry.Sender).
		SetReceiverID(entry.Receiver).
		SetIsSale(entry.IsSale).
		Save(context.Background())

	if err != nil {
		log.Error("createOrderEntryTx: insert ", err)
		return entry, err
	}
	entry.ID = oeRes.ID
	return entry, err
}

// createPaymentForOrderEntryTx creates a payment for an order entry
func createPaymentForOrderEntryTx(tx *ent.Tx, orderID int, entry OrderEntry, errorIfExists bool) (paymentID int, err error) {

	// Check if payment already exists for this entry
	count, err := tx.Payment.Query().
		Where(payment.OrderEntryID(entry.ID)).
		Count(context.Background())
	if err != nil {
		log.Error("createPaymentForOrderEntryTx: count query failed", zap.Int("entryID", entry.ID), zap.Error(err))
		return
	}
	log.Debug("createPaymentForOrderEntryTx: payment count for entry", zap.Int("entryID", entry.ID), zap.Int("count", count))

	// If no payment exists for this entry, create one
	if count == 0 && !errorIfExists {
		p := Payment{
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
		log.Debug("createPaymentForOrderEntryTx: creating payment", zap.Int("orderID", orderID), zap.Int("entryID", entry.ID), zap.Int("sender", p.Sender), zap.Int("receiver", p.Receiver), zap.Int("amount", p.Amount))
		paymentID, err = createPaymentTx(tx, p)
	}

	return
}

// SetOrderTransactionID sets the transaction ID for an order
func (db *Database) SetOrderTransactionID(orderID int, transactionID string) (err error) {
	err = db.EntClient.Order.UpdateOneID(orderID).
		SetTransactionID(transactionID).
		Exec(context.Background())
	if err != nil {
		log.Error("SetOrderTransactionID: ", err)
	}
	return
}

// VerifyOrderAndCreatePayments sets payment order to verified and creates a payment for each order entry if it doesn't already exist
// This means if some payments have already been created with CreatePayedOrderEntries before verifying the order, they will be skipped
func (db *Database) VerifyOrderAndCreatePayments(orderID int, transactionTypeID int) (err error) {
	// Acquire per-order lock to serialize concurrent verification attempts
	// and prevent duplicate payment creation.
	unlock := lockOrder(orderID)
	defer unlock()

	log.Info("VerifyOrderAndCreatePayments: Verifying order ", orderID)
	// Start a transaction
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: Opening DBPool failed", err)
		return err
	}
	defer tx.Rollback()

	// Read current verified state before updating so we can detect a transition
	alreadyVerified, err := tx.Order.Query().
		Where(order.ID(orderID)).
		Select(order.FieldVerified).
		Bool(context.Background())

	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: read payment order verified flag", orderID, err)
		// continue — we'll still try to update and proceed
	}

	// Verify payment order
	err = tx.Order.UpdateOneID(orderID).
		SetVerified(true).
		SetVerifiedAt(time.Now().UTC()).
		SetTransactionTypeID(transactionTypeID).
		Exec(context.Background())

	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: update payment order", orderID, err)
	}

	// Get Paymentorder (including payments)
	o, err := db.GetOrderByIDTx(tx, orderID)
	if err != nil {
		log.Error("VerifyOrderAndCreatePayments: get order by id", orderID, err)
		return err
	}

	if !alreadyVerified && o.CustomerEmail.Valid && o.CustomerEmail.String != "" {

		// We may have multiple order entries for the same order. To avoid
		// assigning groups and sending the same email multiple times for the
		// same customer/order, collect unique license groups and perform
		// assignments and sends once per relevant unit.
		processedLicenseGroups := make(map[string]bool)
		var customerID string
		var newUser bool
		var dbCustomer *Customer
		gotCustomer := false
		customerAssigned := false
		sentDigitalLicenceEmail := false

		for _, entry := range o.Entries {
			item, err := db.GetItemTx(tx, entry.Item)
			if err != nil {
				log.Error("VerifyOrderAndCreatePayments: failed to get item: ", orderID, err)
			}

			if item.LicenseItem.Valid {

				if !item.IsPDFItem {
					// Ensure we only call GetOrCreateUser once per order/customer
					if !gotCustomer {
						customerID, newUser, err = keycloak.KeycloakClient.GetOrCreateUser(o.CustomerEmail.String)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to create keycloak customer: ", orderID, err)
						}
						gotCustomer = true
						dbCustomer, err = db.GetOrCreateCustomerByEmail(o.CustomerEmail.String, customerID)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to get/create customer db record: ", orderID, err)
						}
					}

					// Assign to top-level customer group once per customer
					if gotCustomer && !customerAssigned {
						err = keycloak.KeycloakClient.AssignGroup(customerID, "customer")
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to assign customer to group: ", orderID, err)
						}
						customerAssigned = true
					}

					// Assign to license subgroup once per unique license group
					lg := item.LicenseGroup.String
					if gotCustomer && lg != "" && !processedLicenseGroups[lg] {
						err = keycloak.KeycloakClient.AssignDigitalLicenseGroup(customerID, lg)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to assign customer to license group: ", orderID, err)
						}
						if dbCustomer != nil {
							_, err = db.AddLicenseGroupToCustomer(dbCustomer.ID, lg)
							if err != nil {
								log.Error("VerifyOrderAndCreatePayments: failed to add license group to customer db record: ", orderID, err)
							}
						}
						processedLicenseGroups[lg] = true

						// When an abonement is sold, create an Abonement DB record and also
						// grant access to the currently published online_issue for the same
						// license group so the customer can read it immediately.
						if item.Type == "abonement" && dbCustomer != nil {
							createdAbo, createAboErr := db.CreateAbonement(&Abonement{
								CustomerID: dbCustomer.ID,
								ItemID:     item.ID,
								FromDate:   o.Timestamp,
								ToDate:     o.Timestamp.AddDate(1, 0, 0),
								Status:     "active",
							})
							if createAboErr != nil {
								log.Error("VerifyOrderAndCreatePayments: failed to create abonement record: ", orderID, createAboErr)
							} else if dbCustomer.Email != "" {
								aboTemplateData := map[string]interface{}{
									"CustomerName": dbCustomer.FirstName + " " + dbCustomer.LastName,
									"ItemName":     item.Name,
									"FromDate":     createdAbo.FromDate.Format("2006-01-02"),
									"ToDate":       createdAbo.ToDate.Format("2006-01-02"),
									"Status":       createdAbo.Status,
								}
								aboMail, aboMailErr := BuildEmailRequestFromTemplate("abonementConfirmation", []string{dbCustomer.Email}, aboTemplateData)
								if aboMailErr != nil {
									log.Error("VerifyOrderAndCreatePayments: failed to build abonement confirmation mail: ", orderID, aboMailErr)
								} else if aboMail != nil {
									go func(m *mailer.EmailRequest) {
										success, sendErr := mailer.Send(m)
										if sendErr != nil || !success {
											log.Error("VerifyOrderAndCreatePayments: failed to send abonement confirmation mail: ", orderID, sendErr)
										}
									}(aboMail)
								}
							}
							latestIssue, found, issueErr := db.GetLatestPublishedOnlineIssueByLicenseGroup(lg)
							if issueErr != nil {
								log.Error("VerifyOrderAndCreatePayments: failed to look up latest online_issue: ", orderID, issueErr)
							} else if found && latestIssue.LicenseGroup.Valid && latestIssue.LicenseGroup.String != "" {
								issueLg := latestIssue.LicenseGroup.String
								if !processedLicenseGroups[issueLg] {
									if assignErr := keycloak.KeycloakClient.AssignDigitalLicenseGroup(customerID, issueLg); assignErr != nil {
										log.Error("VerifyOrderAndCreatePayments: failed to assign online_issue license group: ", orderID, assignErr)
									}
									if _, addErr := db.AddLicenseGroupToCustomer(dbCustomer.ID, issueLg); addErr != nil {
										log.Error("VerifyOrderAndCreatePayments: failed to add online_issue license group to customer: ", orderID, addErr)
									}
									processedLicenseGroups[issueLg] = true
								}
							}
						}
					}

					// Send email with link to the license item once per order
					if !sentDigitalLicenceEmail {
						templateData := struct {
							URL   string
							EMAIL string
						}{
							URL:   config.Config.OnlinePaperUrl,
							EMAIL: o.CustomerEmail.String,
						}

						receivers := []string{o.CustomerEmail.String}
						mail, err := db.BuildEmailRequestFromTemplate("digitalLicenceItemTemplate.html", receivers, templateData)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to create mail: ", orderID, err)
						} else if mail != nil {
							// use subject from DB template (do not override)
							go func(m *mailer.EmailRequest) {
								success, err := mailer.Send(m)
								if err != nil || !success {
									log.Error("VerifyOrderAndCreatePayments: failed to send mail: ", orderID, err)
								}
							}(mail)
						}
						sentDigitalLicenceEmail = true
					}

					// If the user was newly created, send welcome email once
					if newUser {
						newUser = false // ensure we only send welcome once
						templateData := struct {
							URL   string
							EMAIL string
						}{
							URL:   config.Config.OnlinePaperUrl,
							EMAIL: o.CustomerEmail.String,
						}
						receivers := []string{o.CustomerEmail.String}
						newUserMail, err := db.BuildEmailRequestFromTemplate("welcome", receivers, templateData)
						if err != nil {
							log.Error("VerifyOrderAndCreatePayments: failed to create welcome mail: ", orderID, err)
						} else if newUserMail != nil {
							go func(m *mailer.EmailRequest) {
								success, err := mailer.Send(m)
								if err != nil || !success {
									log.Error("VerifyOrderAndCreatePayments: failed to send welcome mail: ", orderID, err)
								}
							}(newUserMail)
						}
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
						// Likely not found in Ent context, err is not nil.
						// If specific check needed, use ent.IsNotFound(err).
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
							EMAIL: o.CustomerEmail.String,
						}
						receivers := []string{o.CustomerEmail.String}
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
	for _, entry := range o.Entries {
		_, err = createPaymentForOrderEntryTx(tx, orderID, entry, false)
		if err != nil {
			log.Error("VerifyOrderAndCreatePayments: create payments for order entry: ", orderID, err)
			return err
		}
	}

	return tx.Commit()
}

// CreatePayedOrderEntries creates entries with a payment for an order
func (db *Database) CreatePayedOrderEntries(orderID int, entries []OrderEntry) (err error) {

	// Acquire per-order lock to serialize concurrent payment creation
	// and prevent duplicate payments.
	unlock := lockOrder(orderID)
	defer unlock()

	// Start a transaction
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	return tx.Commit()
}

func convertOrderEntry(e *ent.OrderEntry) OrderEntry {
	oe := OrderEntry{
		ID:       e.ID,
		Item:     e.ItemID,
		Quantity: e.Quantity,
		Price:    e.Price,
		Sender:   e.SenderID,
		Receiver: e.ReceiverID,
		IsSale:   e.IsSale,
	}
	if e.Edges.Sender != nil {
		oe.SenderName = e.Edges.Sender.Name
	}
	if e.Edges.Receiver != nil {
		oe.ReceiverName = e.Edges.Receiver.Name
	}
	return oe
}

func convertOrder(e *ent.Order) Order {
	o := Order{
		ID:                e.ID,
		TransactionID:     e.TransactionID,
		Verified:          e.Verified,
		TransactionTypeID: e.TransactionTypeID,
		Timestamp:         e.Timestamp,
		Vendor:            e.VendorID,
	}
	if e.OrderCode != nil {
		o.OrderCode = null.StringFrom(*e.OrderCode)
	}
	if e.UserID != nil {
		o.User = null.StringFrom(*e.UserID)
	}
	if e.CustomerEmail != nil {
		o.CustomerEmail = null.StringFrom(*e.CustomerEmail)
	}
	if e.VerifiedAt != nil {
		tt := *e.VerifiedAt
		o.VerifiedAt = null.TimeFrom(tt)
	}
	for _, entry := range e.Edges.Entries {
		o.Entries = append(o.Entries, convertOrderEntry(entry))
	}
	return o
}

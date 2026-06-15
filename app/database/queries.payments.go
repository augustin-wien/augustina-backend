package database

import (
	"context"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entaccount "github.com/augustin-wien/augustina-backend/ent/account"
	entorder "github.com/augustin-wien/augustina-backend/ent/order"
	entpayment "github.com/augustin-wien/augustina-backend/ent/payment"
	"gopkg.in/guregu/null.v4"
)

// Payments -------------------------------------------------------------------

// ListPayments returns the payments from the database
func (db *Database) ListPayments(minDate time.Time, maxDate time.Time, vendorLicenseID string, filterPayouts bool, filterSales bool, filterNoPayout bool, excludeBackoffice bool, excludePOS bool) (payments []Payment, err error) {
	ctx := context.Background()

	// Start query
	q := db.EntClient.Payment.Query()

	// Apply filters
	if !minDate.IsZero() {
		q.Where(entpayment.TimestampGTE(minDate))
	}
	if !maxDate.IsZero() {
		q.Where(entpayment.TimestampLTE(maxDate))
	}

	if vendorLicenseID != "" {
		vendor, err := db.GetVendorByLicenseID(vendorLicenseID)
		if err != nil {
			return nil, err
		}
		vendorAccount, err := db.GetAccountByVendorID(vendor.ID)
		if err != nil {
			return nil, err
		}

		q.Where(entpayment.Or(
			entpayment.SenderID(vendorAccount.ID),
			entpayment.ReceiverID(vendorAccount.ID),
		))
	}
	if filterPayouts {
		cashAccountID, err := db.GetAccountTypeID("Cash")
		if err != nil {
			return nil, err
		}
		q.Where(entpayment.ReceiverID(cashAccountID))
	}
	if filterNoPayout {
		// Does not have a payout (Payout IS NULL) and is not a payout (Receiver != Cash)
		cashAccountID, err := db.GetAccountTypeID("Cash")
		if err != nil {
			return nil, err
		}
		q.Where(
			entpayment.PayoutIDIsNil(),
			entpayment.ReceiverIDNEQ(cashAccountID),
		)
	}
	if filterSales {
		q.Where(entpayment.IsSale(true))
	}
	if excludeBackoffice {
		backofficeID, err := db.GetAccountTypeID("Backoffice")
		if err != nil {
			return nil, err
		}
		q.Where(
			entpayment.SenderIDNEQ(backofficeID),
			entpayment.ReceiverIDNEQ(backofficeID),
		)
	}
	if excludePOS {
		q.Where(entpayment.IsPos(false))
	}

	// Order by timestamp
	q.Order(ent.Asc(entpayment.FieldTimestamp))

	// Fetch children for payouts
	q.WithChildren(func(cq *ent.PaymentQuery) {
		cq.Order(ent.Asc(entpayment.FieldTimestamp))
	})

	// Execute query
	ents, err := q.All(ctx)
	if err != nil {
		log.Error("ListPayments: ", err)
		return nil, err
	}

	// Optimization: Batch fetch accounts for names
	accountIDs := make(map[int]struct{})
	for _, p := range ents {
		accountIDs[p.SenderID] = struct{}{}
		accountIDs[p.ReceiverID] = struct{}{}
		for _, child := range p.Edges.Children {
			accountIDs[child.SenderID] = struct{}{}
			accountIDs[child.ReceiverID] = struct{}{}
		}
	}

	var ids []int
	for id := range accountIDs {
		ids = append(ids, id)
	}

	if len(ids) > 0 {
		accounts, err := db.EntClient.Account.Query().Where(entaccount.IDIn(ids...)).All(ctx)
		if err != nil {
			log.Error("ListPayments: fetch accounts ", err)
			return nil, err
		}

		accountMap := make(map[int]*ent.Account)
		for _, acc := range accounts {
			accountMap[acc.ID] = acc
		}

		// Convert to Payment structs
		for _, p := range ents {
			pmt := db.PaymentEntIntoPayment(p)

			// Fill names
			if acc, ok := accountMap[p.SenderID]; ok {
				pmt.SenderName = null.StringFrom(acc.Name)
			}
			if acc, ok := accountMap[p.ReceiverID]; ok {
				pmt.ReceiverName = null.StringFrom(acc.Name)
			}

			// Handle children (payouts)
			for _, child := range p.Edges.Children {
				childPmt := db.PaymentEntIntoPayment(child)
				pmt.IsPayoutFor = append(pmt.IsPayoutFor, childPmt)
			}

			payments = append(payments, pmt)
		}
	}

	return payments, nil
}

// ListPaymentsForCustomer returns payments that belong to orders created by the given customer email.
func (db *Database) ListPaymentsForCustomer(customerEmail string) (payments []Payment, err error) {
	ctx := context.Background()

	if customerEmail == "" {
		return []Payment{}, nil
	}

	orderIDs, err := db.EntClient.Order.Query().
		Where(entorder.CustomerEmail(customerEmail)).
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(orderIDs) == 0 {
		return []Payment{}, nil
	}

	q := db.EntClient.Payment.Query().Where(entpayment.OrderIDIn(orderIDs...))
	q.Order(ent.Asc(entpayment.FieldTimestamp))
	q.WithChildren(func(cq *ent.PaymentQuery) {
		cq.Order(ent.Asc(entpayment.FieldTimestamp))
	})

	ents, err := q.All(ctx)
	if err != nil {
		log.Error("ListPaymentsForCustomer: ", err)
		return nil, err
	}

	accountIDs := make(map[int]struct{})
	for _, p := range ents {
		accountIDs[p.SenderID] = struct{}{}
		accountIDs[p.ReceiverID] = struct{}{}
		for _, child := range p.Edges.Children {
			accountIDs[child.SenderID] = struct{}{}
			accountIDs[child.ReceiverID] = struct{}{}
		}
	}

	var ids []int
	for id := range accountIDs {
		ids = append(ids, id)
	}

	if len(ids) > 0 {
		accounts, err := db.EntClient.Account.Query().Where(entaccount.IDIn(ids...)).All(ctx)
		if err != nil {
			log.Error("ListPaymentsForCustomer: fetch accounts ", err)
			return nil, err
		}

		accountMap := make(map[int]*ent.Account)
		for _, acc := range accounts {
			accountMap[acc.ID] = acc
		}

		for _, p := range ents {
			pmt := db.PaymentEntIntoPayment(p)
			if acc, ok := accountMap[p.SenderID]; ok {
				pmt.SenderName = null.StringFrom(acc.Name)
			}
			if acc, ok := accountMap[p.ReceiverID]; ok {
				pmt.ReceiverName = null.StringFrom(acc.Name)
			}
			for _, child := range p.Edges.Children {
				childPmt := db.PaymentEntIntoPayment(child)
				pmt.IsPayoutFor = append(pmt.IsPayoutFor, childPmt)
			}
			payments = append(payments, pmt)
		}
	}

	return payments, nil
}

// ListPaymentsForPayout returns sales payments that have not been paid out yet
func (db *Database) ListPaymentsForPayout(minDate time.Time, maxDate time.Time, vendorLicenseID string) (payments []Payment, err error) {
	return db.ListPayments(minDate, maxDate, vendorLicenseID, false, false, true, false, true)
}

// GetPayment returns the payment with the given ID
func (db *Database) GetPayment(id int) (payment Payment, err error) {
	p, err := db.EntClient.Payment.Query().
		Where(entpayment.ID(id)).
		First(context.Background())

	if err != nil {
		log.Error("GetPayment: ", err)
		return payment, err
	}

	payment = db.PaymentEntIntoPayment(p)

	// Fetch Account Names
	accs, err := db.EntClient.Account.Query().
		Where(entaccount.IDIn(p.SenderID, p.ReceiverID)).
		All(context.Background())

	if err == nil {
		for _, acc := range accs {
			if acc.ID == p.SenderID {
				payment.SenderName = null.StringFrom(acc.Name)
			}
			if acc.ID == p.ReceiverID {
				payment.ReceiverName = null.StringFrom(acc.Name)
			}
		}
	} else {
		log.Error("GetPayment: accounts fetch ", err)
	}

	return payment, nil
}

// PaymentEntIntoPayment converts ent.Payment to database.Payment
func (db *Database) PaymentEntIntoPayment(p *ent.Payment) Payment {
	pmt := Payment{
		ID:           p.ID,
		Timestamp:    p.Timestamp,
		Sender:       p.SenderID,
		Receiver:     p.ReceiverID,
		Amount:       p.Amount,
		AuthorizedBy: p.AuthorizedBy,
		IsSale:       p.IsSale,
		IsPOS:        p.IsPos,
		Quantity:     p.Quantity,
		Price:        p.Price,
	}

	if p.OrderID != nil {
		pmt.Order = null.IntFrom(int64(*p.OrderID))
	}
	if p.OrderEntryID != nil {
		pmt.OrderEntry = null.IntFrom(int64(*p.OrderEntryID))
	}
	if p.ItemID != nil {
		pmt.Item = null.IntFrom(int64(*p.ItemID))
	}
	if p.PayoutID != nil {
		pmt.Payout = null.IntFrom(int64(*p.PayoutID))
	}

	return pmt
}

// CreatePayment creates a payment in an transaction
func createPaymentTx(tx *ent.Tx, payment Payment) (paymentID int, err error) {

	// Insert payment
	create := tx.Payment.Create().
		SetSenderID(payment.Sender).
		SetReceiverID(payment.Receiver).
		SetAmount(payment.Amount).
		SetAuthorizedBy(payment.AuthorizedBy).
		SetIsSale(payment.IsSale).
		SetIsPos(payment.IsPOS).
		SetQuantity(payment.Quantity).
		SetPrice(payment.Price).
		SetTimestamp(time.Now())

	if payment.Order.Valid {
		create.SetOrderID(int(payment.Order.Int64))
	}
	if payment.OrderEntry.Valid {
		create.SetOrderEntryID(int(payment.OrderEntry.Int64))
	}
	if payment.Item.Valid {
		create.SetItemID(int(payment.Item.Int64))
	}
	if payment.Payout.Valid {
		create.SetPayoutID(int(payment.Payout.Int64))
	}

	pRes, err := create.Save(context.Background())
	if err != nil {
		log.Error("createPaymentTx: insert ", err)
		return 0, err
	}
	paymentID = pRes.ID

	// POS sale records are bookkeeping-only; they must not touch account balances
	// because the actual money flow is captured by the balance-chain and cash payments.
	if !(payment.IsPOS && payment.IsSale) {
		err = updateAccountBalanceTx(tx, payment.Sender, -payment.Amount)
		if err != nil {
			log.Error("createPaymentTx: update sender ", err)
		}
		err = updateAccountBalanceTx(tx, payment.Receiver, payment.Amount)
		if err != nil {
			log.Error("createPaymentTx: update receiver", err)
		}
	}
	return
}

// CreatePayment creates a payment and returns the payment ID
func (db *Database) CreatePayment(payment Payment) (paymentID int, err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("CreatePayment: rollback failed: ", rbErr)
			}
		}
	}()

	paymentID, err = createPaymentTx(tx, payment)
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	return
}

// CreatePayments creates multiple payments
func (db *Database) CreatePayments(payments []Payment) (err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		log.Error("CreatePayments: ", err)
		return err
	}
	defer tx.Rollback()

	// Insert payments within the transaction
	for _, payment := range payments {
		_, err = createPaymentTx(tx, payment)
		if err != nil {
			log.Error("CreatePayments: ", err)
			return err
		}
	}
	return tx.Commit()
}

// CreatePaymentPayout creates a payout for a range of payments
func (db *Database) CreatePaymentPayout(vendor Vendor, vendorAccountID int, authorizedBy string, amount int, payments []Payment) (paymentID int, err error) {

	// Create a transaction to insert all payments at once
	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}
	defer tx.Rollback()

	// Get cash account
	cashAccount, err := db.GetAccountByType("Cash")
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}

	p := Payment{
		Sender:       vendorAccountID,
		Receiver:     cashAccount.ID,
		Amount:       amount,
		AuthorizedBy: authorizedBy,
		Timestamp:    time.Now(),
	}

	// Insert payments within the transaction
	paymentID, err = createPaymentTx(tx, p)
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}

	// Document that these payments have a payout
	for _, pay := range payments {
		err = tx.Payment.UpdateOneID(pay.ID).
			SetPayoutID(paymentID).
			Exec(context.Background())
		if err != nil {
			log.Error("CreatePaymentPayout: ", err)
			return 0, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	// Update last payout date - outside transaction as UpdateVendor likely uses separate connection
	// Ideally should be inside, but for now strict replacement.
	vendor.LastPayout = null.NewTime(time.Now(), true)
	err = db.UpdateVendor(vendor.ID, vendor)
	if err != nil {
		log.Error("CreatePaymentPayout: ", err)
		return
	}

	return
}

// POSOrderItem represents one line item in a POS order.
type POSOrderItem struct {
	ItemID   int    `json:"itemId"`
	ItemName string `json:"itemName"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
	Amount   int    `json:"amount"`
}

// POSOrder groups vendor-linked POS payments that share the same second-level timestamp.
// BalanceUsed is the amount drawn from the vendor's credit balance (Vendor→Orga payment).
// CashAmount is the remainder paid in cash.
// TotalAmount is the sum of all item amounts (zero if only balance-chain payments exist).
type POSOrder struct {
	Timestamp       time.Time      `json:"timestamp"`
	Items           []POSOrderItem `json:"items"`
	TotalAmount     int            `json:"totalAmount"`
	BalanceUsed     int            `json:"balanceUsed"`
	CashAmount      int            `json:"cashAmount"`
	AuthorizedBy    string         `json:"authorizedBy"`
	VendorName      string         `json:"vendorName"`
	VendorLicenseID string         `json:"vendorLicenseId"`
}

// ListPOSOrdersForVendor returns the most recent POS orders for a vendor.
//
// It queries all is_pos=true payments where the vendor account is the sender,
// which covers both:
//   - Vendor→Orga balance-chain payments (is_sale=false)
//   - Per-item bookkeeping records (is_sale=true)
//
// Payments sharing the same truncated-to-second timestamp are grouped into one order.
func (db *Database) ListPOSOrdersForVendor(licenseID string, limit int) ([]POSOrder, error) {
	ctx := context.Background()

	vendor, err := db.GetVendorByLicenseID(licenseID)
	if err != nil {
		return nil, err
	}
	vendorAccount, err := db.GetAccountByVendorID(vendor.ID)
	if err != nil {
		return nil, err
	}

	// Fetch all is_pos payments where vendor is sender (both balance-chain and item records).
	// Over-fetch to handle multi-item orders sharing a timestamp group.
	ents, err := db.EntClient.Payment.Query().
		Where(
			entpayment.SenderID(vendorAccount.ID),
			entpayment.IsPos(true),
		).
		Order(ent.Desc(entpayment.FieldTimestamp)).
		Limit(limit * 20).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Resolve item names in one pass.
	itemNames := make(map[int]string)
	for _, p := range ents {
		if p.ItemID != nil {
			if _, seen := itemNames[*p.ItemID]; !seen {
				if item, e := db.GetItem(*p.ItemID); e == nil {
					itemNames[*p.ItemID] = item.Name
				}
			}
		}
	}

	// Group payments by second-precision timestamp.
	orderMap := make(map[int64]*POSOrder)
	var orderKeys []int64

	for _, p := range ents {
		ts := p.Timestamp.Truncate(time.Second).Unix()
		if _, exists := orderMap[ts]; !exists {
			if len(orderKeys) >= limit {
				break
			}
			orderMap[ts] = &POSOrder{
				Timestamp:    p.Timestamp.Truncate(time.Second),
				AuthorizedBy: p.AuthorizedBy,
			}
			orderKeys = append(orderKeys, ts)
		}
		ord := orderMap[ts]

		if p.IsSale {
			// Per-item bookkeeping record.
			li := POSOrderItem{
				Quantity: p.Quantity,
				Price:    p.Price,
				Amount:   p.Amount,
			}
			if p.ItemID != nil {
				li.ItemID = *p.ItemID
				li.ItemName = itemNames[*p.ItemID]
			}
			ord.Items = append(ord.Items, li)
			ord.TotalAmount += p.Amount
		} else {
			// Balance-chain payment (Vendor→Orga).
			ord.BalanceUsed += p.Amount
		}
	}

	// Compute cash portion for each order.
	for _, k := range orderKeys {
		ord := orderMap[k]
		if ord.TotalAmount > 0 {
			ord.CashAmount = ord.TotalAmount - ord.BalanceUsed
		}
	}

	orders := make([]POSOrder, 0, len(orderKeys))
	for _, k := range orderKeys {
		orders = append(orders, *orderMap[k])
	}
	return orders, nil
}

// ListAllPOSOrders returns recent POS orders across all vendors within the given date range.
// Each order is identified by (vendorAccountID, second-truncated timestamp).
func (db *Database) ListAllPOSOrders(minDate, maxDate time.Time) ([]POSOrder, error) {
	ctx := context.Background()

	q := db.EntClient.Payment.Query().
		Where(entpayment.IsPos(true)).
		Order(ent.Desc(entpayment.FieldTimestamp))

	if !minDate.IsZero() {
		q = q.Where(entpayment.TimestampGTE(minDate))
	}
	if !maxDate.IsZero() {
		q = q.Where(entpayment.TimestampLTE(maxDate))
	}

	ents, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	// Identify which account IDs are vendor accounts to exclude Orga/Cash/Backoffice senders.
	vendorAccounts, err := db.EntClient.Account.Query().
		Where(entaccount.Type("Vendor")).
		All(ctx)
	if err != nil {
		return nil, err
	}
	vendorAccountSet := make(map[int]bool, len(vendorAccounts))
	for _, a := range vendorAccounts {
		vendorAccountSet[a.ID] = true
	}

	// Build vendor lookup: accountID → vendor
	type vendorInfo struct{ name, licenseID string }
	vendorByAccount := make(map[int]vendorInfo)
	allVendors, err := db.ListVendors()
	if err != nil {
		return nil, err
	}
	for _, v := range allVendors {
		acc, e := db.GetAccountByVendorID(v.ID)
		if e == nil {
			vendorByAccount[acc.ID] = vendorInfo{
				name:      v.FirstName + " " + v.LastName,
				licenseID: v.LicenseID.String,
			}
		}
	}

	// Resolve item names.
	itemNames := make(map[int]string)
	for _, p := range ents {
		if p.ItemID != nil {
			if _, seen := itemNames[*p.ItemID]; !seen {
				if item, e := db.GetItem(*p.ItemID); e == nil {
					itemNames[*p.ItemID] = item.Name
				}
			}
		}
	}

	// Group by (vendorAccountID, second-truncated timestamp).
	type orderKey struct {
		accountID int
		ts        int64
	}
	orderMap := make(map[orderKey]*POSOrder)
	var orderKeys []orderKey

	for _, p := range ents {
		// Only consider payments where the sender is a vendor account.
		if !vendorAccountSet[p.SenderID] {
			continue
		}
		k := orderKey{
			accountID: p.SenderID,
			ts:        p.Timestamp.Truncate(time.Second).Unix(),
		}
		if _, exists := orderMap[k]; !exists {
			vi := vendorByAccount[p.SenderID]
			orderMap[k] = &POSOrder{
				Timestamp:       p.Timestamp.Truncate(time.Second),
				AuthorizedBy:    p.AuthorizedBy,
				VendorName:      vi.name,
				VendorLicenseID: vi.licenseID,
			}
			orderKeys = append(orderKeys, k)
		}
		ord := orderMap[k]
		if p.IsSale {
			li := POSOrderItem{Quantity: p.Quantity, Price: p.Price, Amount: p.Amount}
			if p.ItemID != nil {
				li.ItemID = *p.ItemID
				li.ItemName = itemNames[*p.ItemID]
			}
			ord.Items = append(ord.Items, li)
			ord.TotalAmount += p.Amount
		} else {
			ord.BalanceUsed += p.Amount
		}
	}

	for _, k := range orderKeys {
		ord := orderMap[k]
		if ord.TotalAmount > 0 {
			ord.CashAmount = ord.TotalAmount - ord.BalanceUsed
		}
	}

	orders := make([]POSOrder, 0, len(orderKeys))
	for _, k := range orderKeys {
		orders = append(orders, *orderMap[k])
	}
	return orders, nil
}

// DeletePayment deletes a payment (should not be used in production)
func (db *Database) DeletePayment(paymentID int) (err error) {
	err = db.EntClient.Payment.DeleteOneID(paymentID).Exec(context.Background())
	if err != nil {
		log.Error("DeletePayment: ", err)
	}
	return
}

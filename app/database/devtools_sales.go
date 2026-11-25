package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	entorder "github.com/augustin-wien/augustina-backend/ent/order"
	entpayment "github.com/augustin-wien/augustina-backend/ent/payment"
	"gopkg.in/guregu/null.v4"
)

// saleEntries returns the order entries of a QR code sale, including the
// license the vendor pays to the organisation for licensed items
func (db *Database) devSaleEntries(items devItems, buyer, vendorAccount, orga int, digital, calendars, newspapers, donation int) []OrderEntry {
	var entries []OrderEntry
	add := func(item, quantity, sender, receiver int, isSale bool) {
		if item == 0 || quantity == 0 {
			return
		}
		entries = append(entries, OrderEntry{
			Item:     item,
			Quantity: quantity,
			Sender:   sender,
			Receiver: receiver,
			IsSale:   isSale,
		})
	}
	add(items.digitalNewspaper, digital, buyer, vendorAccount, true)
	add(items.digitalLicense, digital, vendorAccount, orga, false)
	add(items.calendar, calendars, buyer, vendorAccount, true)
	add(items.newspaper, newspapers, buyer, vendorAccount, true)
	add(items.donation, donation, buyer, vendorAccount, false)
	return entries
}

// devSale describes one QR code sale of the seed
type devSale struct {
	vendor     int
	code       string
	daysAgo    int
	digital    int
	calendars  int
	newspapers int
	donation   int
	paypal     bool
}

// createDevSale creates, verifies and backdates a QR code sale, including the
// transaction costs the organisation covers
func (db *Database) createDevSale(items devItems, s devSale) (orderID int, err error) {
	buyer, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}
	provider, err := db.GetAccountTypeID("VivaWallet")
	if err != nil {
		return
	}
	transactionType := 0
	if s.paypal {
		if provider, err = db.GetAccountTypeID("Paypal"); err != nil {
			return
		}
		transactionType = config.Config.VivaWalletTransactionTypeIDPaypal
	}
	vendorAccount, err := db.GetAccountByVendorID(s.vendor)
	if err != nil {
		return
	}

	order := Order{
		OrderCode:     null.StringFrom(s.code),
		TransactionID: "dev-" + s.code,
		Vendor:        s.vendor,
		Entries:       db.devSaleEntries(items, buyer, vendorAccount.ID, orga, s.digital, s.calendars, s.newspapers, s.donation),
	}
	if orderID, err = db.CreateOrder(order); err != nil {
		return
	}
	if err = db.VerifyOrderAndCreatePayments(orderID, transactionType); err != nil {
		return
	}
	if items.transactionCosts != 0 {
		err = db.CreatePayedOrderEntries(orderID, []OrderEntry{
			// Vendor pays the payment provider's transaction costs...
			{Item: items.transactionCosts, Quantity: 27, Sender: vendorAccount.ID, Receiver: provider},
			// ...and gets them back from the organisation
			{Item: items.transactionCosts, Quantity: 27, Sender: orga, Receiver: vendorAccount.ID},
		})
		if err != nil {
			return
		}
	}
	err = db.backdateDevOrder(orderID, time.Now().AddDate(0, 0, -s.daysAgo))
	return
}

// backdateDevOrder moves an order and its payments into the past, so the
// statistics have more than a single day to show
func (db *Database) backdateDevOrder(orderID int, at time.Time) error {
	ctx := context.Background()
	at = at.UTC()
	err := db.EntClient.Order.UpdateOneID(orderID).SetTimestamp(at).Exec(ctx)
	if err != nil {
		return err
	}
	err = db.EntClient.Order.Update().
		Where(entorder.ID(orderID), entorder.Verified(true)).
		SetVerifiedAt(at.Add(2 * time.Minute)).
		Exec(ctx)
	if err != nil {
		return err
	}
	return db.EntClient.Payment.Update().
		Where(entpayment.OrderID(orderID)).
		SetTimestamp(at.Add(2 * time.Minute)).
		Exec(ctx)
}

// createDevOrdersAndPayments creates the sales history before the payout: verified
// orders over the last months plus the order states that never lead to payments
func (db *Database) createDevOrdersAndPayments(vendors devVendors, items devItems) (err error) {
	sales := []devSale{
		// The original dev order: 2 digital newspapers, 1 calendar and a donation
		{vendor: vendors.active, code: "devOrder1", daysAgo: 150, digital: 2, calendars: 1, donation: 50},
		{vendor: vendors.active, code: "devOrder2", daysAgo: 120, newspapers: 3},
		{vendor: vendors.active, code: "devOrder3", daysAgo: 95, digital: 1, donation: 100, paypal: true},
		{vendor: vendors.active, code: "devOrder4", daysAgo: 60, calendars: 2},
		{vendor: vendors.active, code: "devOrder5", daysAgo: 35, newspapers: 1, digital: 1, paypal: true},
		{vendor: vendors.active, code: "devOrder6", daysAgo: 14, newspapers: 2, donation: 20},
		{vendor: vendors.openBalance, code: "devOrder7", daysAgo: 80, newspapers: 2, calendars: 1},
		{vendor: vendors.openBalance, code: "devOrder8", daysAgo: 25, digital: 3, paypal: true},
		{vendor: vendors.disabled, code: "devOrder9", daysAgo: 170, newspapers: 4},
		{vendor: vendors.blocked, code: "devOrder10", daysAgo: 45, calendars: 1, donation: 30},
		{vendor: vendors.deleted, code: "devOrder11", daysAgo: 160, newspapers: 1},
	}
	var synced, failed int
	for _, s := range sales {
		var orderID int
		if orderID, err = db.createDevSale(items, s); err != nil {
			return
		}
		switch s.code {
		case "devOrder4":
			synced = orderID
		case "devOrder6":
			failed = orderID
		}
	}

	// Odoo export states
	if err = db.SetOdooSyncSuccess(synced); err != nil {
		return
	}
	if err = db.SetOdooSyncFailure(failed, errors.New("odoo: 502 Bad Gateway")); err != nil {
		return
	}

	buyer, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}
	vendorAccount, err := db.GetAccountByVendorID(vendors.active)
	if err != nil {
		return
	}

	// Abandoned checkout, given up on by the order reconciliation
	abandonedID, err := db.CreateOrder(Order{
		OrderCode: null.StringFrom("devOrderAbandoned"),
		Vendor:    vendors.active,
		Entries:   db.devSaleEntries(items, buyer, vendorAccount.ID, orga, 0, 1, 0, 0),
	})
	if err != nil {
		return
	}
	if err = db.backdateDevOrder(abandonedID, time.Now().AddDate(0, 0, -2)); err != nil {
		return
	}
	if err = db.InvalidateOrder(abandonedID); err != nil {
		return
	}

	// Checkout still in progress; the reconciliation invalidates it after a while
	_, err = db.CreateOrder(Order{
		OrderCode: null.StringFrom("devOrderPending"),
		Vendor:    vendors.openBalance,
		Entries:   db.devSaleEntries(items, buyer, vendorAccount.ID, orga, 0, 0, 2, 10),
	})
	return
}

// createDevRecentSales creates sales after the payout, so vendors have open balances
func (db *Database) createDevRecentSales(vendors devVendors, items devItems) (err error) {
	sales := []devSale{
		{vendor: vendors.active, code: "devOrderRecent1", daysAgo: 3, newspapers: 2, calendars: 1},
		{vendor: vendors.active, code: "devOrderRecent2", daysAgo: 1, digital: 1, donation: 50, paypal: true},
	}
	for _, s := range sales {
		if _, err = db.createDevSale(items, s); err != nil {
			return
		}
	}
	// Online PDF purchase: the customer gets the download link by email
	if _, err = db.createDevPDFSale(vendors.openBalance, items); err != nil {
		return
	}

	// A vendor who bought licenses without selling anything ends up with a
	// negative balance
	vendorAccount, err := db.GetAccountByVendorID(vendors.negative)
	if err != nil {
		return
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return
	}
	_, err = db.CreatePayment(Payment{
		Sender:       vendorAccount.ID,
		Receiver:     orga,
		Amount:       150,
		AuthorizedBy: "devtools",
		Item:         null.IntFrom(int64(items.digitalLicense)),
		Quantity:     3,
		Price:        50,
	})
	return
}

// createDevPDFSale sells the PDF newspaper online with a customer email, which
// creates the PDF download link
func (db *Database) createDevPDFSale(vendorID int, items devItems) (orderID int, err error) {
	buyer, err := db.GetAccountTypeID("UserAnon")
	if err != nil {
		return
	}
	vendorAccount, err := db.GetAccountByVendorID(vendorID)
	if err != nil {
		return
	}
	orderID, err = db.CreateOrder(Order{
		OrderCode:     null.StringFrom("devOrderPDF"),
		TransactionID: "dev-devOrderPDF",
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom("pdf.kaeuferin@example.com"),
		Entries: []OrderEntry{{
			Item:     items.pdfNewspaper,
			Quantity: 1,
			Sender:   buyer,
			Receiver: vendorAccount.ID,
			IsSale:   true,
		}},
	})
	if err != nil {
		return
	}
	err = db.VerifyOrderAndCreatePayments(orderID, 0)
	return
}

// createDevPOSSales creates sales at the backoffice point of sale, paid in cash
// and with the vendor's balance
func (db *Database) createDevPOSSales(vendors devVendors, items devItems) error {
	cash, err := db.GetAccountTypeID("Cash")
	if err != nil {
		return err
	}
	backoffice, err := db.GetAccountTypeID("Backoffice")
	if err != nil {
		return err
	}
	orga, err := db.GetAccountTypeID("Orga")
	if err != nil {
		return err
	}

	posSale := func(vendorID int, balancePortion int, entries map[int]int) error {
		vendorAccount, err := db.GetAccountByVendorID(vendorID)
		if err != nil {
			return err
		}
		total := 0
		var payments []Payment
		for itemID, quantity := range entries {
			item, err := db.GetItem(itemID)
			if err != nil {
				return err
			}
			total += item.Price * quantity
			// Bookkeeping record, doesn't move money (see handlers.pos.go)
			payments = append(payments, Payment{
				Sender:       vendorAccount.ID,
				Receiver:     backoffice,
				Amount:       item.Price * quantity,
				AuthorizedBy: "devtools",
				IsSale:       true,
				IsPOS:        true,
				Item:         null.IntFrom(int64(itemID)),
				Quantity:     quantity,
				Price:        item.Price,
			})
		}
		if balancePortion > total {
			balancePortion = total
		}
		if balancePortion > 0 {
			payments = append(payments,
				Payment{Sender: vendorAccount.ID, Receiver: orga, Amount: balancePortion, AuthorizedBy: "devtools", IsPOS: true, Quantity: 1, Price: balancePortion},
				Payment{Sender: orga, Receiver: backoffice, Amount: balancePortion, AuthorizedBy: "devtools", IsPOS: true, Quantity: 1, Price: balancePortion},
			)
		}
		if cashPortion := total - balancePortion; cashPortion > 0 {
			payments = append(payments, Payment{Sender: cash, Receiver: backoffice, Amount: cashPortion, AuthorizedBy: "devtools", IsPOS: true, Quantity: 1, Price: cashPortion})
		}
		return db.CreatePayments(payments)
	}

	// Paid in cash
	if err = posSale(vendors.active, 0, map[int]int{items.newspaper: 10}); err != nil {
		return err
	}
	// Paid with part of the vendor's balance, the rest in cash
	return posSale(vendors.openBalance, 500, map[int]int{items.newspaper: 5, items.bag: 1})
}

// createDevPayout pays out everything the vendor has earned so far
func (db *Database) createDevPayout(vendorID int) error {
	vendor, err := db.GetVendor(vendorID)
	if err != nil {
		return err
	}

	vendorAccount, err := db.GetAccountByVendorID(vendorID)
	if err != nil {
		return err
	}

	payments, err := db.ListPaymentsForPayout(time.Now().AddDate(-1, 0, 0), time.Now().AddDate(0, 0, 1), vendor.LicenseID.String)
	if err != nil {
		return err
	}

	if len(payments) == 0 {
		return nil
	}

	total := 0
	for _, p := range payments {
		total += p.Amount
	}

	_, err = db.CreatePaymentPayout(vendor, vendorAccount.ID, "devtools", total, payments)
	if err != nil {
		return err
	}
	return db.EntClient.Vendor.UpdateOneID(vendorID).
		SetLastpayout(time.Now().AddDate(0, 0, -7)).
		Exec(context.Background())
}

package database

import (
	"go.uber.org/zap"
)

// The dev seed tries to cover every state the backoffice and the shop can show,
// so each screen has something to render without clicking data together first:
//
//   - vendors: active with full profile, open balance, negative balance and debt,
//     disabled, blocked, deleted and brand new without any activity
//   - locations: every working time mode, with and without vendor or telephone
//   - comments: warnings and notes, resolved and unresolved
//   - items: every type, disabled, colored and ordered, with PDF, archived
//   - orders: verified (card and PayPal) spread over the last months, pending,
//     invalidated, with customer email, synced to Odoo and failed Odoo sync
//   - payments: online sales, POS sales paid cash and by balance, payouts
//   - customers and abonements: active, expired, future, inactive, cancelled
//   - campaigns: running, scheduled, expired, disabled and without time frame
//   - PDF downloads: unused and downloaded links
//   - blocked IPs: active block, expired block and strikes only

// devVendors holds the IDs of the seeded vendors by role
type devVendors struct {
	active      int // fl-123: full profile, sales, payout, locations, comments
	openBalance int // fl-234: sales that were never paid out
	negative    int // fl-345: bought licenses without selling, owes debt
	disabled    int // fl-456
	blocked     int // fl-567
	deleted     int // fl-678
	fresh       int // fl-789: just registered, no activity at all
}

// devItems holds the IDs of the items the seed sells
type devItems struct {
	newspaper        int
	donation         int
	transactionCosts int
	digitalLicense   int
	digitalNewspaper int
	calendar         int
	pdfNewspaper     int
	bag              int
}

// CreateDevData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	vendors, err := db.createDevVendors()
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevLocations(vendors)
	if err != nil {
		log.Error("Dev data location creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevComments(vendors)
	if err != nil {
		log.Error("Dev data comment creation failed ", zap.Error(err))
		return err
	}
	items, err := db.createDevItems()
	if err != nil {
		log.Error("Dev data item creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevOrdersAndPayments(vendors, items)
	if err != nil {
		log.Error("Dev data order creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevArchivedItems([]int{vendors.active})
	if err != nil {
		log.Error("Dev data archived item creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPayout(vendors.active)
	if err != nil {
		log.Error("Dev data payout creation failed ", zap.Error(err))
		return err
	}
	// Created after the payout so the active vendor also has an open balance
	err = db.createDevRecentSales(vendors, items)
	if err != nil {
		log.Error("Dev data recent sales creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPOSSales(vendors, items)
	if err != nil {
		log.Error("Dev data POS sales creation failed ", zap.Error(err))
		return err
	}
	// Blocked and deleted vendors keep their history, so their state is set
	// only after their sales exist
	err = db.finishDevVendorStates(vendors)
	if err != nil {
		log.Error("Dev data vendor state update failed ", zap.Error(err))
		return err
	}
	err = db.createDevCustomersAndAbonements()
	if err != nil {
		log.Error("Dev data customers/abonements creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevCampaigns(items)
	if err != nil {
		log.Error("Dev data campaign creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevPDFDownloads(items)
	if err != nil {
		log.Error("Dev data PDF download creation failed ", zap.Error(err))
		return err
	}
	err = db.createDevBlockedIPs()
	if err != nil {
		log.Error("Dev data blocked IP creation failed ", zap.Error(err))
		return err
	}
	return nil
}

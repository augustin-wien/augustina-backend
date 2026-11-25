package database

import (
	"context"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entaccount "github.com/augustin-wien/augustina-backend/ent/account"
	entitem "github.com/augustin-wien/augustina-backend/ent/item"
	entpayment "github.com/augustin-wien/augustina-backend/ent/payment"
)

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

	itemNames, err := db.paymentItemNames(ctx, ents)
	if err != nil {
		return nil, err
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
	vendorByID := make(map[int]Vendor, len(allVendors))
	for _, v := range allVendors {
		vendorByID[v.ID] = v
	}
	for _, a := range vendorAccounts {
		if v, ok := vendorByID[a.VendorID]; ok {
			vendorByAccount[a.ID] = vendorInfo{
				name:      v.FirstName + " " + v.LastName,
				licenseID: v.LicenseID.String,
			}
		}
	}

	itemNames, err := db.paymentItemNames(ctx, ents)
	if err != nil {
		return nil, err
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

// paymentItemNames returns the names of the items the payments refer to, keyed by item ID
func (db *Database) paymentItemNames(ctx context.Context, payments []*ent.Payment) (map[int]string, error) {
	var ids []int
	for _, p := range payments {
		if p.ItemID != nil {
			ids = append(ids, *p.ItemID)
		}
	}
	names := make(map[int]string)
	if len(ids) == 0 {
		return names, nil
	}
	items, err := db.EntClient.Item.Query().Where(entitem.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		names[it.ID] = it.Name
	}
	return names, nil
}

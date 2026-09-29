package handlers

import (
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // embed zone data, the alpine image has none

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
)

type ItemStatistics struct {
	ID          int
	Name        string
	Type        string // item type, e.g. donation or transaction_costs
	SumAmount   int
	SumQuantity int
}

// DailyItemStatistics holds the sums of one item on one day (Europe/Vienna)
type DailyItemStatistics struct {
	Date        string // YYYY-MM-DD
	ItemID      int
	SumAmount   int
	SumQuantity int
}

// VendorSalesStatistics holds the sales of one vendor
type VendorSalesStatistics struct {
	VendorID    int
	LicenseID   string
	Name        string
	SumAmount   int
	SumQuantity int
}

// DailyPayoutStatistics holds the payouts to vendors on one day (Europe/Vienna)
type DailyPayoutStatistics struct {
	Date      string // YYYY-MM-DD
	Count     int
	SumAmount int
}

// PaymentsStatistics is the response to ListPaymentsStatistics
type PaymentsStatistics struct {
	From               time.Time
	To                 time.Time
	Items              []ItemStatistics
	Days               []DailyItemStatistics
	Payouts            []DailyPayoutStatistics
	TopVendors         []VendorSalesStatistics // best selling vendors by quantity
	TopVendorsByAmount []VendorSalesStatistics // best selling vendors by amount
}

// topVendorsCount is the number of vendors listed in PaymentsStatistics.TopVendors(ByAmount)
const topVendorsCount = 10

// statisticsLocation is the time zone used to group payments by day
var statisticsLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		log.Error("statisticsLocation: ", err)
		return time.UTC
	}
	return loc
}()

// ListPaymentsStatistics godoc
//
//	 	@Summary 		Calculate statistics of items & payments
//		@Description 	Filter by date, get statistical information, sorted by item.
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Success		200	{array}	PaymentsStatistics
//		@Security		KeycloakAuth
//		@Security		KeycloakAuth
//		@Router			/payments/statistics/ [get]
func ListPaymentsStatistics(w http.ResponseWriter, r *http.Request) {
	var err error

	// Get filter parameters
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")

	// Parse filter parameters
	var minDate, maxDate time.Time
	if minDateRaw != "" {
		minDate, err = time.Parse(time.RFC3339, minDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}
	if maxDateRaw != "" {
		maxDate, err = time.Parse(time.RFC3339, maxDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}

	// Get items (including archived ones, since payments may reference deleted items)
	items, err := database.Db.ListItemsIncludingArchived(false, false)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get payments with filter parameters
	payments, err := database.Db.ListPayments(minDate, maxDate, "", false, false, false, true, true)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	accounts, err := database.Db.ListAccounts()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Including disabled vendors, since they may have sold in the period
	vendors, err := database.Db.ListVendorsWithDisabled()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	paymentsStatistics, err := buildPaymentsStatistics(items, payments, accounts, vendors)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	paymentsStatistics.From = minDate
	paymentsStatistics.To = maxDate

	respond(w, err, paymentsStatistics)
}

// buildPaymentsStatistics sums up payments per item, per item and day and per vendor,
// and payouts per day
func buildPaymentsStatistics(items []database.Item, payments []database.Payment, accounts []database.Account, vendors []database.Vendor) (PaymentsStatistics, error) {
	accountsMap := make(map[int]database.Account)
	for _, account := range accounts {
		accountsMap[account.ID] = account
	}

	// Create map of items
	itemsMap := make(map[int]ItemStatistics)
	donationItemIDs := make(map[int]bool)
	transactionCostsItemIDs := make(map[int]bool)
	for _, item := range items {
		if item.Type == "donation" {
			donationItemIDs[item.ID] = true
		}
		if item.Type == "transaction_costs" {
			transactionCostsItemIDs[item.ID] = true
		}
		itemsMap[item.ID] = ItemStatistics{
			ID:          item.ID,
			Name:        item.Name,
			Type:        item.Type,
			SumAmount:   0,
			SumQuantity: 0,
		}
	}

	// Create sums per item and per item and day
	type dayKey struct {
		date   string
		itemID int
	}
	daysMap := make(map[dayKey]DailyItemStatistics)
	vendorsMap := make(map[int]VendorSalesStatistics)
	payoutsMap := make(map[string]DailyPayoutStatistics)
	for _, payment := range payments {
		if !payment.Item.Valid {
			// A payout is a payment without item to the cash account
			if accountsMap[payment.Receiver].Type == "Cash" {
				date := payment.Timestamp.In(statisticsLocation).Format("2006-01-02")
				payout := payoutsMap[date]
				payout.Date = date
				payout.Count++
				payout.SumAmount += payment.Amount
				payoutsMap[date] = payout
			}
			continue
		}
		itemID := int(payment.Item.Int64)
		if entry, ok := itemsMap[itemID]; ok {
			// Transaction costs are booked twice when the organisation takes them over
			// (vendor -> provider, orga -> vendor); only what goes to the provider is a cost
			if transactionCostsItemIDs[itemID] {
				receiverType := accountsMap[payment.Receiver].Type
				if receiverType != "Paypal" && receiverType != "VivaWallet" {
					continue
				}
			}

			// Donations and transaction costs store the amount as quantity, so count payments instead
			quantity := payment.Quantity
			if donationItemIDs[itemID] || transactionCostsItemIDs[itemID] {
				quantity = 1
			}

			// Transaction costs are expenses, not sales of the vendor
			if receiver := accountsMap[payment.Receiver]; payment.IsSale && receiver.Type == "Vendor" && receiver.Vendor.Valid && !transactionCostsItemIDs[itemID] {
				vendorID := int(receiver.Vendor.Int64)
				vendorEntry := vendorsMap[vendorID]
				vendorEntry.VendorID = vendorID
				vendorEntry.SumQuantity += quantity
				vendorEntry.SumAmount += payment.Amount
				vendorsMap[vendorID] = vendorEntry
			}
			entry.SumQuantity += quantity
			entry.SumAmount += payment.Amount
			itemsMap[itemID] = entry

			key := dayKey{payment.Timestamp.In(statisticsLocation).Format("2006-01-02"), itemID}
			day := daysMap[key]
			day.Date = key.date
			day.ItemID = itemID
			day.SumQuantity += quantity
			day.SumAmount += payment.Amount
			daysMap[key] = day
		} else {
			return PaymentsStatistics{}, errors.New("item not found")
		}
	}

	// Create payment statistics
	var paymentsStatistics PaymentsStatistics
	for _, item := range itemsMap {
		paymentsStatistics.Items = append(paymentsStatistics.Items, item)
	}
	for _, day := range daysMap {
		paymentsStatistics.Days = append(paymentsStatistics.Days, day)
	}
	sort.Slice(paymentsStatistics.Days, func(i, j int) bool {
		if paymentsStatistics.Days[i].Date != paymentsStatistics.Days[j].Date {
			return paymentsStatistics.Days[i].Date < paymentsStatistics.Days[j].Date
		}
		return paymentsStatistics.Days[i].ItemID < paymentsStatistics.Days[j].ItemID
	})
	for _, payout := range payoutsMap {
		paymentsStatistics.Payouts = append(paymentsStatistics.Payouts, payout)
	}
	sort.Slice(paymentsStatistics.Payouts, func(i, j int) bool {
		return paymentsStatistics.Payouts[i].Date < paymentsStatistics.Payouts[j].Date
	})

	vendorInfo := make(map[int]database.Vendor)
	for _, vendor := range vendors {
		vendorInfo[vendor.ID] = vendor
	}
	var vendorSales []VendorSalesStatistics
	for vendorID, entry := range vendorsMap {
		vendor := vendorInfo[vendorID]
		entry.LicenseID = vendor.LicenseID.String
		entry.Name = strings.TrimSpace(vendor.FirstName + " " + vendor.LastName)
		vendorSales = append(vendorSales, entry)
	}
	paymentsStatistics.TopVendors = topVendors(vendorSales, func(v VendorSalesStatistics) [2]int {
		return [2]int{v.SumQuantity, v.SumAmount}
	})
	paymentsStatistics.TopVendorsByAmount = topVendors(vendorSales, func(v VendorSalesStatistics) [2]int {
		return [2]int{v.SumAmount, v.SumQuantity}
	})

	return paymentsStatistics, nil
}

// topVendors returns the topVendorsCount vendors with the highest rank, a
// (primary, tie-breaker) pair; remaining ties are ordered by vendor ID
func topVendors(vendors []VendorSalesStatistics, rank func(VendorSalesStatistics) [2]int) []VendorSalesStatistics {
	sorted := slices.Clone(vendors)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := rank(sorted[i]), rank(sorted[j])
		if a != b {
			if a[0] != b[0] {
				return a[0] > b[0]
			}
			return a[1] > b[1]
		}
		return sorted[i].VendorID < sorted[j].VendorID
	})
	if len(sorted) > topVendorsCount {
		sorted = sorted[:topVendorsCount]
	}
	return sorted
}

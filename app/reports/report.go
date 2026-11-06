// Package reports builds the monthly accounting report and exports it as CSV and PDF.
package reports

import (
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // embed zone data, the alpine image has none

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
)

var log = utils.GetLogger()

// Location is the time zone the months of the report are cut in
var Location = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		log.Error("reports.Location: ", err)
		return time.UTC
	}
	return loc
}()

// ItemLine holds the webshop sales of one item in the month
type ItemLine struct {
	ItemID   int
	Name     string
	Type     string
	Quantity int
	Amount   int
}

// MonthlyReport holds the sums of one month, all amounts in cents
type MonthlyReport struct {
	Month       string    // YYYY-MM
	From        time.Time // start of the month
	To          time.Time // start of the next month, exclusive
	GeneratedAt time.Time

	// Income is the money that came in: OnlineSales + POSCash + ExternalPayments
	Income int
	// OnlineSales is what customers paid in the webshop, donations included
	OnlineSales int
	// OnlineOrders is the number of webshop orders with a payment in the month
	OnlineOrders int
	// POSCash is what vendors paid in cash at the POS
	POSCash int
	// POSCredit is what vendors paid at the POS with their credit; no money came in
	POSCredit int
	// ExternalPayments is what came in through other VivaWallet channels, such as a card terminal
	ExternalPayments int

	// LicenseIncome is what vendors paid the organisation for the digital licenses they sold
	LicenseIncome   int
	LicenseQuantity int

	// TransactionCosts is what the payment providers charged, external payments included
	TransactionCosts int
	// TransactionCostsCoveredByOrga is the part of TransactionCosts the organisation took over from the vendors
	TransactionCostsCoveredByOrga int

	// Payouts is the cash paid out to vendors
	Payouts     int
	PayoutCount int

	Items []ItemLine
}

// ParseMonth returns the start of the month (YYYY-MM) and the start of the next one
func ParseMonth(month string) (from, to time.Time, err error) {
	from, err = time.ParseInLocation("2006-01", month, Location)
	if err != nil {
		return from, to, fmt.Errorf("invalid month %q, expected YYYY-MM", month)
	}
	return from, from.AddDate(0, 1, 0), nil
}

// PreviousMonth returns the month (YYYY-MM) before the one t lies in
func PreviousMonth(t time.Time) string {
	t = t.In(Location)
	firstOfMonth := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, Location)
	return firstOfMonth.AddDate(0, -1, 0).Format("2006-01")
}

// ParseRecipients splits a comma separated list of email addresses
func ParseRecipients(s string) ([]string, error) {
	var recipients []string
	for _, r := range strings.Split(s, ",") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if strings.ContainsAny(r, "\r\n") {
			return nil, errors.New("recipient contains invalid characters")
		}
		addr, err := mail.ParseAddress(r)
		if err != nil {
			return nil, fmt.Errorf("invalid recipient %q: %w", r, err)
		}
		recipients = append(recipients, addr.Address)
	}
	return recipients, nil
}

// Build calculates the report of the month (YYYY-MM)
func Build(month string) (MonthlyReport, error) {
	from, to, err := ParseMonth(month)
	if err != nil {
		return MonthlyReport{}, err
	}

	items, err := database.Db.ListItemsIncludingArchived(false, false)
	if err != nil {
		return MonthlyReport{}, err
	}
	accounts, err := database.Db.ListAccounts()
	if err != nil {
		return MonthlyReport{}, err
	}
	payments, err := database.Db.ListPayments(from, to, "", false, false, false, false, false)
	if err != nil {
		return MonthlyReport{}, err
	}
	external, err := database.Db.ListExternalPayments(from, to)
	if err != nil {
		return MonthlyReport{}, err
	}

	report := build(from, to, items, accounts, payments, external)
	report.GeneratedAt = time.Now().In(Location)
	return report, nil
}

func build(from, to time.Time, items []database.Item, accounts []database.Account, payments []database.Payment, external []*ent.ExternalPayment) MonthlyReport {
	report := MonthlyReport{
		Month: from.In(Location).Format("2006-01"),
		From:  from,
		To:    to,
		Items: []ItemLine{},
	}

	accountTypes := make(map[int]string, len(accounts))
	for _, a := range accounts {
		accountTypes[a.ID] = a.Type
	}
	itemsByID := make(map[int]database.Item, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
	}

	lines := make(map[int]*ItemLine)
	orders := make(map[int64]bool)
	for _, p := range payments {
		// ListPayments includes the end of the period
		if p.Timestamp.Before(from) || !p.Timestamp.Before(to) {
			continue
		}
		sender, receiver := accountTypes[p.Sender], accountTypes[p.Receiver]

		if !p.Item.Valid {
			switch {
			case receiver == "Cash" && sender == "Vendor":
				report.Payouts += p.Amount
				report.PayoutCount++
			case p.IsPOS && sender == "Cash" && receiver == "Backoffice":
				report.POSCash += p.Amount
			case p.IsPOS && sender == "Orga" && receiver == "Backoffice":
				report.POSCredit += p.Amount
			}
			continue
		}

		item := itemsByID[int(p.Item.Int64)]
		switch {
		case item.Type == "transaction_costs":
			// Booked twice when the organisation takes them over (vendor -> provider, orga -> vendor)
			if receiver == "VivaWallet" || receiver == "Paypal" {
				report.TransactionCosts += p.Amount
			} else if sender == "Orga" {
				report.TransactionCostsCoveredByOrga += p.Amount
			}
		case item.IsLicenseItem || item.Type == "license_item":
			report.LicenseIncome += p.Amount
			report.LicenseQuantity += p.Quantity
		case sender == "UserAnon" || sender == "UserAuth":
			report.OnlineSales += p.Amount
			if p.Order.Valid {
				orders[p.Order.Int64] = true
			}
			line, ok := lines[item.ID]
			if !ok {
				line = &ItemLine{ItemID: item.ID, Name: item.Name, Type: item.Type}
				lines[item.ID] = line
			}
			// Donations store the amount as quantity, so count payments instead
			if item.Type == "donation" {
				line.Quantity++
			} else {
				line.Quantity += p.Quantity
			}
			line.Amount += p.Amount
		}
	}
	report.OnlineOrders = len(orders)

	for _, e := range external {
		report.ExternalPayments += e.Amount
		report.TransactionCosts += e.Fee
	}

	report.Income = report.OnlineSales + report.POSCash + report.ExternalPayments

	for _, line := range lines {
		report.Items = append(report.Items, *line)
	}
	sort.Slice(report.Items, func(i, j int) bool {
		if report.Items[i].Amount != report.Items[j].Amount {
			return report.Items[i].Amount > report.Items[j].Amount
		}
		return report.Items[i].ItemID < report.Items[j].ItemID
	})
	return report
}

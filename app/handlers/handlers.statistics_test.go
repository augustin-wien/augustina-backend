package handlers

import (
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestBuildPaymentsStatistics(t *testing.T) {
	items := []database.Item{
		{ID: 10, Name: "Zeitung", Type: "issue"},
		{ID: 93, Name: "donation", Type: "donation"},
		{ID: 94, Name: "transactionCosts", Type: "transaction_costs"},
	}
	accounts := []database.Account{
		{ID: 1, Type: "UserAnon"},
		{ID: 2, Type: "Vendor", Vendor: null.IntFrom(7)},
		{ID: 3, Type: "Vendor", Vendor: null.IntFrom(8)},
		{ID: 4, Type: "Paypal"},
		{ID: 5, Type: "Orga"},
	}
	vendors := []database.Vendor{
		{ID: 7, LicenseID: null.StringFrom("fl-7"), FirstName: "Maria", LastName: "Huber"},
		{ID: 8, LicenseID: null.StringFrom("fl-8"), FirstName: "Josef", LastName: "Wagner"},
	}
	day2 := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	payments := []database.Payment{
		// 23:30 UTC is already the next day in Vienna
		{Item: null.IntFrom(10), Sender: 1, Receiver: 2, IsSale: true, Quantity: 2, Amount: 600, Timestamp: time.Date(2026, 9, 1, 23, 30, 0, 0, time.UTC)},
		{Item: null.IntFrom(10), Sender: 1, Receiver: 3, IsSale: true, Quantity: 1, Amount: 300, Timestamp: day2},
		// Donations store the amount as quantity and are not sales
		{Item: null.IntFrom(93), Sender: 1, Receiver: 3, Quantity: 150, Amount: 150, Timestamp: day2},
		{Item: null.IntFrom(93), Sender: 1, Receiver: 3, Quantity: 200, Amount: 200, Timestamp: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)},
		// Transaction costs taken over by the organisation are booked twice
		{Item: null.IntFrom(94), Sender: 2, Receiver: 4, Quantity: 27, Amount: 27, Timestamp: day2},
		{Item: null.IntFrom(94), Sender: 5, Receiver: 2, Quantity: 27, Amount: 27, Timestamp: day2},
		{Sender: 2, Receiver: 5, Quantity: 5, Amount: 500, Timestamp: time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)},
	}

	stats, err := buildPaymentsStatistics(items, payments, accounts, vendors)
	require.NoError(t, err)

	sums := make(map[int]ItemStatistics)
	for _, item := range stats.Items {
		sums[item.ID] = item
	}
	require.Equal(t, 3, sums[10].SumQuantity)
	require.Equal(t, 900, sums[10].SumAmount)
	require.Equal(t, 2, sums[93].SumQuantity)
	require.Equal(t, 350, sums[93].SumAmount)
	require.Equal(t, 1, sums[94].SumQuantity)
	require.Equal(t, 27, sums[94].SumAmount)

	require.Equal(t, []DailyItemStatistics{
		{Date: "2026-09-02", ItemID: 10, SumQuantity: 3, SumAmount: 900},
		{Date: "2026-09-02", ItemID: 93, SumQuantity: 1, SumAmount: 150},
		{Date: "2026-09-02", ItemID: 94, SumQuantity: 1, SumAmount: 27},
		{Date: "2026-09-03", ItemID: 93, SumQuantity: 1, SumAmount: 200},
	}, stats.Days)

	require.Equal(t, []VendorSalesStatistics{
		{VendorID: 7, LicenseID: "fl-7", Name: "Maria Huber", SumQuantity: 2, SumAmount: 600},
		{VendorID: 8, LicenseID: "fl-8", Name: "Josef Wagner", SumQuantity: 1, SumAmount: 300},
	}, stats.TopVendors)
	require.Equal(t, stats.TopVendors, stats.TopVendorsByAmount)

	_, err = buildPaymentsStatistics(items, []database.Payment{{Item: null.IntFrom(999)}}, accounts, vendors)
	require.Error(t, err)
}

func TestBuildPaymentsStatisticsLimitsTopVendors(t *testing.T) {
	items := []database.Item{{ID: 10, Name: "Zeitung", Type: "issue"}}
	var accounts []database.Account
	var payments []database.Payment
	for i := 1; i <= topVendorsCount+2; i++ {
		accounts = append(accounts, database.Account{ID: i, Type: "Vendor", Vendor: null.IntFrom(int64(i))})
		payments = append(payments, database.Payment{Item: null.IntFrom(10), Receiver: i, IsSale: true, Quantity: i, Amount: i * 300})
	}

	stats, err := buildPaymentsStatistics(items, payments, accounts, nil)
	require.NoError(t, err)
	require.Len(t, stats.TopVendors, topVendorsCount)
	require.Equal(t, topVendorsCount+2, stats.TopVendors[0].VendorID)
	require.Len(t, stats.TopVendorsByAmount, topVendorsCount)
}

func TestBuildPaymentsStatisticsTopVendorsByAmount(t *testing.T) {
	items := []database.Item{{ID: 10, Name: "Zeitung", Type: "issue"}}
	accounts := []database.Account{
		{ID: 1, Type: "Vendor", Vendor: null.IntFrom(1)},
		{ID: 2, Type: "Vendor", Vendor: null.IntFrom(2)},
	}
	payments := []database.Payment{
		// Vendor 1 sells more pieces, vendor 2 earns more
		{Item: null.IntFrom(10), Receiver: 1, IsSale: true, Quantity: 5, Amount: 500},
		{Item: null.IntFrom(10), Receiver: 2, IsSale: true, Quantity: 2, Amount: 900},
	}

	stats, err := buildPaymentsStatistics(items, payments, accounts, nil)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, []int{stats.TopVendors[0].VendorID, stats.TopVendors[1].VendorID})
	require.Equal(t, []int{2, 1}, []int{stats.TopVendorsByAmount[0].VendorID, stats.TopVendorsByAmount[1].VendorID})
}

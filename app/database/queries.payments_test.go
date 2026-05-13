package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// Test_ListPayments_Performance tests the performance of ListPayments with many transactions
func Test_ListPayments_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// Initialize empty DB
	Db.InitEmptyTestDb()

	// 1. Setup Data
	// Create Vendors and Accounts
	senderVendorID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom("sender")})
	require.NoError(t, err)

	receiverVendorID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom("receiver")})
	require.NoError(t, err)

	senderAccount, err := Db.GetAccountByVendorID(senderVendorID)
	require.NoError(t, err)
	receiverAccount, err := Db.GetAccountByVendorID(receiverVendorID)
	require.NoError(t, err)

	// Create 5000 Sales (should be skipped by optimization)
	salesCount := 5000
	// Create 200 Payouts (should be batched)
	payoutsCount := 200

	totalCount := salesCount + payoutsCount
	payments := make([]Payment, 0, totalCount)

	// Add Sales
	for i := 0; i < salesCount; i++ {
		payments = append(payments, Payment{
			Sender:       senderAccount.ID,
			Receiver:     receiverAccount.ID,
			Amount:       10,
			AuthorizedBy: "test",
			IsSale:       true,
		})
	}

	// Add Payouts
	for i := 0; i < payoutsCount; i++ {
		payments = append(payments, Payment{
			Sender:       senderAccount.ID,
			Receiver:     receiverAccount.ID,
			Amount:       1000,
			AuthorizedBy: "test",
			IsSale:       false, // Triggers sub-payment lookup
		})
	}

	fmt.Printf("Seeding %d payments (%d sales, %d payouts)...\n", totalCount, salesCount, payoutsCount)
	err = Db.CreatePayments(payments)
	require.NoError(t, err)
	fmt.Println("Seeding complete.")

	// 2. Measure ListPayments performance
	start := time.Now()

	// Query covering the whole range
	minDate := time.Now().Add(-2 * time.Hour)
	maxDate := time.Now().Add(2 * time.Hour)

	fetchedPayments, err := Db.ListPayments(minDate, maxDate, "", false, false, false, false, false)
	require.NoError(t, err)

	duration := time.Since(start)

	// 3. Ascertain results
	require.Equal(t, totalCount, len(fetchedPayments))

	fmt.Printf("ListPayments took %v for %d transactions\n", duration, totalCount) // Performance assertion (loose, just to catch major regressions)
	// 5000 queries (N+1) would take significantly longer (e.g. > 1-2 seconds even locally).
	// Batch query should be very fast (< 500ms).
	if duration > 1000*time.Millisecond {
		t.Errorf("ListPayments took too long: %v", duration)
	}
}

// Test_ListPayments_Correctness verifies that sub-payments are correctly attached
// when using the batch fetching optimization.
func Test_ListPayments_Correctness(t *testing.T) {
	Db.InitEmptyTestDb()

	// Create Vendor
	vendorID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom("vendor-1")})
	require.NoError(t, err)
	vendor, err := Db.GetVendor(vendorID)
	require.NoError(t, err)

	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)

	// Create some sales
	salesCount := 10
	var sales []Payment
	for i := 0; i < salesCount; i++ {
		sales = append(sales, Payment{
			Sender:       vendorAccount.ID,
			Receiver:     vendorAccount.ID, // Self-payment for test? or to Cash
			Amount:       10,
			AuthorizedBy: "test",
			IsSale:       true,
		})
	}

	// We need to create them to get IDs.
	// Since CreatePayments doesn't return IDs, we create one by one for this test.
	for i := range sales {
		id, err := Db.CreatePayment(sales[i])
		require.NoError(t, err)
		sales[i].ID = id // Update ID
	}

	// Create a Payout that groups these sales
	payoutID, err := Db.CreatePaymentPayout(vendor, vendorAccount.ID, "admin", 100, sales)
	require.NoError(t, err)

	// List Payments and check structure
	// We expect:
	// - 10 Sales payments (IsSale=true)
	// - 1 Payout payment (IsSale=false, Sender=Vendor, Receiver=Cash)
	// - The Payout payment should have IsPayoutFor populated with the 10 sales

	results, err := Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false, false, false)
	require.NoError(t, err)

	var payoutPayment *Payment
	for i := range results {
		if results[i].ID == payoutID {
			payoutPayment = &results[i]
			break
		}
	}

	require.NotNil(t, payoutPayment, "Payout payment not found in results")
	require.False(t, payoutPayment.IsSale)
	require.Len(t, payoutPayment.IsPayoutFor, salesCount, "Payout should have %d sub-payments", salesCount)

	// Verify IDs match
	subIds := make(map[int]bool)
	for _, sub := range payoutPayment.IsPayoutFor {
		subIds[sub.ID] = true
	}

	for _, sale := range sales {
		require.True(t, subIds[sale.ID], "Sale ID %d missing from payout sub-list", sale.ID)
	}
}

// Test_ListPayments_VendorFilter_HijackSafe ensures that the vendor filter works
// and is resistant to basic SQL injection attempts (hijacking).
func Test_ListPayments_VendorFilter_HijackSafe(t *testing.T) {
	Db.InitEmptyTestDb()

	// 1. Setup Data
	vendor1Lic := "10000000"
	vendor2Lic := "20000000"

	// Create Vendors
	v1ID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom(vendor1Lic)})
	require.NoError(t, err)
	v2ID, err := Db.CreateVendor(Vendor{LicenseID: null.StringFrom(vendor2Lic)})
	require.NoError(t, err)

	v1Account, err := Db.GetAccountByVendorID(v1ID)
	require.NoError(t, err)
	v2Account, err := Db.GetAccountByVendorID(v2ID)
	require.NoError(t, err)

	// Create Payments
	// Payment for Vendor 1
	err = Db.CreatePayments([]Payment{{
		Sender:       v1Account.ID,
		Receiver:     v1Account.ID, // Self payment? Or just involving V1
		Amount:       100,
		AuthorizedBy: "test",
		IsSale:       true,
		Timestamp:    time.Now(),
	}})
	require.NoError(t, err)

	// Payment for Vendor 2
	err = Db.CreatePayments([]Payment{{
		Sender:       v2Account.ID,
		Receiver:     v2Account.ID,
		Amount:       200,
		AuthorizedBy: "test",
		IsSale:       true,
		Timestamp:    time.Now(),
	}})
	require.NoError(t, err)

	// 2. Test Valid Filter
	minDate := time.Now().Add(-24 * time.Hour)
	maxDate := time.Now().Add(24 * time.Hour)

	payments, err := Db.ListPayments(minDate, maxDate, vendor1Lic, false, false, false, false, false)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Equal(t, 100, payments[0].Amount)

	// 3. Test Invalid Filter (Hijack Attempt)
	// Passing a string that attempts SQL injection.
	// Since GetVendorByLicenseID uses ent (ORM) which uses parameterized queries,
	// this should simply fail to find the vendor and return an error.
	hijackPayload := "10000000' OR '1'='1"
	paymentsHijack, err := Db.ListPayments(minDate, maxDate, hijackPayload, false, false, false, false, false)

	// Expectation: Error because vendor not found (LicenseID equality check fails), OR empty list if strict.
	// Current implementation returns error if vendor not found.
	require.Error(t, err)
	require.Nil(t, paymentsHijack)
	require.Contains(t, err.Error(), "GetVendorByLicenseID")

	// 4. Test Partial Match (should not work)
	partialPayload := "100000"
	paymentsPartial, err := Db.ListPayments(minDate, maxDate, partialPayload, false, false, false, false, false)
	require.Error(t, err)
	require.Nil(t, paymentsPartial)
}

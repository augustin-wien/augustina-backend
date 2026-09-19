package database

import (
	"errors"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// Test_AboPurchaseCompleteFlow is a full end-to-end test of the abonement purchase
// flow inside VerifyOrderAndCreatePayments. It asserts every step in order:
//  1. Customer DB record created from the order's CustomerEmail
//  2. Abonement DB record created, linked to the customer, active, 1-year duration
//  3. The abonement's license group is saved on the customer record
//     (Keycloak sync is attempted but silently skipped — nil client in unit tests)
//  4. The abonement confirmation email is sent to the customer
//  5. The latest published online_issue for that license group is findable, confirming
//     the issue-LG infrastructure is in place (the LG itself was already assigned in step 3
//     so processedLicenseGroups prevents a redundant second assignment)
func Test_AboPurchaseCompleteFlow(t *testing.T) {
	Db.InitEmptyTestDb()

	const (
		customerEmail = "full-flow-customer@example.com"
		licenseGroup  = "full_flow_edition"
	)

	// --- Setup ---

	vendor := Vendor{
		FirstName: "Full",
		LastName:  "FlowVendor",
		Email:     "full-flow-vendor@vendor.com",
		LicenseID: null.StringFrom("ffv-001"),
	}
	vendorID, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	// license_item that the abonement item references
	licenseItem := Item{
		Name:          "Full Flow License Item",
		Description:   "License item for the full purchase flow test",
		Price:         100,
		Type:          "license_item",
		IsLicenseItem: true,
		LicenseGroup:  null.StringFrom(licenseGroup),
	}
	licenseItemID, err := Db.CreateItem(licenseItem)
	utils.CheckError(t, err)

	// abonement item — must have LicenseItem.Valid so the verification flow runs
	aboItem := Item{
		Name:         "Full Flow Abonement",
		Description:  "Abonement item for the full purchase flow test (description long enough)",
		Price:        1200,
		Type:         "abonement",
		Disabled:     false,
		LicenseItem:  null.IntFrom(int64(licenseItemID)),
		LicenseGroup: null.StringFrom(licenseGroup),
	}
	aboItemID, err := Db.CreateItem(aboItem)
	utils.CheckError(t, err)

	// published online_issue carrying the same license group so
	// GetLatestPublishedOnlineIssueByLicenseGroup can find it
	onlineIssue := Item{
		Name:         "Full Flow Online Issue",
		Description:  "The latest published online issue for the full purchase flow test",
		Price:        500,
		Type:         "online_issue",
		Disabled:     false,
		LicenseGroup: null.StringFrom(licenseGroup),
	}
	onlineIssueID, err := Db.CreateItem(onlineIssue)
	utils.CheckError(t, err)

	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	order := Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom(customerEmail),
		Entries: []OrderEntry{
			{
				Item:     aboItemID,
				Quantity: 1,
				Sender:   vendorAccount.ID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	// intercept abonement confirmation email before triggering verification
	var capturedEmailTemplate string
	var capturedEmailRecipient string
	origBuild := BuildEmailRequestFromTemplate
	defer func() { BuildEmailRequestFromTemplate = origBuild }()
	BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		if name == "abonementConfirmation" {
			capturedEmailTemplate = name
			if len(to) > 0 {
				capturedEmailRecipient = to[0]
			}
		}
		return nil, nil
	}

	// --- Exercise ---

	err = Db.VerifyOrderAndCreatePayments(orderID, 1)
	utils.CheckError(t, err)

	// --- Assert ---

	// Step 1: customer DB record created from the order's CustomerEmail
	customer, err := Db.GetCustomerByEmail(customerEmail)
	utils.CheckError(t, err)
	require.NotNil(t, customer)
	require.Equal(t, customerEmail, customer.Email)

	// Step 2: abonement DB record created, linked to customer, active, 1-year duration
	abonements, err := Db.ListAbonementsByCustomer(customer.ID)
	utils.CheckError(t, err)
	require.Len(t, abonements, 1)
	abo := abonements[0]
	require.Equal(t, customer.ID, abo.CustomerID)
	require.Equal(t, aboItemID, abo.ItemID)
	require.Equal(t, "active", abo.Status)
	require.WithinDuration(t, time.Now(), abo.FromDate, 5*time.Second)
	require.WithinDuration(t, time.Now().AddDate(1, 0, 0), abo.ToDate, 5*time.Second)

	// Step 3: license group saved on customer DB record
	require.Contains(t, customer.LicenseGroups, licenseGroup)

	// Step 4: abonement confirmation email sent to the customer
	require.Equal(t, "abonementConfirmation", capturedEmailTemplate)
	require.Equal(t, customerEmail, capturedEmailRecipient)

	// Step 5: latest published online_issue for the license group is findable,
	// confirming the issue-LG infrastructure is wired correctly
	latestIssue, found, err := Db.GetLatestPublishedOnlineIssueByLicenseGroup(licenseGroup)
	utils.CheckError(t, err)
	require.True(t, found, "published online_issue with licenseGroup %q should be findable", licenseGroup)
	require.Equal(t, onlineIssueID, latestIssue.ID)
}

func Test_VerifyOrderSetsVerifiedAt(t *testing.T) {
	Db.InitEmptyTestDb()

	// 1. Create Vendor
	vendor := Vendor{
		FirstName: "Test",
		LastName:  "Vendor",
		Email:     "test@vendor.com",
		LicenseID: null.StringFrom("tv-123"),
	}
	vendorID, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	// 2. Create Item
	item := Item{
		Name:        "Test Item",
		Description: "This is a test item description that is long enough",
		Price:       100,
		Type:        "normal_item",
	}
	itemID, err := Db.CreateItem(item)
	utils.CheckError(t, err)

	// 3. Create Order
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	order := Order{
		OrderCode: null.StringFrom("test-order"),
		Vendor:    vendorID,
		Entries: []OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Sender:   vendorAccount.ID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	// 4. Verify Order
	err = Db.VerifyOrderAndCreatePayments(orderID, 1)
	utils.CheckError(t, err)

	// 5. Check VerifiedAt
	updatedOrder, err := Db.GetOrderByID(orderID)
	utils.CheckError(t, err)

	require.True(t, updatedOrder.Verified)
	require.True(t, updatedOrder.VerifiedAt.Valid)
	require.WithinDuration(t, time.Now(), updatedOrder.VerifiedAt.Time, 5*time.Second)
}

// Test_GetUnsyncedOrders exercises the worklist query added to find orders whose Odoo
// webhook delivery failed or never ran, closing the gap that previously required
// hand-picking order IDs (see scripts/resend_odoo_orders.py) instead of querying for them.
func Test_GetUnsyncedOrders(t *testing.T) {
	Db.InitEmptyTestDb()

	originalOdooURL := config.Config.OdooWebhookURL
	config.Config.OdooWebhookURL = "https://odoo.example.com/webhook"
	defer func() { config.Config.OdooWebhookURL = originalOdooURL }()

	vendor := Vendor{
		FirstName: "Test",
		LastName:  "Vendor",
		Email:     "test@vendor.com",
		LicenseID: null.StringFrom("tv-sync-123"),
	}
	vendorID, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	item := Item{
		Name:        "Test Sync Item",
		Description: "This is a test item description that is long enough",
		Price:       100,
		Type:        "normal_item",
	}
	itemID, err := Db.CreateItem(item)
	utils.CheckError(t, err)

	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	order := Order{
		OrderCode: null.StringFrom("test-order-sync"),
		Vendor:    vendorID,
		Entries: []OrderEntry{
			{
				Item:     itemID,
				Quantity: 1,
				Sender:   vendorAccount.ID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	err = Db.VerifyOrderAndCreatePayments(orderID, 1)
	utils.CheckError(t, err)

	// An unverified order never shows up in the unsynced worklist even if it has never been
	// synced - only verified orders are relevant, matching what GetUnsyncedOrders filters on.
	unsynced, err := Db.GetUnsyncedOrders()
	utils.CheckError(t, err)
	require.Len(t, unsynced, 1)
	require.Equal(t, orderID, unsynced[0].ID)

	// Recording a failure keeps it in the worklist and records the error text.
	err = Db.SetOdooSyncFailure(orderID, errors.New("exceeded maximum retries, aborting"))
	utils.CheckError(t, err)

	unsynced, err = Db.GetUnsyncedOrders()
	utils.CheckError(t, err)
	require.Len(t, unsynced, 1)

	failedOrder, err := Db.GetOrderByID(orderID)
	utils.CheckError(t, err)
	require.True(t, failedOrder.OdooSyncError.Valid)
	require.Equal(t, "exceeded maximum retries, aborting", failedOrder.OdooSyncError.String)

	// Recording success removes it from the worklist and clears the prior error.
	err = Db.SetOdooSyncSuccess(orderID)
	utils.CheckError(t, err)

	unsynced, err = Db.GetUnsyncedOrders()
	utils.CheckError(t, err)
	require.Empty(t, unsynced)

	syncedOrder, err := Db.GetOrderByID(orderID)
	utils.CheckError(t, err)
	require.True(t, syncedOrder.OdooSyncedAt.Valid)
	require.False(t, syncedOrder.OdooSyncError.Valid)
}

func Test_VerifyOrderCreatesAbonementForAboItem(t *testing.T) {
	Db.InitEmptyTestDb()

	// 1. Create Vendor
	vendor := Vendor{
		FirstName: "Abo",
		LastName:  "Vendor",
		Email:     "abo-test-vendor@vendor.com",
		LicenseID: null.StringFrom("atv-123"),
	}
	vendorID, err := Db.CreateVendor(vendor)
	utils.CheckError(t, err)

	// 2. Create license_item
	licenseItem := Item{
		Name:          "Test License Item For Abo",
		Description:   "License for test abonement item",
		Price:         100,
		Type:          "license_item",
		IsLicenseItem: true,
		LicenseGroup:  null.StringFrom("test_edition"),
	}
	licenseItemID, err := Db.CreateItem(licenseItem)
	utils.CheckError(t, err)

	// 3. Create abonement item (enabled, linked license_item, license group)
	aboItem := Item{
		Name:         "Test Abonement Item",
		Description:  "A test abonement item with a sufficiently long description",
		Price:        1000,
		Type:         "abonement",
		Disabled:     false,
		LicenseItem:  null.IntFrom(int64(licenseItemID)),
		LicenseGroup: null.StringFrom("test_edition"),
	}
	aboItemID, err := Db.CreateItem(aboItem)
	utils.CheckError(t, err)

	// 4. Create Order with customer email and the abonement item
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	order := Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom("abo-customer@example.com"),
		Entries: []OrderEntry{
			{
				Item:     aboItemID,
				Quantity: 1,
				Sender:   vendorAccount.ID,
				Receiver: vendorAccount.ID,
				IsSale:   true,
			},
		},
	}
	orderID, err := Db.CreateOrder(order)
	utils.CheckError(t, err)

	// 5. Intercept abonement confirmation email
	var capturedTemplate string
	var capturedRecipient string
	origBuild := BuildEmailRequestFromTemplate
	defer func() { BuildEmailRequestFromTemplate = origBuild }()
	BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		if name == "abonementConfirmation" {
			capturedTemplate = name
			if len(to) > 0 {
				capturedRecipient = to[0]
			}
		}
		return nil, nil // no real mail sending in unit tests
	}

	// 6. Verify Order — Keycloak calls return errors (nil client) but are logged and skipped
	err = Db.VerifyOrderAndCreatePayments(orderID, 1)
	utils.CheckError(t, err)

	// 7. Customer record must have been created
	customer, err := Db.GetCustomerByEmail("abo-customer@example.com")
	utils.CheckError(t, err)
	require.NotNil(t, customer)

	// 8. Exactly one abonement must have been created for that customer
	abonements, err := Db.ListAbonementsByCustomer(customer.ID)
	utils.CheckError(t, err)
	require.Len(t, abonements, 1)

	abo := abonements[0]
	require.Equal(t, customer.ID, abo.CustomerID)
	require.Equal(t, aboItemID, abo.ItemID)
	require.Equal(t, "active", abo.Status)
	require.WithinDuration(t, time.Now(), abo.FromDate, 5*time.Second)
	require.WithinDuration(t, time.Now().AddDate(1, 0, 0), abo.ToDate, 5*time.Second)

	// 9. Abonement confirmation email must have been sent to the customer
	require.Equal(t, "abonementConfirmation", capturedTemplate)
	require.Equal(t, "abo-customer@example.com", capturedRecipient)
}

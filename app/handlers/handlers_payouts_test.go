package handlers

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestPayments tests CRUD operations on payments
func TestPayments(t *testing.T) {
	defer mutex_test.Unlock()
	mutex_test.Lock()
	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Set up a payment account
	senderVendorID, err := database.Db.CreateVendor(
		database.Vendor{LicenseID: null.StringFrom("Test sender"), Email: "testSender@augustina.cc"},
	)
	if err != nil {
		t.Error(err)
	}

	receiverVendorID, err := database.Db.CreateVendor(
		database.Vendor{LicenseID: null.StringFrom("Test receiver"), Email: "testReceiver@augustina.cc"},
	)
	if err != nil {
		t.Error(err)
	}

	_, err = database.Db.CreateVendor(
		database.Vendor{LicenseID: null.StringFrom("Test inactive"), Email: "testInactive@augustina.cc"},
	)
	if err != nil {
		t.Error(err)
	}

	itemID, err := database.Db.CreateItem(database.Item{Name: "Test item for payments", Description: "Payment test item description", Price: 314, Type: "normal_item"})
	if err != nil {
		t.Error(err)
	}

	utils.CheckError(t, err)

	testSenderAccount, err := database.Db.GetAccountByVendorID(senderVendorID)
	utils.CheckError(t, err)

	testReceiverAccount, err := database.Db.GetAccountByVendorID(receiverVendorID)
	utils.CheckError(t, err)

	// Create payments via API
	p1, err := database.Db.CreatePayment(
		database.Payment{
			Sender:       testSenderAccount.ID,
			Receiver:     testReceiverAccount.ID,
			SenderName:   null.StringFrom("Test sender"),
			ReceiverName: null.StringFrom("Test receiver"),
			Amount:       314,
			Quantity:     1,
			Item:         null.IntFrom(int64(itemID)),
		},
	)
	if err != nil {
		t.Error(err)
	}
	response2 := utils.TestRequestWithAuth(t, r, "GET", "/api/payments/", nil, 200, adminUserToken)

	// Unmarshal response
	var payments []database.Payment
	err = json.Unmarshal(response2.Body.Bytes(), &payments)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(payments))
	if t.Failed() {
		return
	}
	require.Equal(t, 1, len(payments))
	if t.Failed() {
		return
	}

	// Test payments response
	require.Equal(t, payments[0].Amount, 314)
	require.Equal(t, payments[0].Sender, testSenderAccount.ID)
	require.Equal(t, payments[0].Receiver, testReceiverAccount.ID)
	require.Equal(t, payments[0].SenderName, null.StringFrom("Test sender"))
	require.Equal(t, payments[0].ReceiverName, null.StringFrom("Test receiver"))
	require.WithinDuration(t, time.Now(), payments[0].Timestamp, time.Minute)

	// Test account balances
	require.Equal(t, 0, testSenderAccount.Balance)
	require.Equal(t, 0, testReceiverAccount.Balance)

	// Test time filters
	timeRequest(t, 0, 0, 1)
	timeRequest(t, -1, 1, 1)
	timeRequest(t, -2, -1, 0)
	timeRequest(t, 1, -1, 0)
	timeRequest(t, 1, 0, 0)
	timeRequest(t, 0, 1, 1)
	timeRequest(t, -1, 0, 1)
	timeRequest(t, 0, -1, 0)

	// Test statistics
	p2, err := database.Db.CreatePayment(
		database.Payment{
			Sender:       testSenderAccount.ID,
			Receiver:     testReceiverAccount.ID,
			SenderName:   null.StringFrom("Test sender"),
			ReceiverName: null.StringFrom("Test receiver"),
			Amount:       314,
			Quantity:     1,
			Item:         null.IntFrom(int64(itemID)),
		},
	)
	utils.CheckError(t, err)
	response3 := utils.TestRequestWithAuth(t, r, "GET", "/api/payments/statistics/?from=2020-01-01T00:00:00Z&to=2999-01-01T00:00:00Z", nil, 200, adminUserToken)
	var statistics PaymentsStatistics
	err = json.Unmarshal(response3.Body.Bytes(), &statistics)
	utils.CheckError(t, err)
	require.Equal(t, statistics.From.String(), "2020-01-01 00:00:00 +0000 UTC")
	require.Equal(t, statistics.To.String(), "2999-01-01 00:00:00 +0000 UTC")
	for _, item := range statistics.Items {
		if item.ID == itemID {
			require.Equal(t, item.SumAmount, 628)
			require.Equal(t, item.SumQuantity, 2)
		}
	}

	// Test vendor usage statistics; disabled and blocked vendors don't count
	_, err = database.Db.CreateVendor(database.Vendor{
		FirstName: "Disabled", LastName: "Vendor", Email: "disabled-usage@vendor.com",
		LicenseID: null.StringFrom("usage-disabled"), IsDisabled: true,
	})
	utils.CheckError(t, err)
	_, err = database.Db.CreateVendor(database.Vendor{
		FirstName: "Blocked", LastName: "Vendor", Email: "blocked-usage@vendor.com",
		LicenseID: null.StringFrom("usage-blocked"), IsBlocked: true, BlockedNote: "test",
	})
	utils.CheckError(t, err)
	response4 := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/statistics/?from=2020-01-01T00:00:00Z&to=2999-01-01T00:00:00Z", nil, 200, adminUserToken)
	var vendorUsageStatistics VendorUsageStatistics
	err = json.Unmarshal(response4.Body.Bytes(), &vendorUsageStatistics)
	utils.CheckError(t, err)
	require.Equal(t, 3, vendorUsageStatistics.TotalVendors)
	require.Equal(t, 2, vendorUsageStatistics.UsedVendors)
	require.Equal(t, 1, vendorUsageStatistics.UnusedVendors)
	require.InDelta(t, 66.6667, vendorUsageStatistics.UsedPercentage, 0.001)
	require.InDelta(t, 33.3333, vendorUsageStatistics.UnusedPercentage, 0.001)

	// Clean up
	database.Db.DeletePayment(p1)
	database.Db.DeletePayment(p2)
	database.Db.DeleteItem(itemID)
}

// TestPaymentPayout tests CRUD operations on payment payouts
func TestPaymentPayout(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	keycloak.KeycloakClient.DeleteUser("testpaymentpayout@example.com")
	keycloak.KeycloakClient.DeleteUser("testotherlicenseid@example.com")

	vendorLicenseId := "testpaymentpayout"
	vendorID := createTestVendor(t, vendorLicenseId)
	vendorIDInt, _ := strconv.Atoi(vendorID)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendorIDInt)
	utils.CheckError(t, err)
	anonUserAccount, err := database.Db.GetAccountByType("UserAnon")
	utils.CheckError(t, err)

	// Create a payment to the vendor
	_, err = database.Db.CreatePayment(
		database.Payment{
			Sender:   anonUserAccount.ID,
			Receiver: vendorAccount.ID,
			Amount:   314,
			IsSale:   true,
		})
	utils.CheckError(t, err)
	// Create a payment from the vendor (should be substracted in payout)
	_, err = database.Db.CreatePayment(
		database.Payment{
			Sender:   vendorAccount.ID,
			Receiver: anonUserAccount.ID,
			Amount:   1,
			IsSale:   true,
		})

	utils.CheckError(t, err)
	// Create invalid payout: vendor with no open payments → amount = 0 → 400
	emptyVendorID := createTestVendor(t, "testpayoutEmpty")
	defer keycloak.KeycloakClient.DeleteUser("testpayoutEmpty@example.com")
	f := createPaymentPayoutRequest{
		VendorLicenseID: "testpayoutEmpty",
	}
	_ = emptyVendorID
	res := utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 400, adminUserToken)
	require.Equal(t, res.Body.String(), `{"error":{"message":"payout amount must be bigger than 0"}}`)

	// Create valid payout
	f = createPaymentPayoutRequest{
		VendorLicenseID: vendorLicenseId,
	}

	account, err := database.Db.GetAccountByVendorID(vendorIDInt)
	utils.CheckError(t, err)

	// Try to check first
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/payments/forpayout/?vendor="+vendorLicenseId, f, 200, adminUserToken)
	var paymentsResp paymentsResponse
	err = json.Unmarshal(res.Body.Bytes(), &paymentsResp)
	utils.CheckError(t, err)
	require.Equal(t, 2, len(paymentsResp.Payments))

	res = utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 200, adminUserToken)

	payoutPaymentID := res.Body.String()
	payoutPaymentIDInt, err := strconv.Atoi(payoutPaymentID)
	if err != nil {
		t.Error(err)
	}

	payoutPayment, err := database.Db.GetPayment(payoutPaymentIDInt)
	if err != nil {
		t.Error(err)
	}

	cashAccount, err := database.Db.GetAccountByType("Cash")
	if err != nil {
		t.Error(err)
	}

	require.Equal(t, payoutPayment.Amount, 314-1)
	require.Equal(t, payoutPayment.Sender, account.ID)
	require.Equal(t, payoutPayment.SenderName, null.StringFrom(account.Name))
	require.Equal(t, payoutPayment.Receiver, cashAccount.ID)
	require.Equal(t, payoutPayment.ReceiverName, null.StringFrom(cashAccount.Name))
	require.Equal(t, payoutPayment.AuthorizedBy, adminUserEmail)

	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseId)
	utils.CheckError(t, err)

	require.Equal(t, vendor.Balance, 0)
	require.Equal(t, cashAccount.Balance, 314-1)
	require.Equal(t, vendor.LastPayout.Time.Day(), time.Now().Day())
	require.WithinDuration(t, time.Now(), vendor.LastPayout.Time, time.Minute)

	// Test GET payment filters for payout
	otherVendorID := createTestVendor(t, "testOTHERLicenseID")
	otherVendorIDInt, _ := strconv.Atoi(otherVendorID)
	otherVendorAccount, _ := database.Db.GetAccountByVendorID(otherVendorIDInt)

	_, err = database.Db.CreatePayment(database.Payment{
		Sender:   anonUserAccount.ID,
		Receiver: otherVendorAccount.ID,
		Amount:   100,
		IsSale:   true,
	})
	utils.CheckError(t, err)

	var payouts []database.Payment
	response := utils.TestRequestWithAuth(t, r, "GET", "/api/payments/?payouts=true&vendor="+vendorLicenseId, nil, 200, adminUserToken)
	err = json.Unmarshal(response.Body.Bytes(), &payouts)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(payouts))
	require.Equal(t, payouts[0].Amount, 314-1)

	response1 := utils.TestRequestWithAuth(t, r, "GET", "/api/payments/?payouts=true", nil, 200, adminUserToken)
	err = json.Unmarshal(response1.Body.Bytes(), &payouts)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(payouts))
	require.Equal(t, 2, len(payouts[0].IsPayoutFor))

	response2 := utils.TestRequestWithAuth(t, r, "GET", "/api/payments/?payouts=true&vendor=testOTHERLicenseID", nil, 200, adminUserToken)
	err = json.Unmarshal(response2.Body.Bytes(), &payouts)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(payouts))

	response3 := utils.TestRequestWithAuth(t, r, "GET", "/api/payments/", nil, 200, adminUserToken)
	err = json.Unmarshal(response3.Body.Bytes(), &payouts)
	utils.CheckError(t, err)
	require.Equal(t, 4, len(payouts))

	// Check that there are no more payments for payout
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/payments/forpayout/?vendor="+vendorLicenseId, f, 200, adminUserToken)
	var payoutResp paymentsResponse
	err = json.Unmarshal(res.Body.Bytes(), &payoutResp)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(payoutResp.Payments))

	// Clean up after test
	keycloak.KeycloakClient.DeleteUser(vendorLicenseId)
	keycloak.KeycloakClient.DeleteUser("testotherlicenseid@example.com")

}

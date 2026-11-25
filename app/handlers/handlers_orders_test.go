package handlers

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestOrders tests CRUD operations on orders
// TODO: Test independent of vivawallet
func TestOrders(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	customerEmail := "test_customer_for_test@example.com"

	keycloak.KeycloakClient.DeleteUser("testorders123@example.com")
	keycloak.KeycloakClient.DeleteUser(customerEmail)
	keycloak.KeycloakClient.DeleteUser("testdeadlock@example.com")
	orders, _ := database.Db.GetOrders()
	for _, order := range orders {
		database.Db.DeleteOrder(order.ID)
	}
	// Set up a payment account
	vendorLicenseId := "testorders123"
	vendorID := createTestVendor(t, vendorLicenseId)
	vendorIDInt, _ := strconv.Atoi(vendorID)
	_, err := database.Db.GetAccountByVendorID(vendorIDInt)
	utils.CheckError(t, err)

	// Test that maxOrderAmount is set and cannot be exceeded
	setMaxOrderAmount(t, 10)

	// Ensure mail templates exist so emails can be built during verification
	err = database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Your license", "Please access your license at {{.URL}}")
	utils.CheckError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("welcome", "Welcome", "Welcome to Augustina")
	utils.CheckError(t, err)

	itemID := CreateTestItem(t, "testordersItemWithoutLicense", 20, "", "")
	request := `{
		"entries": [
			{
			  "item": ` + itemID + `,
			  "quantity": 1
			}
		  ],
		  "vendorLicenseID": "` + vendorLicenseId + `"
	}`
	resWithoutLicense := utils.TestRequestStr(t, r, "POST", "/api/orders/", request, 400)
	require.Equal(t, resWithoutLicense.Body.String(), `{"error":{"message":"order amount is too high"}}`)

	itemIDWithLicense, licenseItemID := CreateTestItemWithLicense(t)
	itemIDWithLicenseInt, _ := strconv.Atoi(itemIDWithLicense)
	licenseItemIDInt, _ := strconv.Atoi(licenseItemID)

	requestWithoutEmail := `{
		"entries": [
			{
			  "item": ` + itemIDWithLicense + `,
			  "quantity": 2
			}
		  ],
		  "vendorLicenseID": "` + vendorLicenseId + `"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithoutEmail, 400)
	require.Equal(t, res.Body.String(), `{"error":{"message":"you are not allowed to purchase this item without a customer email"}}`)

	// Create order with customer email and order amount to high
	requestWithEmail := `{
		"entries": [
			{
			  "item": ` + itemIDWithLicense + `,
			  "quantity": 2
			}
		  ],
		  "vendorLicenseID": "` + vendorLicenseId + `",
		  "customerEmail": "` + customerEmail + `"
	}`
	res2 := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithEmail, 400)
	require.Equal(t, res2.Body.String(), `{"error":{"message":"order amount is too high"}}`)

	// Check that order cannot contain duplicate items
	jsonPost := `{
			"entries": [
				{
				  "item": ` + itemIDWithLicense + `,
				  "quantity": 2
				},
				{
					"item": ` + itemIDWithLicense + `,
					"quantity": 2
				}
			  ],
			  "vendorLicenseID": "` + vendorLicenseId + `",
			  "customerEmail": "` + customerEmail + `"
		}`
	res3 := utils.TestRequestStr(t, r, "POST", "/api/orders/", jsonPost, 400)
	require.Equal(t, res3.Body.String(), `{"error":{"message":"nice try! You are not supposed to have duplicate item ids in your order request"}}`)

	// Test order with customer email and order amount not to high

	// Set max order amount to 5000 so that order can be created
	setMaxOrderAmount(t, 5000)

	res4 := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithEmail, 200)

	var respMap map[string]string
	err = json.Unmarshal(res4.Body.Bytes(), &respMap)
	utils.CheckError(t, err)
	checkoutURL := respMap["SmartCheckoutURL"]
	// In test mode we may get either the real Viva wallet URL or a local test URL.
	// Assert it's a URL that indicates a checkout/success redirect.
	require.True(t, strings.Contains(checkoutURL, "checkout") || strings.Contains(checkoutURL, "success"))

	parsedURL, err := url.Parse(checkoutURL)
	utils.CheckError(t, err)
	orderCode := parsedURL.Query().Get("s")

	// Check if order was created
	orders, _ = database.Db.GetOrders()
	require.Equal(t, 1, len(orders))

	// Use the created order rather than relying on a fixed order code
	order := orders[0]
	// In environments where the Viva checkout URL is used (?ref= format), s= is absent;
	// fall back to the order code stored in the DB.
	if orderCode == "" {
		orderCode = order.OrderCode.String
	}
	// Test order amount
	orderTotal := order.GetTotal()
	require.Equal(t, orderTotal, 20*2)

	senderAccount, err := database.Db.GetAccountByType("UserAnon")
	if err != nil {
		t.Error(err)
	}
	receiverAccount, err := database.Db.GetAccountByVendorID(vendorIDInt)
	if err != nil {
		t.Error(err)
	}

	require.Equal(t, order.Vendor, vendorIDInt)
	require.Equal(t, order.Verified, false)
	require.Equal(t, order.CustomerEmail, null.StringFrom(customerEmail))
	require.Equal(t, order.Entries[0].Item, licenseItemIDInt)
	require.Equal(t, order.Entries[1].Item, itemIDWithLicenseInt)
	require.Equal(t, order.Entries[1].Quantity, 2)
	require.Equal(t, order.Entries[1].Price, 20)
	require.Equal(t, order.Entries[1].Sender, senderAccount.ID)
	require.Equal(t, order.Entries[1].Receiver, receiverAccount.ID)

	// Verify order and create payments
	// verify for deadlock
	errc := make(chan error, 1)
	go func() {
		e := database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
		errc <- e
	}()

	resSuccess := utils.TestRequestStr(t, r, "GET", "/api/orders/verify/?s="+orderCode+"&t=0", "", 200)
	require.Equal(t, strings.Contains(resSuccess.Body.String(), vendorLicenseId), true)

	// Wait for the background verification to finish and check its error
	e := <-errc
	utils.CheckError(t, e)
	// Check payments
	payments, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false, false, false)
	if err != nil {
		t.Error(err)
	}

	require.GreaterOrEqual(t, len(payments), 2)
	// Ensure one of the payments matches the expected total for the item
	found := false
	for _, p := range payments {
		if p.Amount == 20*2 {
			found = true
			break
		}
	}
	require.True(t, found)

	// Check balances
	senderAccount, err = database.Db.GetAccountByType("UserAnon")
	if err != nil {
		t.Error(err)
	}
	receiverAccount, err = database.Db.GetAccountByVendorID(vendorIDInt)
	if err != nil {
		t.Error(err)
	}
	require.Equal(t, senderAccount.Balance, -40)
	require.Equal(t, receiverAccount.Balance, 34)
	// 2*3 has been payed for license item

	// Check if customer has been added to keycloak
	user, err := keycloak.KeycloakClient.GetUser(customerEmail)
	if err != nil {
		panic(err)
	}

	// Check if customer was added to group
	groups, err := keycloak.KeycloakClient.GetUserGroups(*user.ID)
	if err != nil {
		t.Error(err)
	}
	require.GreaterOrEqual(t, len(groups), 2)
	// Ensure expected groups exist (order may vary)
	hasCustomer := false
	hasEdition := false
	for _, g := range groups {
		if *g.Name == "customer" {
			hasCustomer = true
		}
		if *g.Name == "testedition" {
			hasEdition = true
		}
	}
	require.True(t, hasCustomer)
	require.True(t, hasEdition)

	// Cleanup
	for _, payment := range payments {
		database.Db.DeletePayment(payment.ID)
	}

	// Clean up after test
	paymentOrder, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false, false, false)
	utils.CheckError(t, err)
	for _, payment := range paymentOrder {
		database.Db.DeletePayment(payment.ID)
	}

	for _, entry := range order.Entries {
		database.Db.DeleteOrderEntry(entry.ID)
	}
	database.Db.DeleteOrder(order.ID)
}

func TestListUnverifiedOrders(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Save original config
	originalDebugPayments := config.Config.DEBUG_payments
	defer func() {
		config.Config.DEBUG_payments = originalDebugPayments
	}()

	config.Config.DEBUG_payments = true

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	require.NoError(t, err)

	// Create Vendor
	vendorLicenseID := "testunverifiedorders"
	createTestVendor(t, vendorLicenseID)

	// Create Item
	itemIDStr := CreateTestItem(t, "Test Item", 100, "", "")
	itemID, _ := strconv.Atoi(itemIDStr)

	// Create Unverified Order
	reqData := createOrderRequest{
		Entries: []createOrderRequestEntry{
			{Item: itemID, Quantity: 1},
		},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom("unverified@example.com"),
	}
	utils.TestRequest(t, r, "POST", "/api/orders/", reqData, 200)

	// Create Verified Order
	reqDataVerified := createOrderRequest{
		Entries: []createOrderRequestEntry{
			{Item: itemID, Quantity: 1},
		},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom("verified@example.com"),
	}
	resVerified := utils.TestRequest(t, r, "POST", "/api/orders/", reqDataVerified, 200)
	var responseVerified createOrderResponse
	err = json.Unmarshal(resVerified.Body.Bytes(), &responseVerified)
	require.NoError(t, err)

	// Parse URL to get OrderCode
	u, err := url.Parse(responseVerified.SmartCheckoutURL)
	require.NoError(t, err)
	orderCode := u.Query().Get("s")
	require.NotEmpty(t, orderCode)

	// Get Order ID from DB
	orderVerified, err := database.Db.GetOrderByOrderCode(orderCode)
	require.NoError(t, err)

	t.Logf("Verified Order ID: %d", orderVerified.ID)

	// Manually verify the second order
	err = database.Db.EntClient.Order.UpdateOneID(orderVerified.ID).SetVerified(true).Exec(context.Background())
	require.NoError(t, err)
	// t.Logf("Rows affected: %d", tag.RowsAffected()) // Ent handles this check implicitly by error if not found

	// Test List Unverified Orders
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/orders/unverified/", nil, 200, adminUserToken)

	var orders []database.Order
	err = json.Unmarshal(res.Body.Bytes(), &orders)
	require.NoError(t, err)

	require.Len(t, orders, 1)
	require.Equal(t, "unverified@example.com", orders[0].CustomerEmail.String)
}

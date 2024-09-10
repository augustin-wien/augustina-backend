package handlers

import (
	"augustin/config"
	"augustin/database"
	"augustin/keycloak"
	"augustin/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

var r *chi.Mux
var adminUser string
var adminUserEmail string
var adminUserToken *gocloak.JWT
var mutex_test sync.Mutex

// TestMain is executed before all tests and initializes an empty database
func TestMain(m *testing.M) {
	var err error
	// run tests in mainfolder
	err = os.Chdir("..")
	if err != nil {
		panic(err)
	}
	config.InitConfig()

	// Initialize database and empty it
	// Note: Emptying does not work in Github Actions
	err = database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}
	// Initialize keycloak
	err = keycloak.InitializeOauthServer()
	if err != nil {
		panic(err)
	}

	r = GetRouter()
	adminUserEmail = "testadmin@example.com"
	defer func() {
		keycloak.KeycloakClient.DeleteUser(adminUser)
		keycloak.KeycloakClient.DeleteUser(adminUserEmail)
	}()
	keycloak.KeycloakClient.DeleteUser(adminUserEmail)
	adminUser, err = keycloak.KeycloakClient.CreateUser("testadmin", "testadmin", "testadmin", adminUserEmail, "password")
	if err != nil {
		log.Errorf("TestMain: Create user failed testadmin: %v \n", err)
	}
	err = keycloak.KeycloakClient.AssignRole(adminUser, "admin")
	if err != nil {
		log.Errorf("TestMain: Assign role failed: %v \n", err)
	}
	adminUserToken, err = keycloak.KeycloakClient.GetUserToken(adminUserEmail, "password")
	if err != nil {
		log.Errorf("TestMain: Login failed: %v \n", err)
	}
	fmt.Println("Created admin keycloak token")

	returnCode := m.Run()
	err = keycloak.KeycloakClient.DeleteUser(adminUserEmail)
	if err != nil {
		log.Errorf("TestMain: Delete user failed: %v \n", err)
	}

	os.Exit(returnCode)

	os.Exit(m.Run())
}

// TestHelloWorld tests the hello world test function
func TestHelloWorld(t *testing.T) {
	res := utils.TestRequest(t, r, "GET", "/api/hello/", nil, 200)
	require.Equal(t, "\"Hello, world!\"", res.Body.String())
}

func TestHelloWorldAuth(t *testing.T) {
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/auth/hello/", nil, 200, adminUserToken)
	require.Equal(t, "\"Hello, world!\"", res.Body.String())

}

func createTestVendor(t *testing.T, licenseID string) string {
	jsonVendor := `{
		"keycloakID": "test",
		"urlID": "test",
		"licenseID": "` + licenseID + `",
		"firstName": "test1234",
		"lastName": "test",
		"email": "` + licenseID + `@example.com",
		"telephone": "+43123456789",
		"VendorSince": "1/22",
		"PLZ": "1234",
		"Longitude": 16.363449,
		"Latitude": 48.210033
	}`
	res := utils.TestRequestStrWithAuth(t, r, "POST", "/api/vendors/", jsonVendor, 200, adminUserToken)
	vendorID := res.Body.String()
	return vendorID
}

// TestVendors tests CRUD operations on users
func TestVendors(t *testing.T) {
	vendorLicenseId := "testlicenseid1"
	vendorEmail := vendorLicenseId + "@example.com"
	// vendorPassword := "password"
	err := keycloak.KeycloakClient.DeleteUser(vendorEmail)
	if err != nil {
		log.Infof("Delete user %v failed, which is okey: %v \n", vendorLicenseId, err)
	}

	// Initialize database and empty it
	err = database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// // Create
	// vendorID := createTestVendor(t, vendorLicenseId)
	// keycloak.KeycloakClient.UpdateUserPassword(vendorEmail, vendorPassword)

	// Query ListVendors only returns few fields (not all) under /api/vendors/
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	var vendors []database.Vendor
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 5, len(vendors))
	// require.Equal(t, "test1234", vendors[3].FirstName)
	// require.Equal(t, vendorLicenseId, vendors[3].LicenseID.String)
	// require.Equal(t, "test", vendors[3].LastName)

	// // Check if licenseID exists and returns first name of vendor
	// res = utils.TestRequest(t, r, "GET", "/api/vendors/check/"+vendorLicenseId+"/", nil, 200)
	// require.Equal(t, res.Body.String(), `{"FirstName":"test1234"}`)

	// // GetVendorByID returns all fields under /api/vendors/{id}/
	// utils.TestRequest(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 401)
	// res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 200, adminUserToken)
	// var vendor database.Vendor
	// err = json.Unmarshal(res.Body.Bytes(), &vendor)
	// utils.CheckError(t, err)
	// require.Equal(t, "test1234", vendor.FirstName)
	// require.Equal(t, "+43123456789", vendor.Telephone)
	// require.Equal(t, "1/22", vendor.VendorSince)
	// require.Equal(t, "1234", vendor.PLZ)
	// require.Equal(t, vendorEmail, vendor.Email)

	// // Update
	// var vendors2 []database.Vendor
	// jsonVendor := `{"firstName": "nameAfterUpdate", "licenseID": "IDAfterUpdate", "email": "` + vendorEmail + `", "Longitude": 16.363449, "Latitude": 48.210033}`
	// utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/", jsonVendor, 200, adminUserToken)
	// res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	// err = json.Unmarshal(res.Body.Bytes(), &vendors2)
	// utils.CheckError(t, err)
	// require.Equal(t, 6, len(vendors2))
	// require.Equal(t, "nameAfterUpdate", vendors2[1].FirstName)

	// // Test location data
	// var mapData []database.LocationData
	// res = utils.TestRequestWithAuth(t, r, "GET", "/api/map/", nil, 200, adminUserToken)
	// err = json.Unmarshal(res.Body.Bytes(), &mapData)
	// utils.CheckError(t, err)
	// require.Equal(t, 6, len(mapData))
	// require.Equal(t, 16.363449, mapData[5].Longitude)
	// require.Equal(t, 48.210033, mapData[5].Latitude)

	// // Delete
	// utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+vendorID+"/", nil, 204, adminUserToken)
	// res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	// err = json.Unmarshal(res.Body.Bytes(), &vendors)
	// utils.CheckError(t, err)
	// require.Equal(t, 5, len(vendors))

	// // Clean up after test
	// keycloak.KeycloakClient.DeleteUser(vendorEmail)

}

func CreateTestItem(t *testing.T, name string, price int, licenseItemID string, licenseGroup string) string {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", name)
	writer.WriteField("Price", strconv.Itoa(price))
	if licenseItemID != "" {
		writer.WriteField("LicenseItem", licenseItemID)
	}
	if licenseGroup != "" {
		writer.WriteField("LicenseGroup", licenseGroup)
	}
	writer.Close()
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	itemID := res.Body.String()
	return itemID
}

// TestItems tests CRUD operations on items (including images)
// Todo: delete file after test
func TestItems(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create
	itemID := CreateTestItem(t, "Test item", 314, "", "")

	// Read
	res := utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	var resItems []database.Item
	err = json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)

	// For C.I. pipeline
	if len(resItems) == 3 && resItems[1].Name == "" {
		// Remove empty item
		database.Db.DeleteItem(resItems[1].ID)
		res := utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
		err = json.Unmarshal(res.Body.Bytes(), &resItems)
		utils.CheckError(t, err)
		log.Info("res items name 1", resItems[0].Name)
		log.Info("res items name 2", resItems[1].Name)
		require.Equal(t, len(resItems), 2)
	}

	require.Equal(t, len(resItems) == 2, true)
	require.Equal(t, "Test item", resItems[1].Name)

	// Update (multipart form!)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", "Updated item name")
	writer.WriteField("Price", strconv.Itoa(10))
	writer.WriteField("nonexistingfieldname", "10")
	image, _ := writer.CreateFormFile("Image", "test.jpg")
	image.Write([]byte(`i am the content of a jpg file :D`))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+itemID+"/", body, writer.FormDataContentType(), 200, adminUserToken)

	// Read
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)
	require.Equal(t, 2, len(resItems))
	require.Equal(t, "Updated item name", resItems[1].Name)
	require.Contains(t, resItems[1].Image, "test")
	require.Contains(t, resItems[1].Image, ".jpg")

	// Check file
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	file, err := os.ReadFile(dir + "/" + resItems[1].Image)
	utils.CheckError(t, err)
	require.Equal(t, `i am the content of a jpg file :D`, string(file))

	// Update with image as field (not as a file)
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "Updated item name 2")
	writer.WriteField("Image", "Test")
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+itemID+"/", body, writer.FormDataContentType(), 200, adminUserToken)

	// Read
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)
	require.Equal(t, 2, len(resItems))
	require.Equal(t, "Updated item name 2", resItems[1].Name)
	require.Equal(t, resItems[1].Image, "Test")

	// Update item with certain ID (which should fail)
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("ID", "2")
	writer.WriteField("Image", "Test")
	writer.Close()
	res = utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/2/", body, writer.FormDataContentType(), 400, adminUserToken)

	require.Equal(t, res.Body.String(), `{"error":{"message":"Nice try! You are not allowed to update this item"}}`)

}

// Set MaxOrderAmount to avoid errors
func setMaxOrderAmount(t *testing.T, amount int) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("MaxOrderAmount", strconv.Itoa(amount))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, writer.FormDataContentType(), 200, adminUserToken)

	// Check if maxOrderAmount is set
	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	var settings database.Settings
	err := json.Unmarshal(res.Body.Bytes(), &settings)
	utils.CheckError(t, err)
	require.Equal(t, amount, settings.MaxOrderAmount)
}

func CreateTestItemWithLicense(t *testing.T) (string, string) {
	licenseItemID := CreateTestItem(t, "License item", 3, "", "")
	itemID := CreateTestItem(t, "Test item", 20, licenseItemID, "testedition")
	return itemID, licenseItemID
}

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
	require.Equal(t, resWithoutLicense.Body.String(), `{"error":{"message":"Order amount is too high"}}`)

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
	require.Equal(t, res2.Body.String(), `{"error":{"message":"Order amount is too high"}}`)

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
	require.Equal(t, res3.Body.String(), `{"error":{"message":"Nice try! You are not supposed to have duplicate item ids in your order request"}}`)

	// Test order with customer email and order amount not to high

	// Set max order amount to 5000 so that order can be created
	setMaxOrderAmount(t, 5000)

	res4 := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithEmail, 200)

	require.Equal(t, res4.Body.String(), `{"SmartCheckoutURL":"`+config.Config.VivaWalletSmartCheckoutURL+`0"}`)

	// Check if order was created
	orders, _ = database.Db.GetOrders()
	require.Equal(t, 1, len(orders))

	// Check if order was created correctly

	order, err := database.Db.GetOrderByOrderCode("0")
	if err != nil {
		t.Error(err)
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
	c := make(chan int)
	go func() {
		err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
		utils.CheckError(t, err)
		c <- 1
	}()

	resSuccess := utils.TestRequestStr(t, r, "GET", "/api/orders/verify/?s=0&t=0", "", 200)
	require.Equal(t, strings.Contains(resSuccess.Body.String(), vendorLicenseId), true)
	utils.CheckError(t, err)

	if <-c != 1 {
		t.Error("Webhook did not finish")
	}
	// Check payments
	payments, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false)
	if err != nil {
		t.Error(err)
	}

	require.Equal(t, 2, len(payments))
	require.Equal(t, payments[1].Amount, 20*2)

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
	require.Equal(t, 2, len(groups))
	require.Equal(t, *groups[0].Name, "customer")
	require.Equal(t, *groups[1].Name, "testedition")

	// Cleanup
	for _, payment := range payments {
		database.Db.DeletePayment(payment.ID)
	}

	// Clean up after test
	paymentOrder, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false)
	utils.CheckError(t, err)
	for _, payment := range paymentOrder {
		database.Db.DeletePayment(payment.ID)
	}

	for _, entry := range order.Entries {
		database.Db.DeleteOrderEntry(entry.ID)
	}
	database.Db.DeleteOrder(order.ID)
}

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
		database.Vendor{LicenseID: null.StringFrom("Test sender"), Email: "testSender@account.com",},
	)
	if err != nil {
		t.Error(err)
	}

	receiverVendorID, err := database.Db.CreateVendor(
		database.Vendor{LicenseID: null.StringFrom("Test receiver"), Email: "testReceiver@account.com",},
	)
	if err != nil {
		t.Error(err)
	}

	itemID, err := database.Db.CreateItem(database.Item{Name: "Test item for payments", Price: 314})
	if err != nil {
		t.Error(err)
	}

	utils.CheckError(t, err)

	senderAccount2, err := database.Db.GetAccountByVendorID(senderVendorID)
	utils.CheckError(t, err)

	receiverAccount2, err := database.Db.GetAccountByVendorID(receiverVendorID)
	utils.CheckError(t, err)

	// Create payments via API
	p1, err := database.Db.CreatePayment(
		database.Payment{
			Sender:       senderAccount2.ID,
			Receiver:     receiverAccount2.ID,
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
	require.Equal(t, payments[0].Sender, senderAccount2.ID)
	require.Equal(t, payments[0].Receiver, receiverAccount2.ID)
	require.Equal(t, payments[0].SenderName, null.StringFrom("Test sender"))
	require.Equal(t, payments[0].ReceiverName, null.StringFrom("Test receiver"))
	require.Equal(t, payments[0].Timestamp.Day(), time.Now().Day())
	require.Equal(t, payments[0].Timestamp.Hour(), time.Now().UTC().Hour())

	// Test account balances
	require.Equal(t, 0, senderAccount2.Balance)
	require.Equal(t, 0, receiverAccount2.Balance)

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
			Sender:       senderAccount2.ID,
			Receiver:     receiverAccount2.ID,
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

	// Clean up
	database.Db.DeletePayment(p1)
	database.Db.DeletePayment(p2)
	database.Db.DeleteItem(itemID)
}

func timeRequest(t *testing.T, from int, to int, expectedLength int) {
	var payments []database.Payment
	path := "/api/payments/"
	if from != 0 || to != 0 {
		path += "?"
	}
	if from != 0 {
		path += "from=" + time.Now().UTC().Add(time.Duration(from)*time.Hour).Format(time.RFC3339)
	}
	if from != 0 && to != 0 {
		path += "&"
	}
	if to != 0 {
		path += "to=" + time.Now().UTC().Add(time.Duration(to)*time.Hour).Format(time.RFC3339)
	}
	response := utils.TestRequestWithAuth(t, r, "GET", path, nil, 200, adminUserToken)
	err := json.Unmarshal(response.Body.Bytes(), &payments)
	utils.CheckError(t, err)
	require.Equal(t, expectedLength, len(payments))
	return
}

// TestPaymentPayout tests CRUD operations on payment payouts
func TestPaymentPayout(t *testing.T) {
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
	// Create invalid payout
	f := createPaymentPayoutRequest{
		VendorLicenseID: vendorLicenseId,
		From:            time.Now().Add(time.Duration(-200) * time.Hour),
		To:              time.Now().Add(time.Duration(-100) * time.Hour),
	}
	res := utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 400, adminUserToken)
	require.Equal(t, res.Body.String(), `{"error":{"message":"payout amount must be bigger than 0"}}`)

	// Create payments via API
	f = createPaymentPayoutRequest{
		VendorLicenseID: vendorLicenseId,
		From:            time.Now().Add(time.Duration(-100) * time.Hour),
		To:              time.Now().Add(time.Duration(+100) * time.Hour),
	}

	account, err := database.Db.GetAccountByVendorID(vendorIDInt)
	utils.CheckError(t, err)

	// Try to check first
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/payments/forpayout/?vendor="+vendorLicenseId, f, 200, adminUserToken)
	var payments []database.Payment
	err = json.Unmarshal(res.Body.Bytes(), &payments)
	utils.CheckError(t, err)
	require.Equal(t, 2, len(payments))

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
	require.Equal(t, vendor.LastPayout.Time.Hour(), time.Now().Hour())

	// Test GET payment filters for payout
	createTestVendor(t, "testOTHERLicenseID")
	database.Db.CreatePayment(database.Payment{})

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
	require.Equal(t, 3, len(payouts))

	// Check that there are no more payments for payout
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/payments/forpayout/?vendor="+vendorLicenseId, f, 200, adminUserToken)
	var payoutPaymentsAfter []database.Payment
	err = json.Unmarshal(res.Body.Bytes(), &payoutPaymentsAfter)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(payoutPaymentsAfter))

	// Clean up after test
	keycloak.KeycloakClient.DeleteUser(vendorLicenseId)
	keycloak.KeycloakClient.DeleteUser("testotherlicenseid@example.com")

}

// TestSettings tests GET and PUT operations on settings
func TestSettings(t *testing.T) {

	itemID := CreateTestItem(t, "Test main item", 314, "", "")

	// Update (multipart form!)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("MaxOrderAmount", strconv.Itoa(10))
	writer.WriteField("MainItem", itemID)
	image, _ := writer.CreateFormFile("Logo", "test.png")
	image.Write([]byte(`i am the content of a jpg file :D`))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, writer.FormDataContentType(), 200, adminUserToken)

	// Read
	var settings database.Settings
	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	err := json.Unmarshal(res.Body.Bytes(), &settings)
	utils.CheckError(t, err)
	require.Equal(t, "img/logo.png", settings.Logo)
	require.Equal(t, 10, settings.MaxOrderAmount)

	// Check item join
	require.Equal(t, "Test main item", settings.MainItemName.String)
	require.Equal(t, int64(314), settings.MainItemPrice.Int64)

	// Check file
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	file, err := os.ReadFile(dir + "/" + settings.Logo)
	utils.CheckError(t, err)
	require.Equal(t, `i am the content of a jpg file :D`, string(file))

}

func TestVendorsOverview(t *testing.T) {

	// Me

	// Create Vendor
	vendorLicenseId := "testvendoroverview"
	vendorEmail := vendorLicenseId + "@example.com"
	vendorPassword := "password"
	randomUserEmail := "randomuser@example.com"

	err := keycloak.KeycloakClient.DeleteUser(vendorEmail)
	if err != nil {
		log.Infof("Delete user failed which is okey because it's for cleanup: %v \n", err)
	}
	defer func() {
		keycloak.KeycloakClient.DeleteUser(vendorEmail)
		keycloak.KeycloakClient.DeleteUser(randomUserEmail)
	}()
	vendorID := createTestVendor(t, vendorLicenseId)
	keycloak.KeycloakClient.UpdateUserPassword(vendorEmail, vendorPassword)

	vendorIDInt, _ := strconv.Atoi(vendorID)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendorIDInt)
	utils.CheckError(t, err)
	anonUserAccount, err := database.Db.GetAccountByType("UserAnon")
	utils.CheckError(t, err)
	// Create payments via API
	_, err = database.Db.CreatePayment(
		database.Payment{
			Sender:       anonUserAccount.ID,
			Receiver:     vendorAccount.ID,
			SenderName:   null.StringFrom("Test sender"),
			ReceiverName: null.StringFrom("Test receiver"),
			Amount:       314,
		})

	utils.CheckError(t, err)

	vendorToken, err := keycloak.KeycloakClient.GetUserToken(vendorEmail, vendorPassword)
	if err != nil {
		panic(err)
	}
	// test me endpoint
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/me/", nil, 200, vendorToken)
	var meVendor VendorOverview
	err = json.Unmarshal(res.Body.Bytes(), &meVendor)
	utils.CheckError(t, err)
	require.Equal(t, "test1234", meVendor.FirstName)
	require.Equal(t, "+43123456789", meVendor.Telephone)
	require.Equal(t, "1234", meVendor.PLZ)
	require.Equal(t, vendorEmail, meVendor.Email)
	require.Equal(t, 314, meVendor.Balance)
	require.Equal(t, 1, len(meVendor.OpenPayments))

	// Test if vendor can't see other vendors
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 403, vendorToken)

	// Test if admin who is no vendor can't see vendor overview
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/me/", nil, 400, adminUserToken)

	// test if random user can see vendor overview
	_, err = keycloak.KeycloakClient.CreateUser(randomUserEmail, randomUserEmail, randomUserEmail, randomUserEmail, "password")
	if err != nil {
		log.Errorf("TestVendorsOverview: Create user failed: %v \n", err)
	}
	randomUserToken, err := keycloak.KeycloakClient.GetUserToken(randomUserEmail, "password")
	if err != nil {
		panic(err)
	}
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/me/", nil, 403, randomUserToken)

	// Clean up after test
	keycloak.KeycloakClient.DeleteUser(vendorEmail)
	keycloak.KeycloakClient.DeleteUser(randomUserEmail)
}

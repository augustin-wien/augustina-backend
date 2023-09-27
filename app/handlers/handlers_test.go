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
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

var r *chi.Mux
var adminUser string
var adminUserToken *gocloak.JWT

// TestMain is executed before all tests and initializes an empty database
func TestMain(m *testing.M) {
	var err error
	database.Db.InitEmptyTestDb()
	keycloak.InitializeOauthServer()
	r = GetRouter()
	defer func() {
		keycloak.KeycloakClient.DeleteUser(adminUser)
		keycloak.KeycloakClient.DeleteUser("test123@example.com")
	}()
	adminUser, err = keycloak.KeycloakClient.CreateUser("testadmin", "testadmin", "testadmin@example.com", "password")
	if err != nil {
		log.Errorf("Create user failed: %v \n", err)
	}
	err = keycloak.KeycloakClient.AssignRole(adminUser, "admin")
	if err != nil {
		log.Errorf("Assign role failed: %v \n", err)
	}
	adminUserToken, err = keycloak.KeycloakClient.GetUserToken("testadmin@example.com", "password")
	if err != nil {
		log.Errorf("Login failed: %v \n", err)
	}
	fmt.Println("Admin user token: ", adminUserToken.AccessToken)

	return_code := m.Run()
	err = keycloak.KeycloakClient.DeleteUser(adminUser)
	if err != nil {
		log.Errorf("Delete user failed: %v \n", err)
	}

	os.Exit(return_code)
}

// TestHelloWorld tests the hello world test function
func TestHelloWorld(t *testing.T) {
	res := utils.TestRequest(t, r, "GET", "/api/hello/", nil, 200)
	require.Equal(t, "\"Hello, world!\"", res.Body.String())
}

func createTestVendor(t *testing.T, licenseID string) string {
	jsonVendor := `{
		"keycloakID": "test",
		"urlID": "test",
		"licenseID": "` + licenseID + `",
		"firstName": "test1234",
		"lastName": "test",
		"email": "test123@example.com"
	}`
	res := utils.TestRequestStrWithAuth(t, r, "POST", "/api/vendors/", jsonVendor, 200, adminUserToken)
	vendorID := res.Body.String()
	return vendorID
}

// TestVendors tests CRUD operations on users
func TestVendors(t *testing.T) {
	keycloak.KeycloakClient.DeleteUser("test123@example.com")
	database.Db.InitEmptyTestDb()

	// Create
	vendorID := createTestVendor(t, "testLicenseID1")

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	var vendors []database.Vendor
	err := json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))
	require.Equal(t, "test1234", vendors[0].FirstName)
	require.Equal(t, "testLicenseID1", vendors[0].LicenseID.String)

	// Check if licenseID exists and returns first name of vendor
	res = utils.TestRequest(t, r, "GET", "/api/vendors/check/testLicenseID1/", nil, 200)
	require.Equal(t, res.Body.String(), `{"FirstName":"test1234"}`)

	// Update
	var vendors2 []database.Vendor
	jsonVendor := `{"firstName": "nameAfterUpdate"}`
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/", jsonVendor, 200, adminUserToken)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	err = json.Unmarshal(res.Body.Bytes(), &vendors2)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors2))
	require.Equal(t, "nameAfterUpdate", vendors2[0].FirstName)

	// Delete
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+vendorID+"/", nil, 204, adminUserToken)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(vendors))

}

func CreateTestItem(t *testing.T) string {
	f := `{
		"Name": "Test item",
		"Price": 314
	}`
	res := utils.TestRequestStrWithAuth(t, r, "POST", "/api/items/", f, 200, adminUserToken)
	itemID := res.Body.String()
	return itemID
}

// TestItems tests CRUD operations on items (including images)
// Todo: delete file after test
func TestItems(t *testing.T) {
	database.Db.InitEmptyTestDb()

	// Create
	itemID := CreateTestItem(t)

	// Read
	res := utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	var resItems []database.Item
	err := json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)
	require.Equal(t, 2, len(resItems))
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
	file, err := os.ReadFile(".." + resItems[1].Image)
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

}

// Set MaxOrderAmount to avoid errors
func setMaxOrderAmount(t *testing.T, amount int) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("MaxOrderAmount", strconv.Itoa(amount))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, writer.FormDataContentType(), 200, adminUserToken)
}

func CreateTestItemWithLicense(t *testing.T) (string, string) {
	f := `{
		"Name": "License item",
		"Price": 123
	}`
	res := utils.TestRequestStrWithAuth(t, r, "POST", "/api/items/", f, 200, adminUserToken)
	licenseItemID := res.Body.String()

	f2 := `{
		"Name": "Test item",
		"Price": 314,
		"LicenseItem": ` + licenseItemID + `
	}`
	res2 := utils.TestRequestStrWithAuth(t, r, "POST", "/api/items/", f2, 200, adminUserToken)
	itemID := res2.Body.String()
	return itemID, licenseItemID
}

// TestOrders tests CRUD operations on orders
// TODO: Test independent of vivawallet
func TestOrders(t *testing.T) {
	setMaxOrderAmount(t, 10)

	itemID, licenseItemID := CreateTestItemWithLicense(t)
	itemIDInt, _ := strconv.Atoi(itemID)
	licenseItemIDInt, _ := strconv.Atoi(licenseItemID)
	vendorID := createTestVendor(t, "testLicenseID2")
	vendorIDInt, _ := strconv.Atoi(vendorID)
	f := `{
		"entries": [
			{
			  "item": ` + itemID + `,
			  "quantity": 315
			}
		  ],
		  "vendorLicenseID": "testLicenseID2"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", f, 400)
	require.Equal(t, res.Body.String(), `{"error":{"message":"Order amount is too high"}}`)

	setMaxOrderAmount(t, 1000000)
	res = utils.TestRequestStr(t, r, "POST", "/api/orders/", f, 200)
	require.Equal(t, res.Body.String(), `{"SmartCheckoutURL":"`+config.Config.VivaWalletSmartCheckoutURL+`0"}`)

	order, err := database.Db.GetOrderByOrderCode("0")
	if err != nil {
		t.Error(err)
	}

	// Test order amount
	orderTotal := order.GetTotal()
	require.Equal(t, orderTotal, 314*315)

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
	require.Equal(t, order.Entries[0].Item, licenseItemIDInt)
	require.Equal(t, order.Entries[1].Item, itemIDInt)
	require.Equal(t, order.Entries[1].Quantity, 315)
	require.Equal(t, order.Entries[1].Price, 314)
	require.Equal(t, order.Entries[1].Sender, senderAccount.ID)
	require.Equal(t, order.Entries[1].Receiver, receiverAccount.ID)

}

// TestPayments tests CRUD operations on payments
func TestPayments(t *testing.T) {
	database.Db.InitEmptyTestDb()

	// Set up a payment account
	senderAccountID, err := database.Db.CreateAccount(
		database.Account{Name: "Test account"},
	)
	receiverAccountID, err := database.Db.CreateAccount(
		database.Account{Name: "Test account"},
	)
	utils.CheckError(t, err)

	// Create payments via API
	database.Db.CreatePayment(
		database.Payment{
			Sender:   senderAccountID,
			Receiver: receiverAccountID,
			Amount:   314,
		})
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
	require.Equal(t, payments[0].Sender, senderAccountID)
	require.Equal(t, payments[0].Receiver, receiverAccountID)
	require.Equal(t, payments[0].Timestamp.Day(), time.Now().Day())
	require.Equal(t, payments[0].Timestamp.Hour(), time.Now().UTC().Hour())

	// Test account balances
	senderAccount, err := database.Db.GetAccountByID(senderAccountID)
	utils.CheckError(t, err)
	receiverAccount, err := database.Db.GetAccountByID(receiverAccountID)
	utils.CheckError(t, err)
	require.Equal(t, senderAccount.Balance, -314)
	require.Equal(t, receiverAccount.Balance, 314)

	// Test time filters
	timeRequest(t, 0, 0, 1)
	timeRequest(t, -1, 1, 1)
	timeRequest(t, -2, -1, 0)
	timeRequest(t, 1, -1, 0)
	timeRequest(t, 1, 0, 0)
	timeRequest(t, 0, 1, 1)
	timeRequest(t, -1, 0, 1)
	timeRequest(t, 0, -1, 0)

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

	vendorID := createTestVendor(t, "testLicenseID")
	vendorIDInt, _ := strconv.Atoi(vendorID)

	// Create invalid payments via API
	f := createPaymentPayoutRequest{
		Amount:          -314,
		VendorLicenseID: "testLicenseID",
	}
	res := utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 400, adminUserToken)
	require.Equal(t, res.Body.String(), `{"error":{"message":"payout amount must be bigger than 0"}}`)

	// Create invalid payments via API
	f = createPaymentPayoutRequest{
		Amount:          0,
		VendorLicenseID: "testLicenseID",
	}
	res = utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 400, adminUserToken)
	require.Equal(t, res.Body.String(), `{"error":{"message":"payout amount must be bigger than 0"}}`)

	// Create payments via API
	f = createPaymentPayoutRequest{
		Amount:          314,
		VendorLicenseID: "testLicenseID",
	}
	res = utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 400, adminUserToken)
	require.Equal(t, res.Body.String(), `{"error":{"message":"payout amount bigger than vendor account balance"}}`)

	account, err := database.Db.GetAccountByVendorID(vendorIDInt)
	utils.CheckError(t, err)

	err = database.Db.UpdateAccountBalance(account.ID, 1000)
	utils.CheckError(t, err)

	res = utils.TestRequestWithAuth(t, r, "POST", "/api/payments/payout/", f, 200, adminUserToken)

	paymentID := res.Body.String()
	paymentIDInt, err := strconv.Atoi(paymentID)

	payment, err := database.Db.GetPayment(paymentIDInt)
	cashAccount, err := database.Db.GetAccountByType("Cash")

	require.Equal(t, payment.Amount, 314)
	require.Equal(t, payment.Sender, account.ID)
	require.Equal(t, payment.Receiver, cashAccount.ID)

	vendor, err := database.Db.GetVendorByLicenseID("testLicenseID")
	utils.CheckError(t, err)

	require.Equal(t, vendor.Balance, 686)
	require.Equal(t, cashAccount.Balance, 314)
	require.Equal(t, vendor.LastPayout.Time.Day(), time.Now().Day())
	require.Equal(t, vendor.LastPayout.Time.Hour(), time.Now().Hour())

}

// TestSettings tests GET and PUT operations on settings
func TestSettings(t *testing.T) {

	// Update (multipart form!)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
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

	// Check file
	file, err := os.ReadFile("../" + settings.Logo)
	utils.CheckError(t, err)
	require.Equal(t, `i am the content of a jpg file :D`, string(file))
}

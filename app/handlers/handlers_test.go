package handlers

import (
	"augustin/database"
	"augustin/utils"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

var r *chi.Mux

// TestMain is executed before all tests and initializes an empty database
func TestMain(m *testing.M) {
	database.Db.InitEmptyTestDb()
	r = GetRouter()
	os.Exit(m.Run())
}

// TestHelloWorld tests the hello world test function
func TestHelloWorld(t *testing.T) {
	res := utils.TestRequest(t, r, "GET", "/api/hello/", nil, 200)
	require.Equal(t, "\"Hello, world!\"", res.Body.String())
}

func CreateTestVendor(t *testing.T) string {
	json_vendor := `{
		"keycloakID": "test",
		"urlID": "test",
		"licenseID": "test",
		"firstName": "test1234",
		"lastName": "test"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/vendors/", json_vendor, 200)
	vendorID := res.Body.String()
	return vendorID
}

// TestVendors tests CRUD operations on users
func TestVendors(t *testing.T) {
	// Create
	vendorID := CreateTestVendor(t)
	res := utils.TestRequest(t, r, "GET", "/api/vendors/", nil, 200)
	var vendors []database.Vendor
	err := json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))
	require.Equal(t, "test1234", vendors[0].FirstName)

	// Update
	json_vendor := `{"firstName": "nameAfterUpdate"}`
	utils.TestRequestStr(t, r, "PUT", "/api/vendors/"+vendorID+"/", json_vendor, 200)
	res = utils.TestRequest(t, r, "GET", "/api/vendors/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))
	require.Equal(t, "nameAfterUpdate", vendors[0].FirstName)

	// Delete
	utils.TestRequest(t, r, "DELETE", "/api/vendors/"+vendorID+"/", nil, 204)
	res = utils.TestRequest(t, r, "GET", "/api/vendors/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(vendors))
}

func CreateTestItem(t *testing.T) string {
	f := `{
		"Name": "Test item",
		"Price": 314
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/items/", f, 200)
	itemID := res.Body.String()
	return itemID
}

// TestItems tests CRUD operations on items (including images)
// Todo: delete file after test
func TestItems(t *testing.T) {

	// Create
	itemID := CreateTestItem(t)

	// Read
	res := utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	var resItems []database.Item
	err := json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(resItems))
	require.Equal(t, "Test item", resItems[0].Name)

	// Update (multipart form!)
	body := new(bytes.Buffer)
    writer := multipart.NewWriter(body)
    writer.WriteField("Name", "Updated item name")
    writer.WriteField("nonexistingfieldname", "10")
    image, _ := writer.CreateFormFile("Image", "test.jpg")
    image.Write([]byte(`i am the content of a jpg file :D`))
    writer.Close()
	utils.TestRequestMultiPart(t, r, "PUT", "/api/items/"+itemID+"/", body, writer.FormDataContentType(), 200)

	// Read
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(resItems))
	require.Equal(t, "Updated item name", resItems[0].Name)
	require.Contains(t, resItems[0].Image, "test")
	require.Contains(t, resItems[0].Image, ".jpg")

	// Check file
	file, err := os.ReadFile(".." + resItems[0].Image)
	utils.CheckError(t, err)
	require.Equal(t, `i am the content of a jpg file :D`, string(file))

	// Update with image as field (not as a file)
	body = new(bytes.Buffer)
    writer = multipart.NewWriter(body)
	writer.WriteField("Name", "Updated item name 2")
	writer.WriteField("Image", "Test")
    writer.Close()
	utils.TestRequestMultiPart(t, r, "PUT", "/api/items/"+itemID+"/", body, writer.FormDataContentType(), 200)

	// Read
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &resItems)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(resItems))
	require.Equal(t, "Updated item name 2", resItems[0].Name)
	require.Equal(t, resItems[0].Image, "Test")

}

// TestOrders tests CRUD operations on orders
// TODO: Test independent of vivawallet
func TestOrders(t *testing.T) {

	itemID := CreateTestItem(t)
	vendorID := CreateTestVendor(t)
	f := `{
		"entries": [
			{
			  "item": ` + itemID + `,
			  "quantity": 1
			}
		  ],
		  "vendor": ` + vendorID + `
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", f, 200)
	ress := res.Body.String()
	log.Info(ress)
}

// TestPayments tests CRUD operations on payments
func TestPayments(t *testing.T) {

	// Set up a payment account
	account_id, err := database.Db.CreateAccount(
		database.Account{Name: "Test account"},
	)
	utils.CheckError(t, err)

	// Create payments via API
	f := CreatePaymentsRequest{
		Payments: []database.Payment{
			{
				Sender:   account_id,
				Receiver: account_id,
				Amount:   314,
			},
		},
	}
	utils.TestRequest(t, r, "POST", "/api/payments/", f, 200)
	response2 := utils.TestRequest(t, r, "GET", "/api/payments/", nil, 200)

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
	require.Equal(t, payments[0].Sender, account_id)
	require.Equal(t, payments[0].Receiver, account_id)
	require.Equal(t, payments[0].Timestamp.Day(), time.Now().Day())
	require.Equal(t, payments[0].Timestamp.Hour(), time.Now().UTC().Hour())

}

// TestSettings tests GET and PUT operations on settings
func TestSettings(t *testing.T) {
	utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
}

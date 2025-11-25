package handlers

import (
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

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"

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
		panic(err)
	}
	log.Infof("TestMain: Created admin user %s \n", adminUserEmail)
	err = keycloak.KeycloakClient.AssignRole(adminUser, "backoffice")
	if err != nil {
		log.Errorf("TestMain: Assign backoffice role to user %s failed: %v \n", adminUser, err)
		panic(err)
	}
	log.Infof("TestMain: Assigned backoffice role to user %s \n", adminUserEmail)
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
	vendorPassword := "password"
	err := keycloak.KeycloakClient.DeleteUser(vendorEmail)
	if err != nil {
		log.Infof("Delete user %v failed, which is okey: %v \n", vendorLicenseId, err)
	}

	// Initialize database and empty it
	err = database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create
	vendorID := createTestVendor(t, vendorLicenseId)
	keycloak.KeycloakClient.UpdateUserPassword(vendorEmail, vendorPassword)

	// Query ListVendors only returns few fields (not all) under /api/vendors/
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	var vendors []database.Vendor
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))
	require.Equal(t, "test1234", vendors[0].FirstName)
	require.Equal(t, vendorLicenseId, vendors[0].LicenseID.String)
	require.Equal(t, "test", vendors[0].LastName)

	// Check if licenseID exists and returns first name of vendor
	res = utils.TestRequest(t, r, "GET", "/api/vendors/check/"+vendorLicenseId+"/", nil, 200)
	require.Equal(t, res.Body.String(), `{"FirstName":"test1234","AccountProofUrl":""}`)

	// GetVendorByID returns all fields under /api/vendors/{id}/
	utils.TestRequest(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 401)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 200, adminUserToken)
	var vendor database.Vendor
	err = json.Unmarshal(res.Body.Bytes(), &vendor)
	utils.CheckError(t, err)
	require.Equal(t, "test1234", vendor.FirstName)
	require.Equal(t, "+43123456789", vendor.Telephone)
	require.Equal(t, "1/22", vendor.VendorSince)
	require.Equal(t, vendorEmail, vendor.Email)

	// Update
	var vendors2 []database.Vendor
	jsonVendor := `{"firstName": "nameAfterUpdate", "licenseID": "IDAfterUpdate", "email": "` + vendorEmail + `", "Longitude": 16.363449, "Latitude": 48.210033}`
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/", jsonVendor, 200, adminUserToken)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	err = json.Unmarshal(res.Body.Bytes(), &vendors2)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors2))
	require.Equal(t, "nameAfterUpdate", vendors2[0].FirstName)

	// Test location data
	// var mapData []database.LocationData
	// res = utils.TestRequestWithAuth(t, r, "GET", "/api/map/", nil, 200, adminUserToken)
	// err = json.Unmarshal(res.Body.Bytes(), &mapData)
	// utils.CheckError(t, err)
	// require.Equal(t, 1, len(mapData))
	// require.Equal(t, 16.363449, mapData[0].Longitude)
	// require.Equal(t, 48.210033, mapData[0].Latitude)

	// Delete
	utils.TestRequestWithAuth(t, r, "DELETE", "/api/vendors/"+vendorID+"/", nil, 204, adminUserToken)
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/", nil, 200, adminUserToken)
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(vendors))

	// Clean up after test
	keycloak.KeycloakClient.DeleteUser(vendorEmail)

}

func CreateTestItem(t *testing.T, name string, price int, licenseItemID string, licenseGroup string) string {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", name)
	// Provide a description that satisfies ent schema validators
	writer.WriteField("Description", "Automated test item description")
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
	// Include Description when updating to satisfy ent validators
	writer.WriteField("Description", "Updated description")
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
	// Include Description when updating to satisfy ent validators
	writer.WriteField("Description", "Updated description 2")
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

	require.Equal(t, res.Body.String(), `{"error":{"message":"nice try! You are not allowed to update this item"}}`)

}

// TestLicenseUnassignedWhenItemDeleted ensures that when an item that
// references a digital license is deleted (archived), the license becomes
// unassigned (no item references it any more).
func TestLicenseUnassignedWhenItemDeleted(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Initialize database and empty it
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// Create a license item (IsLicenseItem = true)
	licenseItemID := CreateTestItem(t, "License item", 3, "", "")
	// Create a regular item that references the license
	itemID := CreateTestItem(t, "Test item", 20, licenseItemID, "testedition")

	licenseInt, err := strconv.Atoi(licenseItemID)
	require.NoError(t, err)
	itemInt, err := strconv.Atoi(itemID)
	require.NoError(t, err)

	// Delete (archive) the item
	err = database.Db.DeleteItem(itemInt)
	require.NoError(t, err)

	// After deletion, the license should no longer be referenced by any item
	_, found, err := database.Db.GetItemByLicenseID(licenseInt)
	require.NoError(t, err)
	require.False(t, found, "expected license to be unassigned after deleting the item")
}

// Set MaxOrderAmount to avoid errors
func setMaxOrderAmount(t *testing.T, amount int) {
	// Update settings directly in DB to avoid relying on the HTTP handler
	settings, err := database.Db.GetSettings()
	utils.CheckError(t, err)
	if settings == nil {
		t.Fatal("settings not found")
	}
	settings.MaxOrderAmount = amount
	err = database.Db.UpdateSettings(settings)
	utils.CheckError(t, err)

	// Verify
	settings2, err := database.Db.GetSettings()
	utils.CheckError(t, err)
	require.Equal(t, amount, settings2.MaxOrderAmount)
}

func CreateTestItemWithLicense(t *testing.T) (string, string) {
	licenseItemID := CreateTestItem(t, "License item", 3, "", "")
	itemID := CreateTestItem(t, "Test item", 20, licenseItemID, "testedition")
	return itemID, licenseItemID
}

// TestUpdateItemHandlerWithLicense covers updating an item via the HTTP
// handler where the item initially has a LicenseItem: switch to another
// license and then clear the license by omitting the LicenseItem field.
func TestUpdateItemHandlerWithLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// ensure fresh DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// create two license items directly via DB (include Description to satisfy validators)
	licA := database.Item{Name: "license-A", Description: "License A description", Price: 10, IsLicenseItem: true}
	licAID, err := database.Db.CreateItem(licA)
	require.NoError(t, err)
	require.True(t, licAID > 0)
	licB := database.Item{Name: "license-B", Description: "License B description", Price: 20, IsLicenseItem: true}
	licBID, err := database.Db.CreateItem(licB)
	require.NoError(t, err)
	require.True(t, licBID > 0)

	// create an item that references licA also directly via DB
	item := database.Item{Name: "item-with-license", Description: "Item that requires a license", Price: 100, LicenseItem: null.IntFrom(int64(licAID))}
	itemIDInt, err := database.Db.CreateItem(item)
	require.NoError(t, err)
	require.True(t, itemIDInt > 0)
	itemID := strconv.Itoa(itemIDInt)
	licenseAInt := licAID
	licenseBInt := licBID

	// fetch items and verify initial license is A
	res := utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	var items []database.Item
	err = json.Unmarshal(res.Body.Bytes(), &items)
	utils.CheckError(t, err)

	var found database.Item
	for _, it := range items {
		if it.ID == itemIDInt {
			found = it
			break
		}
	}
	require.NotZero(t, found.ID)
	require.True(t, found.LicenseItem.Valid)
	require.Equal(t, int64(licenseAInt), found.LicenseItem.ValueOrZero())

	// update via handler: point to licenseB
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", "item-with-license-updated")
	writer.WriteField("Description", "Updated description")
	writer.WriteField("Price", strconv.Itoa(100))
	writer.WriteField("LicenseItem", strconv.Itoa(licenseBInt))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+itemID+"/", body, writer.FormDataContentType(), 200, adminUserToken)

	// fetch and assert license is now B
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &items)
	utils.CheckError(t, err)
	var found2 database.Item
	for _, it := range items {
		if it.ID == itemIDInt {
			found2 = it
			break
		}
	}
	require.NotZero(t, found2.ID)
	require.True(t, found2.LicenseItem.Valid)
	require.Equal(t, int64(licenseBInt), found2.LicenseItem.ValueOrZero())
	require.Equal(t, "item-with-license-updated", found2.Name)

	// update again but omit LicenseItem to clear it
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "item-without-license")
	writer.WriteField("Description", "Still has description")
	writer.WriteField("Price", strconv.Itoa(100))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+itemID+"/", body, writer.FormDataContentType(), 200, adminUserToken)

	// fetch and assert license cleared
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &items)
	utils.CheckError(t, err)
	var found3 database.Item
	for _, it := range items {
		if it.ID == itemIDInt {
			found3 = it
			break
		}
	}
	require.NotZero(t, found3.ID)
	require.False(t, found3.LicenseItem.Valid)
	require.Equal(t, "item-without-license", found3.Name)
}

// TestCreateItemsWithAndWithoutPDFAndLicense tests creating items via the
// HTTP handler: first without a PDF, then with a PDF upload, and finally
// creating a license item.
func TestCreateItemsWithAndWithoutPDFAndLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// Ensure clean DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// 1) Create item without PDF
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", "no-pdf-item")
	writer.WriteField("Description", "An item without pdf")
	writer.WriteField("Price", strconv.Itoa(123))
	writer.Close()
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	idNoPdf := strings.TrimSpace(res.Body.String())
	require.NotEmpty(t, idNoPdf)
	// idNoPdfInt parsed when needed

	// Verify created item has no PDF
	res = utils.TestRequest(t, r, "GET", "/api/items/", nil, 200)
	var items []database.Item
	err = json.Unmarshal(res.Body.Bytes(), &items)
	utils.CheckError(t, err)
	found := false
	for _, it := range items {
		if it.Name == "no-pdf-item" {
			found = true
			require.False(t, it.PDF.Valid)
			break
		}
	}
	require.True(t, found)

	// 2) Create item with PDF upload
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "with-pdf-item")
	writer.WriteField("Description", "An item with pdf")
	writer.WriteField("Price", strconv.Itoa(200))
	// attach a small pdf file content
	fw, _ := writer.CreateFormFile("PDF", "test.pdf")
	fw.Write([]byte("%PDF-1.4 fake pdf content"))
	writer.Close()
	res = utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	idWithPdf := strings.TrimSpace(res.Body.String())
	require.NotEmpty(t, idWithPdf)

	// Verify created item has a PDF and the file exists (query directly by id)
	idWithPdfInt, _ := strconv.Atoi(idWithPdf)
	dbItem, err := database.Db.GetItem(idWithPdfInt)
	require.NoError(t, err)
	require.True(t, dbItem.PDF.Valid)
	pdfRow, err := database.Db.GetPDFByID(dbItem.PDF.ValueOrZero())
	require.NoError(t, err)
	require.NotEmpty(t, pdfRow.Path)
	// check file exists on disk (try cwd prefix if necessary)
	_, err = os.Stat(pdfRow.Path)
	if os.IsNotExist(err) {
		dir, _ := os.Getwd()
		_, err2 := os.Stat(dir + "/" + pdfRow.Path)
		require.NoError(t, err2)
	} else {
		require.NoError(t, err)
	}

	// 3) Create a license item via handler
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "license-item-handler")
	writer.WriteField("Description", "License via handler")
	writer.WriteField("Price", strconv.Itoa(5))
	writer.WriteField("IsLicenseItem", "true")
	writer.Close()
	res = utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	idLicense := strings.TrimSpace(res.Body.String())
	require.NotEmpty(t, idLicense)

	// Verify created license item has IsLicenseItem true by querying DB directly
	dbLic, err := database.Db.GetItemByName("license-item-handler")
	require.NoError(t, err)
	require.True(t, dbLic.IsLicenseItem)
}

// TestUpdateItemLicenseConflict reproduces the ent unique constraint when two
// different items try to reference the same license item. It creates a
// license item, assigns it to itemA, then attempts to update itemB via the
// HTTP handler to use the same license and expects a 400 with the ent
// constraint error message.
func TestUpdateItemLicenseConflict(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// fresh DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// create the license item (IsLicenseItem = true)
	license := database.Item{Name: "conflict-license", Description: "conflict license", Price: 1, IsLicenseItem: true}
	licenseID, err := database.Db.CreateItem(license)
	require.NoError(t, err)
	require.True(t, licenseID > 0)

	// create itemA that already references the license
	itemA := database.Item{Name: "item-A", Description: "already has license", Price: 10, LicenseItem: null.IntFrom(int64(licenseID))}
	itemAID, err := database.Db.CreateItem(itemA)
	require.NoError(t, err)
	require.True(t, itemAID > 0)

	// create itemB which we'll try to update via handler
	itemB := database.Item{Name: "item-B", Description: "to be updated", Price: 20}
	itemBID, err := database.Db.CreateItem(itemB)
	require.NoError(t, err)
	require.True(t, itemBID > 0)

	// Prepare multipart form to update itemB and set LicenseItem to licenseID
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("ID", strconv.Itoa(itemBID))
	writer.WriteField("Archived", "false")
	writer.WriteField("Disabled", "false")
	writer.WriteField("Description", "Digitale Zeitungsausgabe")
	writer.WriteField("Name", "Digitale Zeitung")
	writer.WriteField("Image", "img/Digital_0_2.jpg")
	writer.WriteField("IsLicenseItem", "false")
	writer.WriteField("IsPDFItem", "false")
	writer.WriteField("ItemOrder", "0")
	writer.WriteField("LicenseGroup", "testedition")
	writer.WriteField("LicenseItem", strconv.Itoa(licenseID))
	writer.WriteField("Price", strconv.Itoa(300))
	writer.Close()

	res := utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+strconv.Itoa(itemBID)+"/", body, writer.FormDataContentType(), 400, adminUserToken)

	// The backend should return a friendly error when the license is already assigned
	require.Contains(t, res.Body.String(), "license item is already assigned")
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
	url := respMap["SmartCheckoutURL"]
	// In test mode we may get either the real Viva wallet URL or a local test URL.
	// Assert it's a URL that indicates a checkout/success redirect.
	require.True(t, strings.Contains(url, "checkout") || strings.Contains(url, "success"))

	// Check if order was created
	orders, _ = database.Db.GetOrders()
	require.Equal(t, 1, len(orders))

	// Use the created order rather than relying on a fixed order code
	order := orders[0]
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

	resSuccess := utils.TestRequestStr(t, r, "GET", "/api/orders/verify/?s=0&t=0", "", 200)
	require.Equal(t, strings.Contains(resSuccess.Body.String(), vendorLicenseId), true)

	// Wait for the background verification to finish and check its error
	e := <-errc
	utils.CheckError(t, e)
	// Check payments
	payments, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false)
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

	itemID, err := database.Db.CreateItem(database.Item{Name: "Test item for payments", Description: "Payment test item description", Price: 314})
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
	require.Equal(t, payments[0].Timestamp.Day(), time.Now().Day())
	require.Equal(t, payments[0].Timestamp.Hour(), time.Now().UTC().Hour())

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

	// Clean up
	database.Db.DeletePayment(p1)
	database.Db.DeletePayment(p2)
	database.Db.DeleteItem(itemID)
}

// TestVerifyOrder_EmailSentOnlyOnce ensures that license/pdf emails are only
// sent once even if VerifyOrderAndCreatePayments is called multiple times.
func TestVerifyOrder_EmailSentOnlyOnce(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	// fresh DB
	err := database.Db.InitEmptyTestDb()
	if err != nil {
		panic(err)
	}

	// create vendor
	vendorLicenseId := "testverifyemail"
	_ = createTestVendor(t, vendorLicenseId)

	// create a PDF resource
	pdf := database.PDF{Path: "testfile.pdf", Timestamp: time.Now()}
	pdfID, err := database.Db.CreatePDF(pdf)
	require.NoError(t, err)

	// create a license item that is a PDF
	licenseItem := database.Item{Name: "license-pdf", Description: "license pdf", Price: 1, IsLicenseItem: true, IsPDFItem: true, PDF: null.IntFrom(pdfID)}
	licenseID, err := database.Db.CreateItem(licenseItem)
	require.NoError(t, err)

	// create a normal item that references the license
	item := database.Item{Name: "item-with-license-pdf", Description: "main item", Price: 200, LicenseItem: null.IntFrom(int64(licenseID)), LicenseGroup: null.StringFrom("pdfgroup")}
	itemID, err := database.Db.CreateItem(item)
	require.NoError(t, err)

	// allow sufficiently large order amounts
	setMaxOrderAmount(t, 5000)

	// Ensure mail templates exist so emails can be built during verification
	err = database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Your license", "Please access your license at {{.URL}}")
	utils.CheckError(t, err)
	err = database.Db.CreateOrUpdateMailTemplate("PDFLicenceItemTemplate.html", "Your PDF", "Download your PDF at {{.URL}}")
	utils.CheckError(t, err)

	customerEmail := "verify_once@example.com"

	// create order via API (this will prepend license entry internally)
	requestWithEmail := `{
		"entries": [
			{ "item": ` + strconv.Itoa(itemID) + `, "quantity": 1 }
		],
		"vendorLicenseID": "` + vendorLicenseId + `",
		"customerEmail": "` + customerEmail + `"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", requestWithEmail, 200)
	var respMap map[string]string
	err = json.Unmarshal(res.Body.Bytes(), &respMap)
	utils.CheckError(t, err)
	url := respMap["SmartCheckoutURL"]
	require.NotEmpty(t, url)
	// In test mode we may get either the real Viva wallet URL or a local test URL.
	// Assert it's a URL that indicates a checkout/success redirect.
	require.True(t, strings.Contains(url, "checkout") || strings.Contains(url, "success"))

	// fetch order: extract OrderCode from SmartCheckoutURL rather than assuming "0"
	orderCodeStr := "0"
	if idx := strings.Index(url, "s="); idx != -1 {
		substr := url[idx+2:]
		if j := strings.Index(substr, "&"); j != -1 {
			substr = substr[:j]
		}
		orderCodeStr = substr
	}
	order, err := database.Db.GetOrderByOrderCode(orderCodeStr)
	require.NoError(t, err)

	// call verify twice
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
	require.NoError(t, err)
	// second call should not resend emails or duplicate payments
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, 48)
	require.NoError(t, err)

	// check payments: should only create payments once (license + item)
	payments, err := database.Db.ListPayments(time.Time{}, time.Time{}, "", false, false, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(payments))

	// check pdf downloads for order: only one download should exist and EmailSent true
	pdfDownloads, err := database.Db.GetPDFDownloadByOrderId(order.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(pdfDownloads))
	require.True(t, pdfDownloads[0].EmailSent)
	require.EqualValues(t, licenseID, pdfDownloads[0].ItemID.ValueOrZero())
	require.EqualValues(t, order.ID, pdfDownloads[0].OrderID.ValueOrZero())

	// cleanup
	for _, payment := range payments {
		database.Db.DeletePayment(payment.ID)
	}
	for _, entry := range order.Entries {
		database.Db.DeleteOrderEntry(entry.ID)
	}
	database.Db.DeleteOrder(order.ID)
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
	var payoutResp paymentsResponse
	err = json.Unmarshal(res.Body.Bytes(), &payoutResp)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(payoutResp.Payments))

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
	var exSettings ExtendedSettings
	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	err := json.Unmarshal(res.Body.Bytes(), &exSettings)
	utils.CheckError(t, err)
	require.Equal(t, "/img/logo.png", exSettings.Settings.Logo)
	require.Equal(t, 10, exSettings.Settings.MaxOrderAmount)

	// Check item join
	require.Equal(t, "Test main item", exSettings.Settings.Edges.MainItem.Name)
	// Price is a float in ent Item
	require.Equal(t, float64(314), exSettings.Settings.Edges.MainItem.Price)

	// Check file
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	file, err := os.ReadFile(dir + "/" + exSettings.Settings.Logo)
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
	require.Equal(t, vendorEmail, meVendor.Email)
	require.Equal(t, 314, meVendor.Balance)
	require.Equal(t, 1, len(meVendor.OpenPayments))

	// Test if vendor can't see other vendors
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 403, vendorToken)

	// Test if admin who is no vendor can't see vendor overview
	// (middleware returns 403 Forbidden now)
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/me/", nil, 403, adminUserToken)

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

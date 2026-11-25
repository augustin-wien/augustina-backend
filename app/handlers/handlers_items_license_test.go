package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

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
	licA := database.Item{Name: "license-A", Description: "License A description", Price: 10, IsLicenseItem: true, Type: "license_item"}
	licAID, err := database.Db.CreateItem(licA)
	require.NoError(t, err)
	require.True(t, licAID > 0)
	licB := database.Item{Name: "license-B", Description: "License B description", Price: 20, IsLicenseItem: true, Type: "license_item"}
	licBID, err := database.Db.CreateItem(licB)
	require.NoError(t, err)
	require.True(t, licBID > 0)

	// create an item that references licA also directly via DB
	item := database.Item{Name: "item-with-license", Description: "Item that requires a license", Price: 100, LicenseItem: null.IntFrom(int64(licAID)), Type: "normal_item"}
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

	// 4) Create online_issue and ensure license item is auto-created
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "online-issue-handler")
	writer.WriteField("Description", "Online issue via handler")
	writer.WriteField("Price", strconv.Itoa(700))
	writer.WriteField("Type", "online_issue")
	writer.WriteField("license_cost", strconv.Itoa(150))
	writer.WriteField("LicenseGroup", "digital_edition")
	writer.Close()
	res = utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	idOnline := strings.TrimSpace(res.Body.String())
	require.NotEmpty(t, idOnline)

	idOnlineInt, err := strconv.Atoi(idOnline)
	require.NoError(t, err)

	dbOnline, err := database.Db.GetItemIncludingDisabled(idOnlineInt)
	require.NoError(t, err)
	require.Equal(t, "online_issue", dbOnline.Type)
	require.True(t, dbOnline.Disabled)
	require.True(t, dbOnline.LicenseItem.Valid)

	autoLicenseID := int(dbOnline.LicenseItem.ValueOrZero())
	dbAutoLicense, err := database.Db.GetItemIncludingDisabled(autoLicenseID)
	require.NoError(t, err)
	require.Equal(t, "online-issue-handler license", dbAutoLicense.Name)
	require.Equal(t, 150, dbAutoLicense.Price)
	require.True(t, dbAutoLicense.IsLicenseItem)
	require.Equal(t, "license_item", dbAutoLicense.Type)
}

// TestCreateOnlineIssueCreatesLicenseItem verifies that POSTing an online_issue
// item via the HTTP handler automatically creates a companion license item and
// links it. It also checks that omitting or zeroing license_cost is rejected.
func TestCreateOnlineIssueCreatesLicenseItem(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	require.NoError(t, err)

	// 1) Missing license_cost → 400
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", "no-cost-issue")
	writer.WriteField("Description", "Missing license cost")
	writer.WriteField("Price", "500")
	writer.WriteField("Type", "online_issue")
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 400, adminUserToken)

	// 2) Zero license_cost → 400
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "zero-cost-issue")
	writer.WriteField("Description", "Zero license cost")
	writer.WriteField("Price", "500")
	writer.WriteField("Type", "online_issue")
	writer.WriteField("LicenseCost", "0")
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 400, adminUserToken)

	// 3) Missing license group → 400
	for _, licenseGroup := range []string{"", "   "} {
		body = new(bytes.Buffer)
		writer = multipart.NewWriter(body)
		writer.WriteField("Name", "no-group-issue")
		writer.WriteField("Description", "Missing license group")
		writer.WriteField("Price", "500")
		writer.WriteField("Type", "online_issue")
		writer.WriteField("LicenseCost", "150")
		writer.WriteField("LicenseGroup", licenseGroup)
		writer.Close()
		utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 400, adminUserToken)
	}

	// 4) Valid online_issue → 200; license item auto-created and linked
	body = new(bytes.Buffer)
	writer = multipart.NewWriter(body)
	writer.WriteField("Name", "test-online-issue")
	writer.WriteField("Description", "Online issue with license")
	writer.WriteField("Price", "700")
	writer.WriteField("Type", "online_issue")
	writer.WriteField("LicenseCost", "150")
	writer.WriteField("LicenseGroup", "testgroup")
	writer.Close()
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 200, adminUserToken)
	idOnline, err := strconv.Atoi(strings.TrimSpace(res.Body.String()))
	require.NoError(t, err)
	require.Positive(t, idOnline)

	// Online issue must start as disabled and reference a license item
	onlineIssue, err := database.Db.GetItemIncludingDisabled(idOnline)
	require.NoError(t, err)
	require.Equal(t, "online_issue", onlineIssue.Type)
	require.True(t, onlineIssue.Disabled, "online_issue should be created as disabled")
	require.True(t, onlineIssue.LicenseItem.Valid, "online_issue must have a linked license item")

	// Auto-created license item must have correct fields
	licenseID := int(onlineIssue.LicenseItem.ValueOrZero())
	licenseItem, err := database.Db.GetItemIncludingDisabled(licenseID)
	require.NoError(t, err)
	require.Equal(t, "test-online-issue license", licenseItem.Name)
	require.Equal(t, 150, licenseItem.Price)
	require.True(t, licenseItem.IsLicenseItem)
	require.Equal(t, "license_item", licenseItem.Type)

	// 5) Removing the license group of the online issue → 400, unless it is a PDF item
	updateOnlineIssue := func(isPDFItem string, status int) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("Name", "test-online-issue")
		writer.WriteField("Description", "Online issue with license")
		writer.WriteField("Price", "700")
		writer.WriteField("Type", "online_issue")
		writer.WriteField("IsPDFItem", isPDFItem)
		writer.WriteField("LicenseGroup", "")
		writer.Close()
		utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+strconv.Itoa(idOnline)+"/", body, writer.FormDataContentType(), status, adminUserToken)
	}
	updateOnlineIssue("false", 400)
	onlineIssue, err = database.Db.GetItemIncludingDisabled(idOnline)
	require.NoError(t, err)
	require.Equal(t, "testgroup", onlineIssue.LicenseGroup.String)
	updateOnlineIssue("true", 200)
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
	license := database.Item{Name: "conflict-license", Description: "conflict license", Price: 1, IsLicenseItem: true, Type: "license_item"}
	licenseID, err := database.Db.CreateItem(license)
	require.NoError(t, err)
	require.True(t, licenseID > 0)

	// create itemA that already references the license
	itemA := database.Item{Name: "item-A", Description: "already has license", Price: 10, LicenseItem: null.IntFrom(int64(licenseID)), Type: "normal_item"}
	itemAID, err := database.Db.CreateItem(itemA)
	require.NoError(t, err)
	require.True(t, itemAID > 0)

	// create itemB which we'll try to update via handler
	itemB := database.Item{Name: "item-B", Description: "to be updated", Price: 20, Type: "normal_item"}
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

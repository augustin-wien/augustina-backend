package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
)

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

// TestNotifyAbonementsOnlineIssue checks that publishing an online issue neither mails the
// abonnents nor grants it to them, and that the backoffice does both explicitly instead.
func TestNotifyAbonementsOnlineIssue(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	require.NoError(t, err)

	// One active abonement recipient.
	customer, err := database.Db.CreateCustomer(&database.Customer{
		KeycloakID: "online-issue-mail-customer",
		Email:      "online.issue.customer@example.com",
		FirstName:  "Online",
		LastName:   "Customer",
	})
	require.NoError(t, err)

	items, err := database.Db.ListItemsWithDisabled(false, false)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	_, err = database.Db.CreateAbonement(&database.Abonement{
		CustomerID: customer.ID,
		ItemID:     items[0].ID,
		FromDate:   time.Now().AddDate(0, 0, -1),
		ToDate:     time.Now().AddDate(0, 0, 7),
		Status:     "active",
	})
	require.NoError(t, err)

	onlineIssueID, err := database.Db.CreateItem(database.Item{
		Name:        "Issue To Publish",
		Description: "Issue description",
		Price:       100,
		Image:       "img/new-issue.jpg",
		Disabled:    true,
		Type:        "online_issue",
	})
	require.NoError(t, err)

	origBuild := database.BuildEmailRequestFromTemplate
	origSend := mailer.Send
	defer func() {
		database.BuildEmailRequestFromTemplate = origBuild
		mailer.Send = origSend
	}()

	config.Config.FrontendURL = "https://frontend.test"

	// The mails are sent in the background
	var mu sync.Mutex
	var sent []*mailer.EmailRequest
	var recipients [][]string
	mailCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(sent)
	}
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		mu.Lock()
		defer mu.Unlock()
		recipients = append(recipients, to)
		return nil, fmt.Errorf("template not found")
	}
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, r)
		return true, nil
	}

	notifyURL := "/api/items/" + strconv.Itoa(onlineIssueID) + "/notify-abonements/"
	groupPath := "/" + keycloak.KeycloakClient.CustomerGroup + "/" + keycloak.KeycloakClient.NewspaperGroup + "/9701"
	_ = keycloak.KeycloakClient.DeleteSubGroupByPath(groupPath)
	defer func() { _ = keycloak.KeycloakClient.DeleteSubGroupByPath(groupPath) }()

	// A disabled online issue can't be sent yet
	utils.TestRequestWithAuth(t, r, "POST", notifyURL, nil, 400, adminUserToken)

	// Publishing the online issue doesn't send anything by itself
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("Name", "Issue To Publish")
	writer.WriteField("Description", "Issue description")
	writer.WriteField("Price", strconv.Itoa(100))
	writer.WriteField("Type", "online_issue")
	writer.WriteField("LicenseGroup", "9701")
	writer.WriteField("Disabled", "false")
	writer.Close()

	utils.TestRequestMultiPartWithAuth(
		t,
		r,
		"PUT",
		"/api/items/"+strconv.Itoa(onlineIssueID)+"/",
		body,
		writer.FormDataContentType(),
		200,
		adminUserToken,
	)
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 0, mailCount())
	licenseGroups := func() []string {
		c, err := database.Db.GetCustomerByID(customer.ID)
		require.NoError(t, err)
		return c.LicenseGroups
	}
	require.NotContains(t, licenseGroups(), "9701")

	// Sending it explicitly grants the issue to the abonnent and mails them
	res := utils.TestRequestWithAuth(t, r, "POST", notifyURL, nil, 200, adminUserToken)
	var response NotifyAbonementsResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	require.Equal(t, 1, response.Recipients)

	require.Eventually(t, func() bool { return mailCount() == 1 }, 5*time.Second, 50*time.Millisecond)
	require.Contains(t, licenseGroups(), "9701")
	_, err = keycloak.KeycloakClient.GetGroupByPath(groupPath)
	require.NoError(t, err, "sending creates the issue's keycloak group")
	mu.Lock()
	mail := sent[0]
	require.Equal(t, [][]string{{"online.issue.customer@example.com"}}, recipients)
	mu.Unlock()
	require.Contains(t, mail.Body(), "<img")
	require.Contains(t, mail.Body(), "https://frontend.test/img/new-issue.jpg")

	// The delivery is stored for the issue
	require.Eventually(t, func() bool {
		res := utils.TestRequestWithAuth(t, r, "GET", notifyURL, nil, 200, adminUserToken)
		var status database.MailDeliveryStatus
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &status))
		return status.Total == 1 && status.Counts[database.MailTaskSent] == 1 && status.LastSentAt.Valid
	}, 5*time.Second, 50*time.Millisecond)

	// Sending again skips abonnents who already got the issue
	res = utils.TestRequestWithAuth(t, r, "POST", notifyURL, nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	require.Equal(t, 0, response.Recipients)
	require.Equal(t, 1, response.AlreadyMailed)
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 1, mailCount())

	// The mail is listed and can be sent again
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/mail-tasks/?kind=online_issue&item_id="+strconv.Itoa(onlineIssueID), nil, 200, adminUserToken)
	var list database.MailTaskList
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &list))
	require.Equal(t, 1, list.Total)
	require.Equal(t, []string{"online.issue.customer@example.com"}, list.Tasks[0].Recipients)
	taskURL := "/api/mail-tasks/" + strconv.Itoa(list.Tasks[0].ID) + "/"
	res = utils.TestRequestWithAuth(t, r, "GET", taskURL, nil, 200, adminUserToken)
	var task database.MailTask
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &task))
	require.Contains(t, task.Body, "<img")
	utils.TestRequestWithAuth(t, r, "POST", taskURL+"retry/", nil, 200, adminUserToken)
	require.Eventually(t, func() bool { return mailCount() == 2 }, 5*time.Second, 50*time.Millisecond)
	utils.TestRequestWithAuth(t, r, "GET", "/api/mail-tasks/?status=bogus", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/mail-tasks/999999/", nil, 404, adminUserToken)
	utils.TestRequest(t, r, "GET", "/api/mail-tasks/", nil, 401)

	// Only online issues can be sent
	utils.TestRequestWithAuth(t, r, "POST", "/api/items/"+strconv.Itoa(items[0].ID)+"/notify-abonements/", nil, 400, adminUserToken)
}

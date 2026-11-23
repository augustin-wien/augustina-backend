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

	"github.com/Nerzal/gocloak/v13"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
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

	// Global mailer mock to prevent real SMTP connections
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		return true, nil
	}

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

package handlers

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

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
	require.Equal(t, res.Body.String(), `{"FirstName":"test1234","AccountProofUrl":"","HasSteckbrief":false}`)

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

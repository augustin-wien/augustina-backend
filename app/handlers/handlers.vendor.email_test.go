package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func getTestVendor(t *testing.T, vendorID string) database.Vendor {
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/"+vendorID+"/", nil, 200, adminUserToken)
	var vendor database.Vendor
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &vendor))
	return vendor
}

func requireKeycloakEmail(t *testing.T, keycloakID, email string) {
	user, err := keycloak.KeycloakClient.GetUserByID(keycloakID)
	require.NoError(t, err)
	require.NotNil(t, user.Email)
	require.Equal(t, email, *user.Email)
}

// TestVendorInternalEmail covers vendors without a mailbox of their own: they
// get a generated internal address, can switch to their own one and back, and
// Keycloak follows every change
func TestVendorInternalEmail(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	// The license ID contains a space, which can't be part of an email address
	internalEmail := "intern-1" + settings.VendorEmailPostfix
	ownEmail := fmt.Sprintf("own_%d@example.org", time.Now().UnixNano())
	_ = keycloak.KeycloakClient.DeleteUser(internalEmail)
	defer func() {
		_ = keycloak.KeycloakClient.DeleteUser(internalEmail)
		_ = keycloak.KeycloakClient.DeleteUser(ownEmail)
	}()

	// Create without an email
	res := utils.TestRequestStrWithAuth(t, r, "POST", "/api/vendors/",
		`{"licenseID": "Intern 1", "firstName": "Ina", "lastName": "Intern", "email": ""}`, 200, adminUserToken)
	vendorID := res.Body.String()

	vendor := getTestVendor(t, vendorID)
	require.Equal(t, internalEmail, vendor.Email)
	require.False(t, vendor.HasOwnEmail)
	requireKeycloakEmail(t, vendor.KeycloakID, internalEmail)

	// Nobody reads the internal address, so no mails go there
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+vendorID+"/password-reset-email/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/"+vendorID+"/verify-email/", nil, 400, adminUserToken)

	// Switch to an own email
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/",
		`{"licenseID": "Intern 1", "firstName": "Ina", "lastName": "Intern", "email": "`+ownEmail+`"}`, 200, adminUserToken)
	vendor = getTestVendor(t, vendorID)
	require.Equal(t, ownEmail, vendor.Email)
	require.True(t, vendor.HasOwnEmail)
	requireKeycloakEmail(t, vendor.KeycloakID, ownEmail)

	// Back to the internal address
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/",
		`{"licenseID": "Intern 1", "firstName": "Ina", "lastName": "Intern", "email": ""}`, 200, adminUserToken)
	vendor = getTestVendor(t, vendorID)
	require.Equal(t, internalEmail, vendor.Email)
	require.False(t, vendor.HasOwnEmail)
	requireKeycloakEmail(t, vendor.KeycloakID, internalEmail)

	// Sending the generated address explicitly (as the CSV import does) is internal too
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/vendors/"+vendorID+"/",
		`{"licenseID": "Intern 1", "firstName": "Ina", "lastName": "Intern", "email": "`+strings.ToUpper("intern 1")+settings.VendorEmailPostfix+`"}`, 200, adminUserToken)
	require.False(t, getTestVendor(t, vendorID).HasOwnEmail)

	// Unknown vendor
	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/999999/password-reset-email/", nil, 404, adminUserToken)
	// Admins only
	utils.TestRequest(t, r, "POST", "/api/vendors/"+vendorID+"/verify-email/", nil, 401)
}

func TestCustomerActionEmails(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	utils.TestRequestWithAuth(t, r, "POST", "/api/customers/999999/password-reset-email/", nil, 404, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/customers/999999/verify-email/", nil, 404, adminUserToken)
	utils.TestRequest(t, r, "POST", "/api/customers/1/verify-email/", nil, 401)
}

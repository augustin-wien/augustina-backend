// Documentation see here: https://go-chi.io/#/pages/testing
package keycloak_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/handlers"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/Nerzal/gocloak/v13"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func lookupRole(roleName string, roles []*gocloak.Role) *gocloak.Role {
	for _, role := range roles {
		if *role.Name == roleName {
			return role
		}
	}
	return nil
}

func lookupGroup(groupName string, groups []*gocloak.Group) *gocloak.Group {
	for _, group := range groups {
		if *group.Name == groupName {
			return group
		}
	}
	return nil
}

func TestKeycloak(t *testing.T) {
	// Test the keycloak functions
	var err error
	// run tests in mainfolder
	err = os.Chdir("..")
	if err != nil {
		panic(err)
	}
	config.InitConfig()

	// Ensure tests don't trigger real SMTP sends via Keycloak.
	// The repo includes an `app/.env` with a sender address for local dev;
	// clear it here so the Keycloak helper will skip ExecuteActionsEmail.
	config.Config.SMTPSenderAddress = ""
	err = keycloak.InitializeOauthServer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		keycloak.KeycloakClient.DeleteGroup("testgroup")
		keycloak.KeycloakClient.DeleteUser("testuser@example.com")

	}()
	role_name := "testrole"
	err = keycloak.KeycloakClient.CreateRole(role_name)
	if err != nil {
		t.Error("TestKeycloak: Create role failed:", err)
	}
	role, err := keycloak.KeycloakClient.GetRole(role_name)
	if err != nil {
		t.Error("TestKeycloak: Get role failed:", err)
	}

	require.Equal(t, role_name, *role.Name)

	_, err = keycloak.KeycloakClient.CreateUser("testuser", "testuser", "testuser", "testuser@example.com", "password")
	if err != nil {
		log.Errorf("TestKeycloak: Create user failed: testuser@example.com %v \n", err)
	}

	user, err := keycloak.KeycloakClient.GetUser("testuser@example.com")
	if err != nil {
		t.Error("TestKeycloak: Get user failed:", err)
		panic(err)
	}

	err = keycloak.KeycloakClient.AssignRole(*user.ID, role_name)
	if err != nil {
		t.Error("TestKeycloak: Assign role failed:", err)
	}

	roles, err := keycloak.KeycloakClient.GetUserRoles(*user.ID)
	if err != nil {
		t.Error("TestKeycloak: Get user failed:", err)
	}
	require.NotNil(t, lookupRole(role_name, roles))

	err = keycloak.KeycloakClient.UnassignRole(*user.ID, role_name)
	if err != nil {
		t.Error("TestKeycloak: Unassign role failed:", err)
	}

	roles, err = keycloak.KeycloakClient.GetUserRoles(*user.ID)
	if err != nil {
		t.Error("TestKeycloak: Get user failed:", err)
	}
	require.Nil(t, lookupRole(role_name, roles))

	err = keycloak.KeycloakClient.DeleteUser(*user.Email)
	if err != nil {
		t.Error("TestKeycloak: Delete user failed:", err)
	}

	err = keycloak.KeycloakClient.DeleteRole(role_name)
	if err != nil {
		t.Error("TestKeycloak: Delete role failed:", err)
	}

	// Create Group
	groupName := "testgroup"
	err = keycloak.KeycloakClient.CreateGroup(groupName)
	if err != nil {
		t.Error("TestKeycloak: Create group failed:", err)
	}
	group, err := keycloak.KeycloakClient.GetGroup(groupName)
	if err != nil {
		t.Error("TestKeycloak: Get group failed:", err)
	}
	require.Equal(t, groupName, *group.Name)

	// Create Subgroup
	subGroupName := "testsubgroup"
	err = keycloak.KeycloakClient.CreateSubGroup(subGroupName, *group.ID)
	if err != nil {
		t.Error("TestKeycloak: Create subgroup failed:", err)
	}
	subGroupPath := "/" + groupName + "/" + subGroupName
	subGroup, err := keycloak.KeycloakClient.GetGroupByPath(subGroupPath)
	if err != nil {
		t.Error("TestKeycloak: Get subgroup failed:", err)
	}
	require.Equal(t, subGroupName, *subGroup.Name)

	// Get or Create User
	userName := "testuser"
	userEmail := userName + "@example.com"
	userID, _, err := keycloak.KeycloakClient.GetOrCreateUser(userEmail)
	if err != nil {
		t.Error("TestKeycloak: Get or create user failed:", err)
	}
	require.NotNil(t, userID)

	// Assign digital newspaper license to user
	err = keycloak.KeycloakClient.AssignDigitalLicenseGroup(userID, "keycloaktestedition")
	if err != nil {
		t.Error("TestKeycloak: Assign digital license group failed:", err)
	}

	// Get user groups
	groups, err := keycloak.KeycloakClient.GetUserGroups(userID)
	if err != nil {
		t.Error("TestKeycloak: Get user groups failed:", err)
	}
	require.NotNil(t, lookupGroup("keycloaktestedition", groups))

	// update user password
	err = keycloak.KeycloakClient.UpdateUserPassword(userEmail, "password")
	if err != nil {
		t.Error("TestKeycloak: Update user password failed:", err)
	}

	// login
	token, err := keycloak.KeycloakClient.GetUserToken(userEmail, "password")
	if err != nil {
		t.Error("TestKeycloak: Login failed:", err)
	}
	require.NotNil(t, token)

	// introspect token
	_, err = keycloak.KeycloakClient.IntrospectToken(token.AccessToken)
	if err != nil {
		t.Error("TestKeycloak: Introspect token failed:", err)
	}

	// get user info
	userInfo, err := keycloak.KeycloakClient.GetUserInfo(token.AccessToken)
	if err != nil {
		t.Error("TestKeycloak: Get user info failed:", err)
	}
	require.Equal(t, userEmail, *userInfo.Email)

	// get user by id
	user, err = keycloak.KeycloakClient.GetUserByID(userID)
	if err != nil {
		t.Error("TestKeycloak: Get user by id failed:", err)
	}
	require.Equal(t, userEmail, *user.Email)

	// update user
	err = keycloak.KeycloakClient.UpdateUser(userEmail, "testuser2", "testuser2", userEmail)
	if err != nil {
		t.Error("TestKeycloak: Update user failed:", err)
	}

	// Delete Subgroup
	err = keycloak.KeycloakClient.DeleteSubGroupByPath(subGroupPath)
	if err != nil {
		t.Error("TestKeycloak: Delete subgroup failed:", err)
	}

	// Delete Group
	err = keycloak.KeycloakClient.DeleteGroup(groupName)
	if err != nil {
		t.Error("TestKeycloak: Delete group failed:", err)
	}

	// Get realm roles
	roles, err = keycloak.KeycloakClient.GetRoles()
	if err != nil {
		t.Error("TestKeycloak: Get roles failed:", err)
	}
	require.NotNil(t, lookupRole("admin", roles))

	keycloak.KeycloakClient.DeleteUser("testuser@example.com")
	keycloak.KeycloakClient.DeleteSubGroupByPath("/customer/newspaper/keycloaktestedition")

}

func TestHelloWorldAuth(t *testing.T) {
	// Initialize database
	database.Db.InitEmptyTestDb()
	router := handlers.GetRouter()

	// Create a New Request
	req, _ := http.NewRequest("GET", "/api/auth/hello/", nil)

	// Execute Request
	response := utils.SubmitRequest(req, router)

	// Check the response code
	utils.CheckResponse(t, 401, response.Code)

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, "{\"error\":{\"message\":\"Unauthorized\"}}", response.Body.String())
}

// TestEnsureGroupPath verifies EnsureGroupPath creates nested groups and allows cleanup.
func TestEnsureGroupPath(t *testing.T) {
	// prepare environment
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	config.InitConfig()
	if err := keycloak.InitializeOauthServer(); err != nil {
		t.Fatalf("InitializeOauthServer failed: %v", err)
	}

	// unique group path to avoid collisions
	uniq := fmt.Sprintf("testgrp%d", time.Now().UnixNano())
	// create under vendor group to avoid relying on config fields
	path := "/" + keycloak.KeycloakClient.GetVendorGroup() + "/" + uniq

	// Ensure group path
	if err := keycloak.KeycloakClient.EnsureGroupPath(path); err != nil {
		t.Fatalf("EnsureGroupPath returned error: %v", err)
	}

	// verify group exists
	grp, err := keycloak.KeycloakClient.GetGroupByPath(path)
	if err != nil {
		t.Fatalf("GetGroupByPath failed: %v", err)
	}
	if grp == nil || grp.ID == nil {
		t.Fatalf("expected group at path %s, got nil", path)
	}

	// cleanup: delete deepest subgroup
	if err := keycloak.KeycloakClient.DeleteSubGroupByPath(path); err != nil {
		t.Fatalf("DeleteSubGroupByPath failed: %v", err)
	}
}

// TestAssignGroupNormalization verifies AssignGroup tolerates group names with or without leading slash.
func TestAssignGroupNormalization(t *testing.T) {
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	config.InitConfig()
	if err := keycloak.InitializeOauthServer(); err != nil {
		t.Fatalf("InitializeOauthServer failed: %v", err)
	}

	// create unique test user
	email := fmt.Sprintf("assign_test_%d@example.com", time.Now().UnixNano())
	// ensure user does not exist
	_ = keycloak.KeycloakClient.DeleteUser(email)
	uid, err := keycloak.KeycloakClient.CreateUser(email, "Test", "User", email, "password")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	defer func() {
		_ = keycloak.KeycloakClient.DeleteUser(email)
	}()

	// assign by name without leading slash
	if err := keycloak.KeycloakClient.AssignGroup(uid, keycloak.KeycloakClient.GetVendorGroup()); err != nil {
		t.Fatalf("AssignGroup failed: %v", err)
	}

	// assign by path with leading slash
	if err := keycloak.KeycloakClient.AssignGroup(uid, "/"+keycloak.KeycloakClient.GetVendorGroup()); err != nil {
		t.Fatalf("AssignGroup with leading slash failed: %v", err)
	}

	// verify user is in group (GetUserGroups will return groups)
	groups, err := keycloak.KeycloakClient.GetUserGroups(uid)
	if err != nil {
		t.Fatalf("GetUserGroups failed: %v", err)
	}
	if len(groups) == 0 {
		t.Fatalf("expected user to be assigned to at least one group, got 0")
	}
}

func TestSimpleErrorBranches(t *testing.T) {
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	config.InitConfig()
	if err := keycloak.InitializeOauthServer(); err != nil {
		t.Fatalf("InitializeOauthServer failed: %v", err)
	}

	// GetUser with empty username
	if _, err := keycloak.KeycloakClient.GetUser(""); err == nil {
		t.Fatalf("expected error for empty username")
	}

	// GetUserByEmail with empty email
	if _, err := keycloak.KeycloakClient.GetUserByEmail(""); err == nil {
		t.Fatalf("expected error for empty email")
	}

	// CreateUser empty email
	if _, err := keycloak.KeycloakClient.CreateUser("u1", "f", "l", "", "p"); err == nil {
		t.Fatalf("expected error for create user with empty email")
	}

	// GetOrCreateUser empty
	if _, _, err := keycloak.KeycloakClient.GetOrCreateUser(""); err == nil {
		t.Fatalf("expected error for GetOrCreateUser empty")
	}

	// GetOrCreateVendor empty
	if _, err := keycloak.KeycloakClient.GetOrCreateVendor(""); err == nil {
		t.Fatalf("expected error for GetOrCreateVendor empty")
	}
}

func TestUpdateAndVendorFlows(t *testing.T) {
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	config.InitConfig()
	// ensure SMTPSenderAddress empty so email sending is skipped
	config.Config.SMTPSenderAddress = ""
	if err := keycloak.InitializeOauthServer(); err != nil {
		t.Fatalf("InitializeOauthServer failed: %v", err)
	}

	// create an initial user (oldEmail)
	old := fmt.Sprintf("old_%d@example.com", time.Now().UnixNano())
	newE := fmt.Sprintf("new_%d@example.com", time.Now().UnixNano())
	// cleanup
	_ = keycloak.KeycloakClient.DeleteUser(old)
	_ = keycloak.KeycloakClient.DeleteUser(newE)

	_, err := keycloak.KeycloakClient.CreateUser(old, "Old", "User", old, "password")
	if err != nil {
		t.Fatalf("CreateUser old failed: %v", err)
	}
	// UpdateVendor when old exists -> should update by id
	gotID, err := keycloak.KeycloakClient.UpdateVendor(old, newE, "lic", "First", "Last")
	if err != nil {
		t.Fatalf("UpdateVendor existing failed: %v", err)
	}
	if gotID == "" {
		t.Fatalf("UpdateVendor returned empty id")
	}

	// Now call UpdateVendor with old not existing (use a fresh email)
	anotherOld := fmt.Sprintf("another_old_%d@example.com", time.Now().UnixNano())
	_ = keycloak.KeycloakClient.DeleteUser(anotherOld)
	gotID2, err := keycloak.KeycloakClient.UpdateVendor(anotherOld, fmt.Sprintf("created_%d@example.com", time.Now().UnixNano()), "lic2", "F", "L")
	if err != nil {
		t.Fatalf("UpdateVendor creating new failed: %v", err)
	}
	if gotID2 == "" {
		t.Fatalf("UpdateVendor (create) returned empty id")
	}

	// Test UpdateUserById path: create user and update by id
	uemail := fmt.Sprintf("upbyid_%d@example.com", time.Now().UnixNano())
	_ = keycloak.KeycloakClient.DeleteUser(uemail)
	uid3, err := keycloak.KeycloakClient.CreateUser(uemail, "U", "N", uemail, "pass")
	if err != nil {
		t.Fatalf("CreateUser for UpdateUserById failed: %v", err)
	}
	// update by id with different email to toggle EmailVerified branch
	if err := keycloak.KeycloakClient.UpdateUserById(uid3, "newuname", "First", "Last", fmt.Sprintf("diff_%d@example.com", time.Now().UnixNano())); err != nil {
		t.Fatalf("UpdateUserById failed: %v", err)
	}

	// ensure delete cleanup
	_ = keycloak.KeycloakClient.DeleteUser(old)
	_ = keycloak.KeycloakClient.DeleteUser(newE)
	_ = keycloak.KeycloakClient.DeleteUser(uid3)
}

func TestSendPasswordResetGuard(t *testing.T) {
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	config.InitConfig()
	// ensure sender empty
	config.Config.SMTPSenderAddress = ""
	if err := keycloak.InitializeOauthServer(); err != nil {
		t.Fatalf("InitializeOauthServer failed: %v", err)
	}
	email := fmt.Sprintf("pw_%d@example.com", time.Now().UnixNano())
	_ = keycloak.KeycloakClient.DeleteUser(email)
	_, err := keycloak.KeycloakClient.CreateUser(email, "P", "W", email, "pw")
	if err != nil {
		t.Fatalf("CreateUser for pw reset failed: %v", err)
	}
	defer func() { _ = keycloak.KeycloakClient.DeleteUser(email) }()

	if err := keycloak.KeycloakClient.SendPasswordResetEmail(email); err != nil {
		t.Fatalf("SendPasswordResetEmail guard expected nil, got: %v", err)
	}
	if err := keycloak.KeycloakClient.SendPasswordResetEmailVendor(email); err != nil {
		t.Fatalf("SendPasswordResetEmailVendor guard expected nil, got: %v", err)
	}
}

func TestAssignDigitalLicenseGroup(t *testing.T) {
	// Ensure we are in the app root
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if err := os.Chdir(".."); err != nil {
			t.Fatal(err)
		}
	}

	config.InitConfig()
	// We don't need actual email sending for this test
	config.Config.SMTPSenderAddress = ""

	if err := keycloak.InitializeOauthServer(); err != nil {
		t.Fatalf("InitializeOauthServer failed: %v", err)
	}

	// Create a unique user
	userName := "license_test_user_" + utils.RandomString(5)
	userEmail := userName + "@example.com"

	// Cleanup
	defer func() {
		_ = keycloak.KeycloakClient.DeleteUser(userEmail)
	}()

	// 1. Create User
	userID, _, err := keycloak.KeycloakClient.GetOrCreateUser(userEmail)
	require.NoError(t, err)
	require.NotEmpty(t, userID)

	// 2. Assign Digital License Group
	// This should create the group structure /customer/newspapers/test_digital_edition if it doesn't exist
	licenseGroup := "test_digital_edition"
	err = keycloak.KeycloakClient.AssignDigitalLicenseGroup(userID, licenseGroup)
	require.NoError(t, err)

	// 3. Verify
	groups, err := keycloak.KeycloakClient.GetUserGroups(userID)
	require.NoError(t, err)

	found := false
	for _, g := range groups {
		// The user should be assigned to the specific edition group
		if g.Name != nil && *g.Name == licenseGroup {
			found = true
			break
		}
	}
	require.True(t, found, "User should be assigned to the digital license group 'test_digital_edition'")
}

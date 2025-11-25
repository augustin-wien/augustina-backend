// Documentation see here: https://go-chi.io/#/pages/testing
package keycloak_test

import (
	"net/http"
	"os"
	"testing"

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
	userID, err := keycloak.KeycloakClient.GetOrCreateUser(userEmail)
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

// Documentation see here: https://go-chi.io/#/pages/testing
package keycloak_test

import (
	"augustin/config"
	"augustin/database"
	"augustin/handlers"
	"augustin/keycloak"
	"augustin/utils"
	"net/http"
	"os"
	"testing"

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

func TestKeycloak(t *testing.T) {
	// Test the keycloak functions
	var err error
	// run tests in mainfolder
	err = os.Chdir("..")
	if err != nil {
		panic(err)
	}
	config.InitConfig()
	err = keycloak.InitializeOauthServer()
	if err != nil {
		log.Fatal(err)
	}
	role_name := "testrole"
	err = keycloak.KeycloakClient.CreateRole(role_name)
	if err != nil {
		log.Error("Create role failed:", err)
	}
	role, err := keycloak.KeycloakClient.GetRole(role_name)
	if err != nil {
		log.Error("Get role failed:", err)
	}

	require.Equal(t, role_name, *role.Name)

	_, err = keycloak.KeycloakClient.CreateUser("testuser", "testuser", "testuser@example.com", "password")
	if err != nil {
		log.Errorf("Create user failed: %v \n", err)
	}

	user, err := keycloak.KeycloakClient.GetUser("testuser@example.com")
	if err != nil {
		log.Error("Get user failed:", err)
		panic(err)
	}

	err = keycloak.KeycloakClient.AssignRole(*user.ID, role_name)
	if err != nil {
		log.Error("Assign role failed:", err)
	}

	roles, err := keycloak.KeycloakClient.GetUserRoles(*user.ID)
	if err != nil {
		log.Error("Get user failed:", err)
	}
	require.NotNil(t, lookupRole(role_name, roles))

	err = keycloak.KeycloakClient.UnassignRole(*user.ID, role_name)
	if err != nil {
		log.Error("Unassign role failed:", err)
	}

	roles, err = keycloak.KeycloakClient.GetUserRoles(*user.ID)
	if err != nil {
		log.Error("Get user failed:", err)
	}
	require.Nil(t, lookupRole(role_name, roles))

	err = keycloak.KeycloakClient.DeleteUser(*user.ID)
	if err != nil {
		log.Error("Delete user failed:", err)
	}

	err = keycloak.KeycloakClient.DeleteRole(role_name)
	if err != nil {
		log.Error("Delete role failed:", err)
	}

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

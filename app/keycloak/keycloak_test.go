// Documentation see here: https://go-chi.io/#/pages/testing
package keycloak_test

import (
	"augustin/database"
	"augustin/handlers"
	"augustin/keycloak"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// WARNING: The tests use the main database, do not run tests in production

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeRequest(req *http.Request, s *handlers.Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)

	return rr
}

// checkResponseCode is a simple utility to check the response code
// of the response
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func lookupRole(roleName string, roles []*gocloak.Role) *gocloak.Role {
	for _, role := range roles {
		if *role.Name == roleName {
			return role
		}
	}
	return nil
}

func TestKeycloak(t *testing.T) {
	godotenv.Load("../.env")
	// Test the keycloak functions
	keycloak.InitializeOauthServer()
	var err error
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

	_, err = keycloak.KeycloakClient.CreateUser("testuser", "testuser", "testuser@example.com")
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
	// TODO: This is not a test database, but the real one!
	err := godotenv.Load("../.env")
	if err != nil {
		log.Error("Error loading .env file")
	}
	database.InitDb()

	// Create a New Server Struct
	s := handlers.CreateNewServer()

	// Mount Handlers
	s.MountHandlers()

	// Create a New Request
	req, _ := http.NewRequest("GET", "/api/auth/hello/", nil)

	// Execute Request
	response := executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, 401, response.Code)

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, "Unauthorized\n", response.Body.String())
}

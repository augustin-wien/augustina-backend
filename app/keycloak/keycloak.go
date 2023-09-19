package keycloak

import (
	"augustin/utils"
	"context"
	"os"

	"github.com/Nerzal/gocloak/v13"
	"github.com/joho/godotenv"
)

var log = utils.GetLogger()

// KeycloakClient global variable
var KeycloakClient Keycloak

// Keycloak struct
type Keycloak struct {
	hostname     string
	ClientID     string
	ClientSecret string
	Realm        string
	Client       *gocloak.GoCloak
	Context      context.Context
	adminToken   *gocloak.JWT
	clientToken  *gocloak.JWT
}

// InitializeOauthServer initializes the Keycloak client
// and stores it in the global variable KeycloakClient
func InitializeOauthServer() (err error) {
	godotenv.Load("../.env")
	KeycloakClient = Keycloak{
		hostname:     os.Getenv("KEYCLOAK_HOST"),
		ClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		Realm:        os.Getenv("KEYCLOAK_REALM"),
		Client:       nil,
		Context:      context.Background(),
		adminToken:   nil,
		clientToken:  nil,
	}
	// Initialize Keycloak client
	client := gocloak.NewClient(KeycloakClient.hostname)
	KeycloakClient.Client = client
	KeycloakClient.adminToken, err = KeycloakClient.Login("admin", "admin")
	if err != nil {
		log.Error("Error logging in Keycloak admin ", err)
	}
	KeycloakClient.clientToken, err = KeycloakClient.LoginClient()
	if err != nil {
		log.Error("Error logging in Keycloak client ", err)
	}
	return err
}

// Login function returns the admin token
func (k *Keycloak) Login(username string, password string) (*gocloak.JWT, error) {
	return k.Client.LoginAdmin(k.Context, username, password, "master")
}

// LoginClient function returns the client token
func (k *Keycloak) LoginClient() (*gocloak.JWT, error) {
	return k.Client.LoginClient(k.Context, k.ClientID, k.ClientSecret, k.Realm)
}

// GetUserInfo function returns the user info
func (k *Keycloak) GetUserInfo(userToken string) (*gocloak.UserInfo, error) {
	return k.Client.GetUserInfo(k.Context, userToken, k.Realm)
}

func (k *Keycloak) GetUserToken(user, password string) (*gocloak.JWT, error) {
	return k.Client.Login(k.Context, k.ClientID, k.ClientSecret, k.Realm, user, password)
}

// GetUserByID function queries the user id
func (k *Keycloak) GetUserByID(userToken string, id string) (*gocloak.User, error) {
	return k.Client.GetUserByID(k.Context, userToken, k.Realm, id)
}

// IntrospectToken function returns the token info
func (k *Keycloak) IntrospectToken(userToken string) (*gocloak.IntroSpectTokenResult, error) {
	return k.Client.RetrospectToken(k.Context, userToken, k.ClientID, k.ClientSecret, k.Realm)
}

// GetRoles function returns the roles
func (k *Keycloak) GetRoles() ([]*gocloak.Role, error) {
	return k.Client.GetRealmRoles(k.Context, k.adminToken.AccessToken, k.Realm, gocloak.GetRoleParams{})
}

// GetUserRoles function returns the user roles
func (k *Keycloak) GetUserRoles(userID string) ([]*gocloak.Role, error) {
	return k.Client.GetRealmRolesByUserID(k.Context, k.adminToken.AccessToken, k.Realm, userID)
}

// GetRole function returns the role of the given name
func (k *Keycloak) GetRole(name string) (*gocloak.Role, error) {
	return k.Client.GetRealmRole(k.Context, k.adminToken.AccessToken, k.Realm, name)
}

// CreateRole function creates a role given by name
func (k *Keycloak) CreateRole(name string) error {
	var role = gocloak.Role{
		Name: &name,
	}
	_, err := k.Client.CreateRealmRole(k.Context, k.adminToken.AccessToken, k.Realm, role)
	return err
}

// DeleteRole function deletes a role given by name
func (k *Keycloak) DeleteRole(name string) error {
	return k.Client.DeleteRealmRole(k.Context, k.adminToken.AccessToken, k.Realm, name)
}

// AssignRole function assigns a role to a user by userID
func (k *Keycloak) AssignRole(userID string, roleName string) error {
	role, err := k.GetRole(roleName)
	if err != nil {
		return err
	}
	return k.Client.AddRealmRoleToUser(k.Context, k.adminToken.AccessToken, k.Realm, userID, []gocloak.Role{*role})
}

// UnassignRole function unassigns a role from a user by userID
func (k *Keycloak) UnassignRole(userID string, roleName string) error {
	role, err := k.GetRole(roleName)
	if err != nil {
		return err
	}
	return k.Client.DeleteRealmRoleFromUser(k.Context, k.adminToken.AccessToken, k.Realm, userID, []gocloak.Role{*role})
}

// GetUser function returns the user by username
func (k *Keycloak) GetUser(username string) (*gocloak.User, error) {
	exact := true
	p := gocloak.GetUsersParams{
		Username: &username,
		Exact:    &exact,
	}
	users, err := k.Client.GetUsers(k.Context, k.adminToken.AccessToken, k.Realm, p)
	// if length of users is 0, then user does not exist
	if len(users) == 0 {
		err = gocloak.APIError{
			Code:    404,
			Message: "User does not exist",
		}
		return nil, err
	}
	return users[0], err
}

// CreateUser function creates a user given by first_name, last_name and email returns the userID
func (k *Keycloak) CreateUser(firstName string, lastName string, email string, password string) (userID string, err error) {
	credentials := []gocloak.CredentialRepresentation{
		{
			Type:      gocloak.StringP("password"),
			Value:     gocloak.StringP(password),
			Temporary: gocloak.BoolP(false),
		},
	}
	return k.Client.CreateUser(k.Context, k.adminToken.AccessToken, k.Realm, gocloak.User{
		Username:      &email,
		FirstName:     &firstName,
		LastName:      &lastName,
		Email:         &email,
		Credentials:   &credentials,
		EmailVerified: gocloak.BoolP(true),
		Enabled:       gocloak.BoolP(true),
	})
}

// DeleteUser function deletes a user given by userID
func (k *Keycloak) DeleteUser(userID string) error {
	return k.Client.DeleteUser(k.Context, k.adminToken.AccessToken, k.Realm, userID)
}

package keycloak

import (
	"context"
	"os"

	"github.com/Nerzal/gocloak/v13"
	log "github.com/sirupsen/logrus"
)

var KeycloakClient Keycloak

type Keycloak struct {
	Hostname     string
	ClientId     string
	ClientSecret string
	Realm        string
	Client       *gocloak.GoCloak
	Context      context.Context
	admin_token  *gocloak.JWT
	client_token *gocloak.JWT
}

// InitializeOauthServer initializes the Keycloak client
// and stores it in the global variable KeycloakClient
func InitializeOauthServer() {
	log.Info("Initializing Keycloak client")
	var err error
	KeycloakClient = Keycloak{
		Hostname:     os.Getenv("KEYCLOAK_HOST"),
		ClientId:     os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		Realm:        os.Getenv("KEYCLOAK_REALM"),
		Client:       nil,
		Context:      context.Background(),
		admin_token:  nil,
		client_token: nil,
	}
	// Initialize Keycloak client
	client := gocloak.NewClient(KeycloakClient.Hostname)
	KeycloakClient.Client = client
	KeycloakClient.admin_token, err = KeycloakClient.Login("admin", "admin")
	if err != nil {
		log.Error("Error logging in Keycloak admin ", err)
	}
	KeycloakClient.client_token, err = KeycloakClient.LoginClient()
	if err != nil {
		log.Error("Error logging in Keycloak client ", err)
	}
}
func (k *Keycloak) Login(username string, password string) (*gocloak.JWT, error) {
	return k.Client.LoginAdmin(k.Context,username, password, "master")
}

func (k *Keycloak) LoginClient() (*gocloak.JWT, error) {
	return k.Client.LoginClient(k.Context, k.ClientId, k.ClientSecret, k.Realm)
}


//  User functions
func (k *Keycloak) GetUserInfo(user_token string) (*gocloak.UserInfo, error) {
	return k.Client.GetUserInfo(k.Context, user_token, k.Realm)
}

func (k *Keycloak) GetUserByID(user_token string, id string) (*gocloak.User, error) {
	return k.Client.GetUserByID(k.Context, user_token, k.Realm, id)
}

func (k *Keycloak) IntrospectToken(user_token string) (*gocloak.IntroSpectTokenResult, error) {
	return k.Client.RetrospectToken(k.Context, user_token, k.ClientId, k.ClientSecret, k.Realm)
}


// Admin functions
func (k *Keycloak) GetRoles() ([]*gocloak.Role, error) {
	return k.Client.GetRealmRoles(k.Context, k.admin_token.AccessToken, k.Realm, gocloak.GetRoleParams{})
}

func (k *Keycloak) GetUserRoles(user_id string) ([]*gocloak.Role, error) {
	return k.Client.GetRealmRolesByUserID(k.Context, k.admin_token.AccessToken, k.Realm, user_id)
}

func (k *Keycloak) GetRole(name string) (*gocloak.Role, error) {
	return k.Client.GetRealmRole(k.Context, k.admin_token.AccessToken, k.Realm, name)
}

func (k *Keycloak) CreateRole(name string) error {
	var role = gocloak.Role{
		Name: &name,
	}
	_, err := k.Client.CreateRealmRole(k.Context, k.admin_token.AccessToken, k.Realm, role)
	return err
}

func (k *Keycloak) DeleteRole(name string) error {
	return k.Client.DeleteRealmRole(k.Context, k.admin_token.AccessToken, k.Realm, name)
}

func (k *Keycloak) AssignRole(user_id string, role_name string) error {
	role, err := k.GetRole(role_name)
	if err != nil {
		return err
	}
	return k.Client.AddRealmRoleToUser(k.Context, k.admin_token.AccessToken, k.Realm, user_id, []gocloak.Role{*role,})
}

func (k *Keycloak) UnassignRole(user_id string, role_name string) error {
	role, err := k.GetRole(role_name)
	if err != nil {
		return err
	}
	return k.Client.DeleteRealmRoleFromUser(k.Context, k.admin_token.AccessToken, k.Realm, user_id, []gocloak.Role{*role,})
}

func (k *Keycloak) GetUser(username string) (*gocloak.User, error) {
	exact := true
	p := gocloak.GetUsersParams{
		Username: &username,
		Exact:   &exact,
	}
	users, err := k.Client.GetUsers(k.Context, k.admin_token.AccessToken, k.Realm, p)
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

func (k *Keycloak) CreateUser(first_name string, last_name string, email string) (user_id string, err error) {
	return k.Client.CreateUser(k.Context, k.admin_token.AccessToken, k.Realm, gocloak.User{
		Username: &email,
		FirstName: &first_name,
		LastName:  &last_name,
		Email:     &email,
	})
}

func (k *Keycloak) DeleteUser(user_id string) error {
	return k.Client.DeleteUser(k.Context, k.admin_token.AccessToken, k.Realm, user_id)
}

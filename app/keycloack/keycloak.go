package keycloack

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
	token        *gocloak.JWT
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
		token:        nil,
	}
	// Initialize Keycloak client
	client := gocloak.NewClient(KeycloakClient.Hostname)
	KeycloakClient.Client = client
	KeycloakClient.token, err = KeycloakClient.LoginClient()
	if err != nil {
		log.Error("Error logging in Keycloak client ", err)
	}
}
func (k *Keycloak) Login(username string, password string) (*gocloak.JWT, error) {
	return k.Client.LoginAdmin(k.Context, k.Realm, k.ClientId, k.ClientSecret)
}

func (k *Keycloak) LoginClient() (*gocloak.JWT, error) {
	return k.Client.LoginClient(k.Context, k.ClientId, k.ClientSecret, k.Realm)
}

func (k *Keycloak) GetUserInfo(token string) (*gocloak.UserInfo, error) {
	return k.Client.GetUserInfo(k.Context, token, k.Realm)
}

func (k *Keycloak) GetUserByID(token string, id string) (*gocloak.User, error) {
	return k.Client.GetUserByID(k.Context, token, k.Realm, id)
}

func (k *Keycloak) IntrospectToken(token string) (*gocloak.IntroSpectTokenResult, error) {
	return k.Client.RetrospectToken(k.Context, token, k.ClientId, k.ClientSecret, k.Realm)
}

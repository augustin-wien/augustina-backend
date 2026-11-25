package keycloak

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/Nerzal/gocloak/v13"
	"github.com/go-resty/resty/v2"
)

var log = utils.GetLogger()

// tokenRenewMargin is how long before it expires the admin token is renewed, so
// it doesn't run out in the middle of a series of admin API calls
const tokenRenewMargin = 30 * time.Second

// KeycloakClient global variable
var KeycloakClient Keycloak

// Keycloak struct
type Keycloak struct {
	hostname           string
	ClientID           string
	ClientSecret       string
	Realm              string
	Client             *gocloak.GoCloak
	Context            context.Context
	clientToken        *gocloak.JWT
	clientTokenRenewAt time.Time
	VendorGroup        string
	CustomerGroup      string
	BackofficeGroup    string
	NewspaperGroup     string
	mu                 sync.Mutex
}

// InitializeOauthServer initializes the Keycloak client
// and stores it in the global variable KeycloakClient
func InitializeOauthServer() (err error) {
	if err != nil {
		log.Info("Error loading .env file which is okey if we are in production")
	}
	// Normalize hostname to avoid double slashes in request paths.
	host := strings.TrimRight(config.Config.KeycloakHostname, "/")
	KeycloakClient = Keycloak{
		hostname:        host,
		ClientID:        config.Config.KeycloakClientID,
		ClientSecret:    config.Config.KeycloakClientSecret,
		Realm:           config.Config.KeycloakRealm,
		Client:          nil,
		Context:         context.Background(),
		clientToken:     nil,
		VendorGroup:     "vendors",
		CustomerGroup:   "customer",
		BackofficeGroup: "backoffice",
		NewspaperGroup:  "newspapers",
	}
	// Initialize Keycloak client
	KeycloakClient.Client = KeycloakClient.newGoCloakClient()
	KeycloakClient.mu.Lock()
	err = KeycloakClient.renewClientToken()
	KeycloakClient.mu.Unlock()
	if err != nil {
		return fmt.Errorf("logging in Keycloak client, a running keycloak server is necessary: %w", err)
	}

	// Ensure required group paths exist (creates missing groups and subgroups)
	if err := KeycloakClient.EnsureGroupPath("/" + KeycloakClient.VendorGroup); err != nil {
		log.Error("Error ensuring keycloak vendor group", KeycloakClient.VendorGroup, err)
	}
	if err := KeycloakClient.EnsureGroupPath("/" + KeycloakClient.CustomerGroup); err != nil {
		log.Error("Error ensuring keycloak customer group", KeycloakClient.CustomerGroup, err)
	}
	if err := KeycloakClient.EnsureGroupPath("/" + KeycloakClient.BackofficeGroup); err != nil {
		log.Error("Error ensuring keycloak backoffice group", KeycloakClient.BackofficeGroup, err)
	}
	// Ensure customer/newspapers exists
	if err := KeycloakClient.EnsureGroupPath("/" + KeycloakClient.CustomerGroup + "/" + KeycloakClient.NewspaperGroup); err != nil {
		log.Error("Error ensuring keycloak newspaper group", err)
	}

	return err
}

// newGoCloakClient returns a gocloak client that retries an admin API request once
// with a new token when Keycloak rejects the cached one, which can happen before
// the token expires, e.g. after a Keycloak restart. Requests that do not reach
// Keycloak at all, or hit it while it is unavailable, are retried with backoff.
func (k *Keycloak) newGoCloakClient() *gocloak.GoCloak {
	client := gocloak.NewClient(k.hostname)
	restyClient := client.RestyClient()
	restyClient.SetRetryCount(3)
	restyClient.SetRetryWaitTime(500 * time.Millisecond)
	restyClient.SetRetryMaxWaitTime(2 * time.Second)
	restyClient.AddRetryCondition(func(resp *resty.Response, _ error) bool {
		return resp != nil && resp.StatusCode() == http.StatusUnauthorized &&
			resp.Request.Attempt == 1 && strings.Contains(resp.Request.URL, "/admin/realms/")
	})
	restyClient.AddRetryCondition(func(resp *resty.Response, err error) bool {
		if resp == nil {
			return false
		}
		if err != nil {
			return true
		}
		switch resp.StatusCode() {
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return false
	})
	restyClient.AddRetryHook(func(resp *resty.Response, _ error) {
		if resp == nil || resp.StatusCode() != http.StatusUnauthorized {
			return
		}
		token, err := k.renewRejectedToken(resp.Request.Token)
		if err != nil {
			log.Error("Error logging in Keycloak client after the token was rejected ", err)
			return
		}
		resp.Request.SetAuthToken(token)
	})
	return client
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

// GetUserToken function returns the user token
func (k *Keycloak) GetUserToken(user, password string) (*gocloak.JWT, error) {
	return k.Client.Login(k.Context, k.ClientID, k.ClientSecret, k.Realm, user, password)
}

// IntrospectToken function returns the token info
func (k *Keycloak) IntrospectToken(userToken string) (*gocloak.IntroSpectTokenResult, error) {
	return k.Client.RetrospectToken(k.Context, userToken, k.ClientID, k.ClientSecret, k.Realm)
}

func (k *Keycloak) checkAdminToken() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.clientToken != nil && time.Now().Before(k.clientTokenRenewAt) {
		return
	}
	if err := k.renewClientToken(); err != nil {
		log.Error("Error logging in Keycloak client ", err)
	}
}

// renewRejectedToken returns the token to retry a request with that Keycloak
// rejected, logging in again unless a concurrent request already did
func (k *Keycloak) renewRejectedToken(rejected string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.clientToken == nil || k.clientToken.AccessToken == rejected {
		if err := k.renewClientToken(); err != nil {
			return "", err
		}
	}
	return k.clientToken.AccessToken, nil
}

// renewClientToken logs the service account in again. A failed login keeps the
// previous token. Callers must hold k.mu.
func (k *Keycloak) renewClientToken() error {
	requestedAt := time.Now()
	token, err := k.LoginClient()
	if err != nil {
		return err
	}
	lifetime := time.Duration(token.ExpiresIn) * time.Second
	k.clientToken = token
	k.clientTokenRenewAt = requestedAt.Add(lifetime - min(tokenRenewMargin, lifetime/2))
	return nil
}

// adminAccessToken returns the current admin access token, or an empty string
// if no token could be obtained so that Keycloak calls fail with an error.
func (k *Keycloak) adminAccessToken() string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.clientToken == nil {
		return ""
	}
	return k.clientToken.AccessToken
}

func (k *Keycloak) GetVendorGroup() string {
	return k.VendorGroup
}

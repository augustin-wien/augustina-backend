package keycloak

import (
	"augustin/config"
	"augustin/utils"
	"context"

	"github.com/Nerzal/gocloak/v13"
)

var log = utils.GetLogger()

// KeycloakClient global variable
var KeycloakClient Keycloak

// Keycloak struct
type Keycloak struct {
	hostname                string
	ClientID                string
	ClientSecret            string
	Realm                   string
	Client                  *gocloak.GoCloak
	Context                 context.Context
	clientToken             *gocloak.JWT
	clientTokenCreationTime int64
	vendorGroup             string
	customerGroup           string
	backofficeGroup         string
	newspaperGroup          string
}

// InitializeOauthServer initializes the Keycloak client
// and stores it in the global variable KeycloakClient
func InitializeOauthServer() (err error) {
	if err != nil {
		log.Info("Error loading .env file which is okey if we are in production")
	}
	KeycloakClient = Keycloak{
		hostname:        config.Config.KeycloakHostname,
		ClientID:        config.Config.KeycloakClientID,
		ClientSecret:    config.Config.KeycloakClientSecret,
		Realm:           config.Config.KeycloakRealm,
		Client:          nil,
		Context:         context.Background(),
		clientToken:     nil,
		vendorGroup:     config.Config.KeycloakVendorGroup,
		customerGroup:   config.Config.KeycloakCustomerGroup,
		backofficeGroup: config.Config.KeycloakBackofficeGroup,
		newspaperGroup:  config.Config.KeycloakCustomerGroup + "/newspapers",
	}
	// Initialize Keycloak client
	client := gocloak.NewClient(KeycloakClient.hostname)
	KeycloakClient.Client = client
	KeycloakClient.clientTokenCreationTime = utils.GetUnixTime()
	KeycloakClient.clientToken, err = KeycloakClient.LoginClient()
	if err != nil {
		log.Error("Error logging in Keycloak client ", err)
	}

	// Check if groups exists
	_, err = KeycloakClient.Client.GetGroupByPath(KeycloakClient.Context, KeycloakClient.clientToken.AccessToken, KeycloakClient.Realm, KeycloakClient.vendorGroup)
	if err != nil {
		// Create group
		err = KeycloakClient.CreateGroup(KeycloakClient.vendorGroup)
		if err != nil {
			log.Error("Error creating keycloak vendor group ", err)
		}
	}
	_, err = KeycloakClient.Client.GetGroupByPath(KeycloakClient.Context, KeycloakClient.clientToken.AccessToken, KeycloakClient.Realm, KeycloakClient.customerGroup)
	if err != nil {
		// Create group
		err = KeycloakClient.CreateGroup(KeycloakClient.customerGroup)
		if err != nil {
			log.Error("Error creating keycloak customer group ", err)
		}
	}
	_, err = KeycloakClient.Client.GetGroupByPath(KeycloakClient.Context, KeycloakClient.clientToken.AccessToken, KeycloakClient.Realm, KeycloakClient.backofficeGroup)
	if err != nil {
		// Create group
		err = KeycloakClient.CreateGroup(KeycloakClient.backofficeGroup)
		if err != nil {
			log.Error("Error creating keycloak backoffice group ", err)
		}
	}
	_, err = KeycloakClient.Client.GetGroupByPath(KeycloakClient.Context, KeycloakClient.clientToken.AccessToken, KeycloakClient.Realm, KeycloakClient.newspaperGroup)
	if err != nil {
		// Create group
		err = KeycloakClient.CreateGroup(KeycloakClient.newspaperGroup)
		if err != nil {
			log.Error("Error creating keycloak newspaper group ", err)
		}
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

// GetUserToken function returns the user token
func (k *Keycloak) GetUserToken(user, password string) (*gocloak.JWT, error) {
	return k.Client.Login(k.Context, k.ClientID, k.ClientSecret, k.Realm, user, password)
}

// GetUserByID function queries the user id
func (k *Keycloak) GetUserByID(id string) (*gocloak.User, error) {
	k.checkAdminToken()
	return k.Client.GetUserByID(k.Context, k.clientToken.AccessToken, k.Realm, id)
}

// IntrospectToken function returns the token info
func (k *Keycloak) IntrospectToken(userToken string) (*gocloak.IntroSpectTokenResult, error) {
	return k.Client.RetrospectToken(k.Context, userToken, k.ClientID, k.ClientSecret, k.Realm)
}

// GetRoles function returns the roles
func (k *Keycloak) GetRoles() ([]*gocloak.Role, error) {
	k.checkAdminToken()
	return k.Client.GetRealmRoles(k.Context, k.clientToken.AccessToken, k.Realm, gocloak.GetRoleParams{})
}

// GetUserRoles function returns the user roles
func (k *Keycloak) GetUserRoles(userID string) ([]*gocloak.Role, error) {
	k.checkAdminToken()
	return k.Client.GetCompositeRealmRolesByUserID(k.Context, k.clientToken.AccessToken, k.Realm, userID)
}

// GetUserGroups function returns the user groups
func (k *Keycloak) GetUserGroups(userID string) ([]*gocloak.Group, error) {
	k.checkAdminToken()
	return k.Client.GetUserGroups(k.Context, k.clientToken.AccessToken, k.Realm, userID, gocloak.GetGroupsParams{})
}

func (k *Keycloak) checkAdminToken() {
	var err error
	if k.clientToken == nil {
		k.clientToken, err = KeycloakClient.LoginClient()
		if err != nil {
			log.Error("Error logging in Keycloak client ", err)
		}
	}
	// admin  token is expired
	if utils.GetUnixTime()-(k.clientTokenCreationTime+int64(k.clientToken.ExpiresIn)) > 0 {
		k.clientToken, err = KeycloakClient.LoginClient()
		if err != nil {
			log.Error("Error logging in Keycloak admin ", err)
		}
		k.clientTokenCreationTime = utils.GetUnixTime()
	}
}

// GetRole function returns the role of the given name
func (k *Keycloak) GetRole(name string) (*gocloak.Role, error) {
	k.checkAdminToken()
	return k.Client.GetRealmRole(k.Context, k.clientToken.AccessToken, k.Realm, name)
}

// CreateRole function creates a role given by name
func (k *Keycloak) CreateRole(name string) error {
	var role = gocloak.Role{
		Name: &name,
	}
	_, err := k.Client.CreateRealmRole(k.Context, k.clientToken.AccessToken, k.Realm, role)
	return err
}

// DeleteRole function deletes a role given by name
func (k *Keycloak) DeleteRole(name string) error {
	k.checkAdminToken()
	return k.Client.DeleteRealmRole(k.Context, k.clientToken.AccessToken, k.Realm, name)
}

// AssignRole function assigns a role to a user by userID
func (k *Keycloak) AssignRole(userID string, roleName string) error {
	k.checkAdminToken()
	role, err := k.GetRole(roleName)
	if err != nil {
		return err
	}
	return k.Client.AddRealmRoleToUser(k.Context, k.clientToken.AccessToken, k.Realm, userID, []gocloak.Role{*role})
}

// Assign group to user
func (k *Keycloak) AssignGroup(userID string, groupName string) error {
	k.checkAdminToken()
	group, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, groupName)
	if err != nil {
		log.Errorf("Error getting group by path %s", groupName)
		return err
	}
	return k.Client.AddUserToGroup(k.Context, k.clientToken.AccessToken, k.Realm, userID, *group.ID)
}

func (k *Keycloak) AssignDigitalLicenseGroup(userID string, licenseGroup string) error {
	k.checkAdminToken()
	licenseGroupPath := k.newspaperGroup + "/" + licenseGroup
	// Check if group exists
	_, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, licenseGroupPath)
	if err != nil {
		// Create group
		parentGroup, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, k.newspaperGroup)
		if err != nil {
			return err
		}
		err = k.CreateSubGroup(licenseGroup, *parentGroup.ID)
		if err != nil {
			return err
		}
		// Check if group exists
		_, err = k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, licenseGroupPath)
		if err != nil {
			return err
		}
	}
	// Assign user to group
	return k.AssignGroup(userID, licenseGroupPath)
}

func (k *Keycloak) CreateGroup(groupName string) error {
	k.checkAdminToken()
	group := gocloak.Group{
		Name: &groupName,
	}
	_, err := k.Client.CreateGroup(k.Context, k.clientToken.AccessToken, k.Realm, group)
	return err
}
func (k *Keycloak) CreateSubGroup(groupName string, parentGroupID string) error {
	k.checkAdminToken()
	group := gocloak.Group{
		Name: &groupName,
	}
	_, err := k.Client.CreateChildGroup(k.Context, k.clientToken.AccessToken, k.Realm, parentGroupID, group)
	return err
}

// GetFroupByPath function returns the group of the given name
func (k *Keycloak) GetGroupByPath(path string) (*gocloak.Group, error) {
	k.checkAdminToken()
	return k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, path)
}

// GetGroup function returns the group of the given name
func (k *Keycloak) GetGroup(name string) (*gocloak.Group, error) {
	k.checkAdminToken()
	return k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, "/"+name)
}

// DeleteGroup function deletes a group given by name
func (k *Keycloak) DeleteGroup(name string) error {
	k.checkAdminToken()
	group, err := k.GetGroup(name)
	if err != nil {
		return err
	}
	return k.Client.DeleteGroup(k.Context, k.clientToken.AccessToken, k.Realm, *group.ID)
}

func (k *Keycloak) DeleteSubGroupByPath(path string) error {
	k.checkAdminToken()
	group, err := k.GetGroupByPath(path)
	if err != nil {
		return err
	}
	return k.Client.DeleteGroup(k.Context, k.clientToken.AccessToken, k.Realm, *group.ID)
}

// UnassignRole function unassigns a role from a user by userID
func (k *Keycloak) UnassignRole(userID string, roleName string) error {
	k.checkAdminToken()
	role, err := k.GetRole(roleName)
	if err != nil {
		return err
	}
	return k.Client.DeleteRealmRoleFromUser(k.Context, k.clientToken.AccessToken, k.Realm, userID, []gocloak.Role{*role})
}

// GetUser function returns the user by username
func (k *Keycloak) GetUser(username string) (*gocloak.User, error) {
	k.checkAdminToken()
	exact := true
	p := gocloak.GetUsersParams{
		Username: &username,
		Exact:    &exact,
	}
	users, err := k.Client.GetUsers(k.Context, k.clientToken.AccessToken, k.Realm, p)
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
	k.checkAdminToken()
	credentials := []gocloak.CredentialRepresentation{
		{
			Type:      gocloak.StringP("password"),
			Value:     gocloak.StringP(password),
			Temporary: gocloak.BoolP(false),
		},
	}
	return k.Client.CreateUser(k.Context, k.clientToken.AccessToken, k.Realm, gocloak.User{
		Username:      &email,
		FirstName:     &firstName,
		LastName:      &lastName,
		Email:         &email,
		Credentials:   &credentials,
		EmailVerified: gocloak.BoolP(true),
		Enabled:       gocloak.BoolP(true),
	})
}

func (k *Keycloak) GetOrCreateUser(email string) (userID string, err error) {
	k.checkAdminToken()
	user, err := k.GetUser(email)
	if err != nil {
		// User does not exist
		password := utils.RandomString(10)
		user, err := k.CreateUser(email, "", email, password)
		if err != nil {
			return "", err
		}
		log.Info("Created user ", user)
		// Todo: send welcome email with password reset link
		// if config.Config.SendCustomerEmail && config.Config.OnlinePaperUrl != "" {
		// 	log.Info("Keycloak: execute actions email for ", user)

		// 	err = k.Client.ExecuteActionsEmail(k.Context, k.clientToken.AccessToken, k.Realm, gocloak.ExecuteActionsEmail{
		// 		UserID:      &user,
		// 		Lifespan:    gocloak.IntP(0),
		// 		Actions:     &[]string{"UPDATE_PASSWORD"},
		// 		ClientID:    gocloak.StringP("wordpress"),
		// 		RedirectURI: gocloak.StringP(config.Config.OnlinePaperUrl),
		// 	})
		// 	if err != nil {
		// 		log.Errorf("Keycloak: execute actions email failed for %s %s: %v", user, email, err)
		// 	}
		// }
		return user, nil
	}
	return *user.ID, nil
}

// DeleteUser function deletes a user given by userID
func (k *Keycloak) DeleteUser(username string) error {
	k.checkAdminToken()
	// get user for username
	user, err := k.GetUser(username)
	if err != nil {
		return err
	}
	return k.Client.DeleteUser(k.Context, k.clientToken.AccessToken, k.Realm, *user.ID)
}

// UpdateUserPassword function updates a user password given by userID
func (k *Keycloak) UpdateUserPassword(username string, password string) error {
	k.checkAdminToken()
	user, err := k.GetUser(username)
	if err != nil {
		return err
	}
	return k.Client.SetPassword(k.Context, k.clientToken.AccessToken, *user.ID, k.Realm, password, false)
}

// UpdateUser function updates a user given by userID
func (k *Keycloak) UpdateUser(username string, firstName string, lastName string, email string) error {
	k.checkAdminToken()
	user, err := k.GetUser(username)
	if err != nil {
		return err
	}
	user.FirstName = &firstName
	user.LastName = &lastName
	user.Email = &email
	user.EmailVerified = gocloak.BoolP(true)
	user.Enabled = gocloak.BoolP(true)
	user.Username = &email
	return k.Client.UpdateUser(k.Context, k.clientToken.AccessToken, k.Realm, *user)
}

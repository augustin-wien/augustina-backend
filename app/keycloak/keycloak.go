package keycloak

import (
	"context"
	"fmt"
	"strings"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"

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
	VendorGroup             string
	CustomerGroup           string
	BackofficeGroup         string
	NewspaperGroup          string
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
	client := gocloak.NewClient(KeycloakClient.hostname)
	KeycloakClient.Client = client
	KeycloakClient.clientTokenCreationTime = utils.GetUnixTime()
	KeycloakClient.clientToken, err = KeycloakClient.LoginClient()
	if err != nil {
		log.Fatalf("Error logging in Keycloak client. A running keycloak server is necessary! ", err)
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
	fmt.Printf("Assigning role %s to user %s\n", roleName, userID)
	return k.Client.AddRealmRoleToUser(k.Context, k.clientToken.AccessToken, k.Realm, userID, []gocloak.Role{*role})
}

// Assign group to user
func (k *Keycloak) AssignGroup(userID string, groupName string) error {
	k.checkAdminToken()
	// Groups can only be searched by group paths and not by group names. Group paths have to start with / and if it's not there, we add it.
	if groupName[0] != '/' {
		groupName = "/" + groupName
	}
	group, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, groupName)
	if err != nil {
		log.Errorf("Error getting group by path %s", groupName)
		return err
	}
	log.Infof("Assigning user to group %s %s %s", userID, *group.ID, groupName)
	return k.Client.AddUserToGroup(k.Context, k.clientToken.AccessToken, k.Realm, userID, *group.ID)
}

func (k *Keycloak) AssignDigitalLicenseGroup(userID string, licenseGroup string) error {
	k.checkAdminToken()
	licenseGroupPath := "/" + k.CustomerGroup + "/" + k.NewspaperGroup + "/" + licenseGroup
	// Check if group exists
	_, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, licenseGroupPath)
	if err != nil {
		// Create group
		parentGroup, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, "/"+k.CustomerGroup+"/"+k.NewspaperGroup)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error getting group by path %s for %s", "/"+k.CustomerGroup+"/"+k.NewspaperGroup, k.NewspaperGroup)
			return err
		}
		err = k.CreateSubGroup(licenseGroup, *parentGroup.ID)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error creating group %s", licenseGroup)
			return err
		}
		// Check if group exists
		_, err = k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, licenseGroupPath)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error getting group by path %s", licenseGroupPath)
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

// EnsureGroupPath ensures that a group path (e.g. "/customers/newspapers/edition") exists.
// It will create any missing top-level groups or subgroups along the path.
func (k *Keycloak) EnsureGroupPath(path string) error {
	k.checkAdminToken()
	// Normalize path
	p := "/" + strings.Trim(path, "/")
	if p == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	curr := ""
	for i, part := range parts {
		curr = curr + "/" + part
		_, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, curr)
		if err == nil {
			continue
		}
		// Missing group — create it. If top-level, use CreateGroup, else create as child of previous.
		if i == 0 {
			if err := k.CreateGroup(part); err != nil {
				log.Errorf("EnsureGroupPath: failed to create top-level group %s: %v", part, err)
				return err
			}
		} else {
			parent := strings.Join(parts[:i], "/")
			parentPath := "/" + strings.Trim(parent, "/")
			parentGroup, err := k.Client.GetGroupByPath(k.Context, k.clientToken.AccessToken, k.Realm, parentPath)
			if err != nil {
				log.Errorf("EnsureGroupPath: parent group %s not found: %v", parentPath, err)
				return err
			}
			if err := k.CreateSubGroup(part, *parentGroup.ID); err != nil {
				log.Errorf("EnsureGroupPath: failed to create subgroup %s under %s: %v", part, parentPath, err)
				return err
			}
		}
	}
	return nil
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
	if username == "" {
		return nil, fmt.Errorf("GetUser: username is empty")
	}
	k.checkAdminToken()
	exact := true
	// set username to lowercase
	username = utils.ToLower(username)

	p := gocloak.GetUsersParams{
		Username: &username,
		Exact:    &exact,
	}
	users, err := k.Client.GetUsers(k.Context, k.clientToken.AccessToken, k.Realm, p)
	if err != nil {
		log.Error("Keycloak GetUser: Error getting users ", err)
		return nil, err
	}
	// if length of users is 0, then user does not exist
	if len(users) == 0 {
		err = gocloak.APIError{
			Code:    404,
			Message: "Keycloak GetUser: User does not exist " + username,
		}
		return nil, err
	}
	return users[0], err
}

// GetUserByEmail function returns the user by email
func (k *Keycloak) GetUserByEmail(email string) (*gocloak.User, error) {
	if email == "" {
		return nil, fmt.Errorf("GetUserByEmail: email is empty")
	}
	k.checkAdminToken()
	exact := true
	// set email to lowercase
	email = utils.ToLower(email)
	p := gocloak.GetUsersParams{
		Email: &email,
		Exact: &exact,
	}
	users, err := k.Client.GetUsers(k.Context, k.clientToken.AccessToken, k.Realm, p)
	if err != nil {
		return nil, err
	}
	// if length of users is 0, then user does not exist
	if len(users) == 0 {
		err = gocloak.APIError{
			Code:    404,
			Message: "Keycloak GetUser: User does not exist " + email,
		}
		return nil, err
	}
	return users[0], err
}

// CreateUser function creates a user given by first_name, last_name and email returns the userID
func (k *Keycloak) CreateUser(userid string, firstName string, lastName string, email string, password string) (userID string, err error) {
	if email == "" {
		return "", fmt.Errorf("CreateUser: email is empty")
	}
	// set email to lowercase
	email = utils.ToLower(email)
	k.checkAdminToken()
	credentials := []gocloak.CredentialRepresentation{
		{
			Type:      gocloak.StringP("password"),
			Value:     gocloak.StringP(password),
			Temporary: gocloak.BoolP(false),
		},
	}
	return k.Client.CreateUser(k.Context, k.clientToken.AccessToken, k.Realm, gocloak.User{
		Username:      &userid,
		FirstName:     &firstName,
		LastName:      &lastName,
		Email:         &email,
		Credentials:   &credentials,
		EmailVerified: gocloak.BoolP(true),
		Enabled:       gocloak.BoolP(true),
	})
}
func (k *Keycloak) GetOrCreateVendor(email string) (userID string, err error) {
	if email == "" {
		return "", fmt.Errorf("GetOrCreateVendor: email is empty")
	}
	email = utils.ToLower(email)
	k.checkAdminToken()
	user, err := k.GetUser(email)
	if err != nil {
		log.Info("GetOrCreateVendor: User does not exist we create one ", email)
		// User does not exist
		password := utils.RandomString(10)
		user, err := k.CreateUser(email, email, "", email, password)
		if err != nil {
			log.Errorf("GetOrCreateVendor: Error creating keycloak user %s", email)
			return "", err
		}
		log.Info("GetOrCreateVendor: Created user ", user)

		// send welcome email with password reset link
		err = k.SendPasswordResetEmailVendor(email)
		if err != nil {
			// send password reset email only should soft fail
			log.Error("GetOrCreateVendor: Error sending password reset email ", err)
			return user, nil
		}
		return user, nil

	}
	return *user.ID, nil
}

func (k *Keycloak) GetOrCreateUser(email string) (userID string, newUser bool, err error) {
	if email == "" {
		return "", false, fmt.Errorf("GetOrCreateUser: email is empty")
	}
	email = utils.ToLower(email)
	k.checkAdminToken()
	user, err := k.GetUser(email)
	if err != nil {
		log.Info("GetOrCreateUser: User does not exist we create one ", email)
		// User does not exist
		password := utils.RandomString(10)
		user, err := k.CreateUser(email, email, "", email, password)
		if err != nil {
			log.Errorf("GetOrCreateUser: Error creating keycloak user %s", email)
			return "", false, err
		}
		log.Info("GetOrCreateUser: Created user ", user)

		// send welcome email with password reset link
		err = k.SendPasswordResetEmail(email)
		if err != nil {
			// send password reset email only should soft fail
			log.Error("GetOrCreateUser: Error sending password reset email ", err)
		}
		return user, true, nil

	}
	return *user.ID, false, nil
}

// SendPasswordResetEmail function sends a password reset email to the user
func (k *Keycloak) SendPasswordResetEmail(email string) error {
	return k.sendPasswordResetEmail(email, config.Config.OnlinePaperUrl)
}

// SendPasswordResetEmail function sends a password reset email to the user
func (k *Keycloak) SendPasswordResetEmailVendor(email string) error {
	return k.sendPasswordResetEmail(email, config.Config.FrontendURL+"/me")
}

func (k *Keycloak) sendPasswordResetEmail(email, redirectURI string) error {
	k.checkAdminToken()
	email = utils.ToLower(email)
	user, err := k.GetUserByEmail(email)
	if err != nil {
		log.Error("SendPasswordResetEmail: Error getting user by email ", err)
		return err
	}
	// If no SMTP sender is configured in the app config, skip attempting
	// to trigger Keycloak to send emails. This avoids noisy errors during
	// tests and in environments where Keycloak SMTP isn't set up.
	if config.Config.SMTPSenderAddress == "" {
		log.Infof("SendPasswordResetEmail: skipping email for %s because SMTPSenderAddress is not configured", email)
		return nil
	}

	log.Info("SendPasswordResetEmail: Keycloak: execute password reset email for ", email)
	return k.Client.ExecuteActionsEmail(k.Context, k.clientToken.AccessToken, k.Realm, gocloak.ExecuteActionsEmail{
		UserID:      user.ID,
		Lifespan:    gocloak.IntP(600),
		Actions:     &[]string{"UPDATE_PASSWORD"},
		ClientID:    gocloak.StringP("frontend"),
		RedirectURI: gocloak.StringP(redirectURI),
	})
}

// DeleteUser function deletes a user given by userID
func (k *Keycloak) DeleteUser(username string) error {
	k.checkAdminToken()
	username = utils.ToLower(username)
	// get user for username
	user, err := k.GetUser(username)
	if err != nil {
		return err
	}
	return k.Client.DeleteUser(k.Context, k.clientToken.AccessToken, k.Realm, *user.ID)
}

// UpdateUserPassword function updates a user password given by userID
func (k *Keycloak) UpdateUserPassword(username string, password string) error {
	username = utils.ToLower(username)
	k.checkAdminToken()
	user, err := k.GetUser(username)
	if err != nil {
		return err
	}
	return k.Client.SetPassword(k.Context, k.clientToken.AccessToken, *user.ID, k.Realm, password, false)
}

// UpdateUser function updates a user given by userID
func (k *Keycloak) UpdateUser(username string, firstName string, lastName string, email string) error {
	username = utils.ToLower(username)
	email = utils.ToLower(email)

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
	user.Username = &username
	return k.Client.UpdateUser(k.Context, k.clientToken.AccessToken, k.Realm, *user)
}

func (k *Keycloak) UpdateUserById(userID, username, firstName, lastName, email string) error {
	email = utils.ToLower(email)
	k.checkAdminToken()
	user, err := k.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user.Email != nil && *user.Email != email {
		user.EmailVerified = gocloak.BoolP(false)
	} else {
		user.EmailVerified = gocloak.BoolP(true)
	}
	user.FirstName = &firstName
	user.LastName = &lastName
	user.Email = &email
	user.Enabled = gocloak.BoolP(true)
	user.Username = &email
	return k.Client.UpdateUser(k.Context, k.clientToken.AccessToken, k.Realm, *user)
}

func (k *Keycloak) GetVendorGroup() string {
	return k.VendorGroup
}

func (k *Keycloak) UpdateVendor(oldEmail, newEmail, licenseID, firstName, lastName string) (string, error) {

	oldEmail = utils.ToLower(oldEmail)
	newEmail = utils.ToLower(newEmail)
	// Update user in keycloak
	user, err := k.GetUserByEmail(oldEmail)
	if err != nil {
		keycloak_user_id := ""
		// check if new email already exists in keycloak
		new_keycloak_user, err := k.GetUserByEmail(newEmail)
		if err != nil {

			keycloakUser, _, err2 := k.GetOrCreateUser(newEmail)
			if err2 != nil {
				log.Errorf("UpdateVendor: create keycloak user for "+newEmail+" failed: %v %v \n", err2, err)
				return "", fmt.Errorf("UpdateVendor: create keycloak user for "+newEmail+" failed: %v %v", err2, err)
			}
			keycloak_user_id = keycloakUser
		} else {
			keycloak_user_id = *new_keycloak_user.ID
		}

		err = k.AssignGroup(keycloak_user_id, k.VendorGroup)
		if err != nil {

			return "", fmt.Errorf("UpdateVendor: assign keycloak group for "+newEmail+" failed: %v", err)
		}
		return keycloak_user_id, nil
	} else {
		err = k.UpdateUserById(*user.ID, licenseID, firstName, lastName, newEmail)
		if err != nil {
			return "", fmt.Errorf("UpdateVendor: update keycloak user for %s failed: %v", newEmail, fmt.Sprint(err))
		}
		return *user.ID, nil
	}

}

package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

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
// the token expires, e.g. after a Keycloak restart
func (k *Keycloak) newGoCloakClient() *gocloak.GoCloak {
	client := gocloak.NewClient(k.hostname)
	restyClient := client.RestyClient()
	restyClient.SetRetryCount(1)
	restyClient.AddRetryCondition(func(resp *resty.Response, _ error) bool {
		return resp != nil && resp.StatusCode() == http.StatusUnauthorized &&
			resp.Request.Attempt == 1 && strings.Contains(resp.Request.URL, "/admin/realms/")
	})
	restyClient.AddRetryHook(func(resp *resty.Response, _ error) {
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

// GetUserByID function queries the user id
func (k *Keycloak) GetUserByID(id string) (*gocloak.User, error) {
	k.checkAdminToken()
	return k.Client.GetUserByID(k.Context, k.adminAccessToken(), k.Realm, id)
}

// IntrospectToken function returns the token info
func (k *Keycloak) IntrospectToken(userToken string) (*gocloak.IntroSpectTokenResult, error) {
	return k.Client.RetrospectToken(k.Context, userToken, k.ClientID, k.ClientSecret, k.Realm)
}

// GetRoles function returns the roles
func (k *Keycloak) GetRoles() ([]*gocloak.Role, error) {
	k.checkAdminToken()
	return k.Client.GetRealmRoles(k.Context, k.adminAccessToken(), k.Realm, gocloak.GetRoleParams{})
}

// GetUserRoles function returns the user roles
func (k *Keycloak) GetUserRoles(userID string) ([]*gocloak.Role, error) {
	k.checkAdminToken()
	return k.Client.GetCompositeRealmRolesByUserID(k.Context, k.adminAccessToken(), k.Realm, userID)
}

// GetUserGroups function returns the user groups
func (k *Keycloak) GetUserGroups(userID string) ([]*gocloak.Group, error) {
	k.checkAdminToken()
	return k.Client.GetUserGroups(k.Context, k.adminAccessToken(), k.Realm, userID, gocloak.GetGroupsParams{})
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

// GetRole function returns the role of the given name
func (k *Keycloak) GetRole(name string) (*gocloak.Role, error) {
	k.checkAdminToken()
	return k.Client.GetRealmRole(k.Context, k.adminAccessToken(), k.Realm, name)
}

// CreateRole function creates a role given by name
func (k *Keycloak) CreateRole(name string) error {
	var role = gocloak.Role{
		Name: &name,
	}
	_, err := k.Client.CreateRealmRole(k.Context, k.adminAccessToken(), k.Realm, role)
	return err
}

// DeleteRole function deletes a role given by name
func (k *Keycloak) DeleteRole(name string) error {
	k.checkAdminToken()
	return k.Client.DeleteRealmRole(k.Context, k.adminAccessToken(), k.Realm, name)
}

// AssignRole function assigns a role to a user by userID
func (k *Keycloak) AssignRole(userID string, roleName string) error {
	k.checkAdminToken()
	role, err := k.GetRole(roleName)
	if err != nil {
		return err
	}
	fmt.Printf("Assigning role %s to user %s\n", roleName, userID)
	return k.Client.AddRealmRoleToUser(k.Context, k.adminAccessToken(), k.Realm, userID, []gocloak.Role{*role})
}

// Assign group to user
func (k *Keycloak) AssignGroup(userID string, groupName string) error {
	if k.Client == nil {
		return fmt.Errorf("AssignGroup: keycloak client not initialized")
	}
	if groupName == "" {
		return fmt.Errorf("AssignGroup: groupName is empty")
	}
	k.checkAdminToken()
	// Groups can only be searched by group paths and not by group names. Group paths have to start with / and if it's not there, we add it.
	if groupName[0] != '/' {
		groupName = "/" + groupName
	}
	group, err := k.GetGroupByPath(groupName)
	if err != nil {
		log.Errorf("AssignGroup: Error getting group by path %s", groupName)
		return err
	}
	log.Infof("AssignGroup: Assigning user to group %s %s %s", userID, *group.ID, groupName)
	return k.Client.AddUserToGroup(k.Context, k.adminAccessToken(), k.Realm, userID, *group.ID)
}

func (k *Keycloak) AssignDigitalLicenseGroup(userID string, licenseGroup string) error {
	if k.Client == nil {
		return fmt.Errorf("AssignDigitalLicenseGroup: keycloak client not initialized")
	}
	k.checkAdminToken()
	licenseGroupPath := "/" + k.CustomerGroup + "/" + k.NewspaperGroup + "/" + licenseGroup
	// Check if group exists
	_, err := k.GetGroupByPath(licenseGroupPath)
	if err != nil {
		// Create group
		parentGroup, err := k.GetGroupByPath("/" + k.CustomerGroup + "/" + k.NewspaperGroup)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error getting group by path %s for %s", "/"+k.CustomerGroup+"/"+k.NewspaperGroup, k.NewspaperGroup)
			return err
		}
		_, err = k.CreateSubGroup(licenseGroup, *parentGroup.ID)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error creating group %s", licenseGroup)
			return err
		}
		// Check if group exists
		_, err = k.GetGroupByPath(licenseGroupPath)
		if err != nil {
			log.Errorf("AssignDigitalLicenseGroup: Error getting group by path %s", licenseGroupPath)
			return err
		}
	}
	// Assign user to group
	return k.AssignGroup(userID, licenseGroupPath)
}

// SyncLicenseGroupsDiffToKeycloak applies the diff between oldGroups and newGroups to
// Keycloak: groups added by the admin are assigned, groups explicitly removed by the
// admin are unassigned. Groups that were never in our DB (from other applications) are
// never touched.
func (k *Keycloak) SyncLicenseGroupsDiffToKeycloak(userID string, oldGroups, newGroups []string) error {
	oldSet := make(map[string]bool, len(oldGroups))
	for _, g := range oldGroups {
		if g != "" {
			oldSet[g] = true
		}
	}
	newSet := make(map[string]bool, len(newGroups))
	for _, g := range newGroups {
		if g != "" {
			newSet[g] = true
		}
	}

	for g := range newSet {
		if !oldSet[g] {
			if err := k.AssignDigitalLicenseGroup(userID, g); err != nil {
				log.Errorf("SyncLicenseGroupsDiffToKeycloak: add %s to %s: %v", g, userID, err)
			}
		}
	}
	for g := range oldSet {
		if !newSet[g] {
			licenseGroupPath := "/" + k.CustomerGroup + "/" + k.NewspaperGroup + "/" + g
			group, err := k.GetGroupByPath(licenseGroupPath)
			if err != nil {
				continue // group doesn't exist in Keycloak, nothing to remove
			}
			k.checkAdminToken()
			if err := k.Client.DeleteUserFromGroup(k.Context, k.adminAccessToken(), k.Realm, userID, *group.ID); err != nil {
				log.Errorf("SyncLicenseGroupsDiffToKeycloak: remove %s from %s: %v", g, userID, err)
			}
		}
	}
	return nil
}

func (k *Keycloak) CreateGroup(groupName string) (string, error) {
	k.checkAdminToken()
	group := gocloak.Group{
		Name: &groupName,
	}
	id, err := k.Client.CreateGroup(k.Context, k.adminAccessToken(), k.Realm, group)
	return id, err
}
func (k *Keycloak) CreateSubGroup(groupName string, parentGroupID string) (string, error) {
	k.checkAdminToken()
	group := gocloak.Group{
		Name: &groupName,
	}
	id, err := k.Client.CreateChildGroup(k.Context, k.adminAccessToken(), k.Realm, parentGroupID, group)
	return id, err
}

// EnsureGroupPath ensures that a group path (e.g. "/customers/newspapers/edition") exists.
// It will create any missing top-level groups or subgroups along the path.
// Each level is looked up by its path: Keycloak no longer lists subgroups in a
// group's representation, so they can't be found through the parent.
func (k *Keycloak) EnsureGroupPath(path string) error {
	k.checkAdminToken()
	// Normalize path
	p := "/" + strings.Trim(path, "/")
	if p == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")

	var currentParentID string
	for i, part := range parts {
		currentPath := "/" + strings.Join(parts[:i+1], "/")
		id, err := k.groupIDByPath(currentPath)
		if err != nil {
			return fmt.Errorf("EnsureGroupPath: failed to look up %s: %w", currentPath, err)
		}
		if id == "" {
			if i == 0 {
				id, err = k.CreateGroup(part)
			} else {
				id, err = k.CreateSubGroup(part, currentParentID)
			}
			if err != nil {
				// Created concurrently
				if !strings.Contains(err.Error(), "409") && !strings.Contains(err.Error(), "Conflict") {
					return fmt.Errorf("EnsureGroupPath: failed to create %s: %w", currentPath, err)
				}
				id, err = k.groupIDByPath(currentPath)
				if err != nil || id == "" {
					return fmt.Errorf("EnsureGroupPath: failed to find %s after conflict: %v", currentPath, err)
				}
			}
		}
		currentParentID = id
	}
	return nil
}

// groupIDByPath returns the ID of the group at path, or "" when there is none
func (k *Keycloak) groupIDByPath(path string) (string, error) {
	// gocloak puts a "/" before the path; a second one is rejected as not normalized
	group, err := k.Client.GetGroupByPath(k.Context, k.adminAccessToken(), k.Realm, strings.TrimPrefix(path, "/"))
	if err != nil {
		var apiErr *gocloak.APIError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}
	if group == nil || group.ID == nil {
		return "", nil
	}
	return *group.ID, nil
}

// GetGroupByPath function returns the group of the given name
func (k *Keycloak) GetGroupByPath(path string) (*gocloak.Group, error) {
	k.checkAdminToken()
	// Re-implement manual traversal to avoid 400 normalization errors
	p := "/" + strings.Trim(path, "/")
	if p == "/" {
		return nil, fmt.Errorf("root path not supported")
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")

	var currentGroup *gocloak.Group

	for i, part := range parts {
		if i == 0 {
			params := gocloak.GetGroupsParams{Search: &part}
			foundGroups, err := k.Client.GetGroups(k.Context, k.adminAccessToken(), k.Realm, params)
			if err != nil {
				return nil, err
			}
			for _, g := range foundGroups {
				if g.Name != nil && *g.Name == part {
					currentGroup = g
					break
				}
			}
			if currentGroup == nil {
				return nil, fmt.Errorf("group not found: %s", part)
			}
		} else {
			// Retry loop for subgroups (consistency)
			var found *gocloak.Group
			for attempt := 0; attempt < 20; attempt++ {
				// Refresh current group to get subgroups
				fullGroup, err := k.Client.GetGroup(k.Context, k.adminAccessToken(), k.Realm, *currentGroup.ID)
				if err != nil {
					return nil, err
				}

				if fullGroup.SubGroups != nil {
					for _, sub := range *fullGroup.SubGroups {
						if sub.Name != nil && *sub.Name == part {
							// copy to avoid iterator reuse issues
							s := sub
							found = &s
							break
						}
					}
				}
				if found != nil {
					break
				}

				// Fallback: Search globally for the group and check path
				searchParams := gocloak.GetGroupsParams{
					Search: &part,
				}
				searchResults, err := k.Client.GetGroups(k.Context, k.adminAccessToken(), k.Realm, searchParams)
				if err == nil {
					expectedPath := "/" + strings.Join(parts[:i+1], "/")
					for _, g := range searchResults {
						if g.Path != nil && *g.Path == expectedPath {
							found = g
							break
						}
						// Search matches might return the hierarchy (parent -> child)
						// Iterate deeply to find matching path
						found = k.findGroupInHierarchy(g, expectedPath)
						if found != nil {
							break
						}
					}
				}
				if found != nil {
					break
				}

				// Wait before retry
				time.Sleep(500 * time.Millisecond)
			}

			if found == nil {
				return nil, fmt.Errorf("subgroup %s not found in %s after retries", part, *currentGroup.Name)
			}
			currentGroup = found
		}
	}
	return currentGroup, nil
}

func (k *Keycloak) findGroupInHierarchy(g *gocloak.Group, expectedPath string) *gocloak.Group {
	if g.Path != nil && *g.Path == expectedPath {
		return g
	}
	if g.SubGroups != nil {
		for i := range *g.SubGroups {
			sub := &(*g.SubGroups)[i]
			found := k.findGroupInHierarchy(sub, expectedPath)
			if found != nil {
				return found
			}
		}
	}
	return nil
}

// GetGroup function returns the group of the given name
func (k *Keycloak) GetGroup(name string) (*gocloak.Group, error) {
	k.checkAdminToken()
	// First try using GetGroups with search which is safer against path normalization issues
	params := gocloak.GetGroupsParams{
		Search: &name,
	}
	groups, err := k.Client.GetGroups(k.Context, k.adminAccessToken(), k.Realm, params)
	if err == nil {
		for _, g := range groups {
			if g.Name != nil && *g.Name == name {
				return g, nil
			}
		}
	}
	// Fallback to path
	return k.Client.GetGroupByPath(k.Context, k.adminAccessToken(), k.Realm, "/"+name)
}

// DeleteGroup function deletes a group given by name
func (k *Keycloak) DeleteGroup(name string) error {
	k.checkAdminToken()
	group, err := k.GetGroup(name)
	if err != nil {
		return err
	}
	return k.Client.DeleteGroup(k.Context, k.adminAccessToken(), k.Realm, *group.ID)
}

func (k *Keycloak) DeleteSubGroupByPath(path string) error {
	k.checkAdminToken()
	id, err := k.groupIDByPath("/" + strings.Trim(path, "/"))
	if err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("group not found: %s", path)
	}
	return k.Client.DeleteGroup(k.Context, k.adminAccessToken(), k.Realm, id)
}

// UnassignRole function unassigns a role from a user by userID
func (k *Keycloak) UnassignRole(userID string, roleName string) error {
	k.checkAdminToken()
	role, err := k.GetRole(roleName)
	if err != nil {
		return err
	}
	return k.Client.DeleteRealmRoleFromUser(k.Context, k.adminAccessToken(), k.Realm, userID, []gocloak.Role{*role})
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
	users, err := k.Client.GetUsers(k.Context, k.adminAccessToken(), k.Realm, p)
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
	users, err := k.Client.GetUsers(k.Context, k.adminAccessToken(), k.Realm, p)
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

// findUserBySearch looks the user up with Keycloak's fuzzy search and keeps only
// the one whose email or username equals value. It finds users the exact
// username/email lookups miss.
func (k *Keycloak) findUserBySearch(value string) (*gocloak.User, error) {
	k.checkAdminToken()
	value = utils.ToLower(value)
	users, err := k.Client.GetUsers(k.Context, k.adminAccessToken(), k.Realm, gocloak.GetUsersParams{Search: &value})
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if (u.Email != nil && utils.ToLower(*u.Email) == value) || (u.Username != nil && utils.ToLower(*u.Username) == value) {
			return u, nil
		}
	}
	return nil, gocloak.APIError{Code: 404, Message: "Keycloak findUserBySearch: User does not exist " + value}
}

// personNameProhibited mirrors Keycloak's default person-name-prohibited-characters
// validator, which rejects firstName/lastName containing any of these runes.
const personNameProhibited = "<>&\"$%!#?§;*~/\\|^=[]{}()\v"

// sanitizePersonName drops the runes Keycloak refuses in a person name and
// collapses the whitespace left behind.
func sanitizePersonName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(personNameProhibited, r) {
			return ' '
		}
		return r
	}, name)
	return strings.Join(strings.Fields(cleaned), " ")
}

// CreateUser function creates a user given by first_name, last_name and email returns the userID.
// An empty password creates the user without one; they set it themselves through a Keycloak link.
func (k *Keycloak) CreateUser(userid string, firstName string, lastName string, email string, password string) (userID string, err error) {
	if email == "" {
		return "", fmt.Errorf("CreateUser: email is empty")
	}
	// set email to lowercase
	email = utils.ToLower(email)
	firstName = sanitizePersonName(firstName)
	lastName = sanitizePersonName(lastName)
	k.checkAdminToken()
	user := gocloak.User{
		Username:      &userid,
		FirstName:     &firstName,
		LastName:      &lastName,
		Email:         &email,
		EmailVerified: gocloak.BoolP(true),
		Enabled:       gocloak.BoolP(true),
	}
	if password != "" {
		user.Credentials = &[]gocloak.CredentialRepresentation{
			{
				Type:      gocloak.StringP("password"),
				Value:     gocloak.StringP(password),
				Temporary: gocloak.BoolP(false),
			},
		}
	}
	return k.Client.CreateUser(k.Context, k.adminAccessToken(), k.Realm, user)
}

// GetOrCreateVendor returns the Keycloak user for email, creating it if needed.
// A newly created user gets a welcome mail with a password reset link, unless
// sendWelcomeMail is false (e.g. for a generated internal address nobody reads).
func (k *Keycloak) GetOrCreateVendor(email string, sendWelcomeMail bool) (userID string, err error) {
	if email == "" {
		return "", fmt.Errorf("GetOrCreateVendor: email is empty")
	}
	email = utils.ToLower(email)
	k.checkAdminToken()
	user, err := k.GetUser(email)
	if err != nil {
		log.Info("GetOrCreateVendor: User does not exist we create one ", email)
		// User does not exist
		user, err := k.CreateUser(email, email, "", email, "")
		if err != nil {
			var apiErr *gocloak.APIError
			if errors.As(err, &apiErr) && apiErr.Code == http.StatusConflict {
				if existing, searchErr := k.findUserBySearch(email); searchErr == nil && existing.ID != nil {
					return *existing.ID, nil
				}
			}
			log.Errorf("GetOrCreateVendor: Error creating keycloak user %s", email)
			return "", err
		}
		log.Info("GetOrCreateVendor: Created user ", user)
		if !sendWelcomeMail {
			return user, nil
		}

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
	if k.Client == nil {
		return "", false, fmt.Errorf("GetOrCreateUser: keycloak client not initialized")
	}
	if email == "" {
		return "", false, fmt.Errorf("GetOrCreateUser: email is empty")
	}
	email = utils.ToLower(email)
	k.checkAdminToken()
	user, err := k.GetUser(email)
	if err != nil {
		log.Info("GetOrCreateUser: User does not exist we create one ", email, err)
		// User does not exist
		user, err := k.CreateUser(email, email, "", email, "")
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

// SendVerifyEmail sends the customer a mail to confirm their email address
func (k *Keycloak) SendVerifyEmail(email string) error {
	return k.sendActionsEmail(email, config.Config.OnlinePaperUrl, "VERIFY_EMAIL")
}

// SendVerifyEmailVendor sends the vendor a mail to confirm their email address
func (k *Keycloak) SendVerifyEmailVendor(email string) error {
	return k.sendActionsEmail(email, config.Config.FrontendURL+"/me", "VERIFY_EMAIL")
}

func (k *Keycloak) sendPasswordResetEmail(email, redirectURI string) error {
	return k.sendActionsEmail(email, redirectURI, "UPDATE_PASSWORD")
}

// sendActionsEmail lets Keycloak mail the user a link to perform action
func (k *Keycloak) sendActionsEmail(email, redirectURI, action string) error {
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
	// Keycloak rejects an empty (or otherwise unregistered) redirect URI with an opaque
	// "invalid redirect uri" 400 - for the customer path, redirectURI comes from the
	// admin-editable OnlinePaperUrl setting, which defaults to empty until someone fills it
	// in. Skip with a specific, actionable log instead of letting that surface as an
	// unexplained Keycloak API error days later.
	// Keycloak rejects an empty (or otherwise unregistered) redirect URI with an opaque
	// "invalid redirect uri" 400 - for the customer path, redirectURI comes from the
	// admin-editable OnlinePaperUrl setting, which defaults to empty until someone fills it
	// in. Skip with a specific, actionable log instead of letting that surface as an
	// unexplained Keycloak API error days later.
	if redirectURI == "" {
		log.Errorf("SendPasswordResetEmail: skipping email for %s because the redirect URI is empty - set OnlinePaperUrl in the admin settings", email)
		return nil
	}

	log.Info("SendPasswordResetEmail: Keycloak: execute "+action+" email for ", email)
	err = k.Client.ExecuteActionsEmail(k.Context, k.adminAccessToken(), k.Realm, gocloak.ExecuteActionsEmail{
		UserID: user.ID,
		// The mail is often only read hours later, e.g. after the welcome mail for a
		// new vendor or abo customer, so 10 minutes were far too short. Whole days
		// only: Keycloak's formatter writes "1 Tage" for a single day
		Lifespan:    gocloak.IntP(2 * 24 * 60 * 60),
		Actions:     &[]string{action},
		ClientID:    gocloak.StringP("frontend"),
		RedirectURI: gocloak.StringP(redirectURI),
	})
	if err != nil {
		// An "invalid redirect uri" here means redirectURI is not in the Valid Redirect URIs
		// of this realm's "frontend" client - name both so the fix is clear from the alert.
		return fmt.Errorf("realm %s, client frontend, redirect_uri %q: %w", k.Realm, redirectURI, err)
	}
	return nil
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
	return k.Client.DeleteUser(k.Context, k.adminAccessToken(), k.Realm, *user.ID)
}

// UpdateUserPassword function updates a user password given by userID
func (k *Keycloak) UpdateUserPassword(username string, password string) error {
	username = utils.ToLower(username)
	k.checkAdminToken()
	user, err := k.GetUser(username)
	if err != nil {
		return err
	}
	return k.Client.SetPassword(k.Context, k.adminAccessToken(), *user.ID, k.Realm, password, false)
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
	firstName = sanitizePersonName(firstName)
	lastName = sanitizePersonName(lastName)
	user.FirstName = &firstName
	user.LastName = &lastName
	user.Email = &email
	user.EmailVerified = gocloak.BoolP(true)
	user.Enabled = gocloak.BoolP(true)
	user.Username = &username
	return k.Client.UpdateUser(k.Context, k.adminAccessToken(), k.Realm, *user)
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
	firstName = sanitizePersonName(firstName)
	lastName = sanitizePersonName(lastName)
	user.FirstName = &firstName
	user.LastName = &lastName
	user.Email = &email
	user.Enabled = gocloak.BoolP(true)
	user.Username = &email
	return k.Client.UpdateUser(k.Context, k.adminAccessToken(), k.Realm, *user)
}

func (k *Keycloak) GetVendorGroup() string {
	return k.VendorGroup
}

// ErrEmailTaken is returned when another Keycloak user already owns the email
var ErrEmailTaken = errors.New("email is already used by another keycloak user")

// UpdateVendor moves the vendor's Keycloak user to newEmail, creating the user if
// it doesn't exist yet. A newly created user only gets a welcome mail when
// sendWelcomeMail is set.
func (k *Keycloak) UpdateVendor(oldEmail, newEmail, licenseID, firstName, lastName string, sendWelcomeMail bool) (string, error) {

	oldEmail = utils.ToLower(oldEmail)
	newEmail = utils.ToLower(newEmail)
	// Update user in keycloak
	user, err := k.GetUserByEmail(oldEmail)
	if err != nil {
		keycloak_user_id := ""
		// check if new email already exists in keycloak
		new_keycloak_user, err := k.GetUserByEmail(newEmail)
		if err != nil {

			keycloakUser, err2 := k.GetOrCreateVendor(newEmail, sendWelcomeMail)
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
		if newEmail != oldEmail {
			other, lookupErr := k.GetUserByEmail(newEmail)
			if lookupErr == nil && other.ID != nil && *other.ID != *user.ID {
				return "", fmt.Errorf("%w: %s", ErrEmailTaken, newEmail)
			}
		}
		err = k.UpdateUserById(*user.ID, licenseID, firstName, lastName, newEmail)
		if err != nil {
			currentUsername, currentEmail := "", ""
			if user.Username != nil {
				currentUsername = *user.Username
			}
			if user.Email != nil {
				currentEmail = *user.Email
			}
			details := fmt.Sprintf("keycloakID=%s username=%q email=%q oldEmail=%q newEmail=%q", *user.ID, currentUsername, currentEmail, oldEmail, newEmail)
			if other, lookupErr := k.GetUserByEmail(newEmail); lookupErr == nil && other.ID != nil {
				details += fmt.Sprintf(" userWithNewEmail=%s", *other.ID)
			}
			return "", fmt.Errorf("UpdateVendor: update keycloak user for %s failed (%s): %v", newEmail, details, fmt.Sprint(err))
		}
		return *user.ID, nil
	}

}

package keycloak

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/Nerzal/gocloak/v13"
	"github.com/augustin-wien/augustina-backend/utils"
)

// GetUserByID function queries the user id
func (k *Keycloak) GetUserByID(id string) (*gocloak.User, error) {
	k.checkAdminToken()
	return k.Client.GetUserByID(k.Context, k.adminAccessToken(), k.Realm, id)
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
		user, err = k.GetUserByEmail(email)
	}
	if err != nil || user.ID == nil {
		log.Info("GetOrCreateUser: User does not exist we create one ", email, err)
		// User does not exist
		user, err := k.CreateUser(email, email, "", email, "")
		if err != nil {
			var apiErr *gocloak.APIError
			if errors.As(err, &apiErr) && apiErr.Code == http.StatusConflict {
				if existing, searchErr := k.findUserBySearch(email); searchErr == nil && existing.ID != nil {
					return *existing.ID, false, nil
				}
			}
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

// GetOrCreateCustomerUser returns the customer's Keycloak user: the one with
// keycloakID, else the one owning email, else a new one without a password.
// created reports whether the user was created.
func (k *Keycloak) GetOrCreateCustomerUser(keycloakID, email, firstName, lastName string) (userID string, created bool, err error) {
	if k.Client == nil {
		return "", false, fmt.Errorf("GetOrCreateCustomerUser: keycloak client not initialized")
	}
	if keycloakID != "" {
		if user, err := k.GetUserByID(keycloakID); err == nil && user.ID != nil {
			return *user.ID, false, nil
		}
	}
	if email == "" {
		return "", false, fmt.Errorf("GetOrCreateCustomerUser: email is empty")
	}
	email = utils.ToLower(email)
	if user, err := k.GetUserByEmail(email); err == nil && user.ID != nil {
		return *user.ID, false, nil
	}
	if user, err := k.GetUser(email); err == nil && user.ID != nil {
		return *user.ID, false, nil
	}
	userID, err = k.CreateUser(email, firstName, lastName, email, "")
	if err != nil {
		var apiErr *gocloak.APIError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusConflict {
			if existing, searchErr := k.findUserBySearch(email); searchErr == nil && existing.ID != nil {
				return *existing.ID, false, nil
			}
		}
		return "", false, err
	}
	log.Info("GetOrCreateCustomerUser: created user ", userID)
	return userID, true, nil
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

package keycloak

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Nerzal/gocloak/v13"
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/utils"
)

// SendPasswordResetEmailToUser sends the customer with the given Keycloak ID a password reset email
func (k *Keycloak) SendPasswordResetEmailToUser(userID string) error {
	return k.executeActionsEmail(userID, userID, config.Config.OnlinePaperUrl, "UPDATE_PASSWORD")
}

// SendVerifyEmailToUser sends the customer with the given Keycloak ID a mail to confirm their email address
func (k *Keycloak) SendVerifyEmailToUser(userID string) error {
	return k.executeActionsEmail(userID, userID, config.Config.OnlinePaperUrl, "VERIFY_EMAIL")
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
		var apiErr gocloak.APIError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
			log.Warn("SendPasswordResetEmail: Error getting user by email ", err)
		} else {
			log.Error("SendPasswordResetEmail: Error getting user by email ", err)
		}
		return err
	}
	return k.executeActionsEmail(*user.ID, email, redirectURI, action)
}

// executeActionsEmail lets Keycloak mail the user with userID a link to perform
// action; recipient names the user in the logs
func (k *Keycloak) executeActionsEmail(userID, recipient, redirectURI, action string) error {
	k.checkAdminToken()
	// If no SMTP sender is configured in the app config, skip attempting
	// to trigger Keycloak to send emails. This avoids noisy errors during
	// tests and in environments where Keycloak SMTP isn't set up.
	if config.Config.SMTPSenderAddress == "" {
		log.Infof("SendPasswordResetEmail: skipping email for %s because SMTPSenderAddress is not configured", recipient)
		return nil
	}
	// Keycloak rejects an empty (or otherwise unregistered) redirect URI with an opaque
	// "invalid redirect uri" 400 - for the customer path, redirectURI comes from the
	// admin-editable OnlinePaperUrl setting, which defaults to empty until someone fills it
	// in. Skip with a specific, actionable log instead of letting that surface as an
	// unexplained Keycloak API error days later.
	if redirectURI == "" {
		log.Errorf("SendPasswordResetEmail: skipping email for %s because the redirect URI is empty - set OnlinePaperUrl in the admin settings", recipient)
		return nil
	}

	log.Info("SendPasswordResetEmail: Keycloak: execute "+action+" email for ", recipient)
	err := k.Client.ExecuteActionsEmail(k.Context, k.adminAccessToken(), k.Realm, gocloak.ExecuteActionsEmail{
		UserID: &userID,
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

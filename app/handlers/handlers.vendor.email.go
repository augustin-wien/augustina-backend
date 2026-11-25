package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// errVendorHasNoOwnEmail is returned when a mail should go to a vendor that
// only has the generated internal address
var errVendorHasNoOwnEmail = errors.New("vendor has no own email address")

// normalizeVendorEmailPostfix checks a VendorEmailPostfix setting, which is
// appended to a license ID as-is. It may carry more than the domain (e.g.
// "-@example.com"); a bare domain without "@" gets one prepended.
func normalizeVendorEmailPostfix(raw string) (string, error) {
	postfix := strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(postfix, "@") {
		postfix = "@" + postfix
	}
	if strings.Count(postfix, "@") != 1 || strings.HasSuffix(postfix, "@") || strings.ContainsAny(postfix, " \t") {
		return "", fmt.Errorf("vendor email postfix %q must contain a domain and at most one @, e.g. \"@example.com\" or \"-@example.com\"", raw)
	}
	return postfix, nil
}

// vendorEmailPostfix returns the normalized VendorEmailPostfix setting
func vendorEmailPostfix() (string, error) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		return "", err
	}
	return normalizeVendorEmailPostfix(settings.VendorEmailPostfix)
}

// internalVendorEmail builds the address for a vendor without a mailbox of
// their own: the license ID followed by the VendorEmailPostfix setting
func internalVendorEmail(licenseID string) (string, error) {
	postfix, err := vendorEmailPostfix()
	if err != nil {
		return "", err
	}
	local := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		default:
			return '-'
		}
	}, strings.ToLower(strings.TrimSpace(licenseID)))
	if local == "" {
		return "", errors.New("a license ID is required to generate an internal email address")
	}
	return local + postfix, nil
}

// resolveVendorEmail derives HasOwnEmail from the submitted email: an empty
// email, or the generated internal one, means the vendor has no mailbox of
// their own and gets the internal address. Clients that don't know about
// HasOwnEmail (flour, the CSV import) therefore keep working unchanged.
func resolveVendorEmail(vendor *database.Vendor) error {
	email := utils.ToLower(strings.TrimSpace(vendor.Email))
	internal, err := internalVendorEmail(vendor.LicenseID.String)
	if err != nil {
		if email == "" {
			return err
		}
		// No internal address possible, but the vendor brought their own
		vendor.Email = email
		vendor.HasOwnEmail = true
		return nil
	}
	// The CSV import builds the address from the raw license ID, without
	// replacing characters an email address can't contain
	postfix, err := vendorEmailPostfix()
	if err != nil {
		return err
	}
	rawInternal := utils.ToLower(strings.TrimSpace(vendor.LicenseID.String)) + postfix
	if email == "" || email == internal || email == rawInternal {
		vendor.Email = internal
		vendor.HasOwnEmail = false
		return nil
	}
	vendor.Email = email
	vendor.HasOwnEmail = true
	return nil
}

// SendVendorPasswordResetEmail godoc
//
//	 	@Summary 		Send the vendor a password reset email
//		@Description	Fails with 400 if the vendor only has the generated internal address
//		@Tags			Vendors
//		@Success		204
//		@Security		KeycloakAuth
//	    @Param          vendorid   path int  true  "Vendor ID"
//		@Router			/vendors/{vendorid}/password-reset-email/ [post]
func SendVendorPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	sendVendorActionEmail(w, r, "password reset", keycloak.KeycloakClient.SendPasswordResetEmailVendor)
}

// SendVendorVerifyEmail godoc
//
//	 	@Summary 		Send the vendor an email to verify their address
//		@Description	Fails with 400 if the vendor only has the generated internal address
//		@Tags			Vendors
//		@Success		204
//		@Security		KeycloakAuth
//	    @Param          vendorid   path int  true  "Vendor ID"
//		@Router			/vendors/{vendorid}/verify-email/ [post]
func SendVendorVerifyEmail(w http.ResponseWriter, r *http.Request) {
	sendVendorActionEmail(w, r, "verification", keycloak.KeycloakClient.SendVerifyEmailVendor)
}

func sendVendorActionEmail(w http.ResponseWriter, r *http.Request, kind string, send func(email string) error) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorSimple(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	if !vendor.HasOwnEmail {
		utils.ErrorJSON(w, errVendorHasNoOwnEmail, http.StatusBadRequest)
		return
	}
	sendActionEmail(w, r, kind, "vendor", vendorID, vendor.Email, send)
}

// sendActionEmail sends a Keycloak action mail (password reset, verification)
// triggered from the backoffice
func sendActionEmail(w http.ResponseWriter, r *http.Request, kind, recipientType string, id int, email string, send func(email string) error) {
	// Keycloak's mail helpers silently skip sending without an SMTP sender;
	// the backoffice must not report a mail as sent that never left
	if config.Config.SMTPSenderAddress == "" {
		utils.ErrorJSON(w, errors.New("sending emails is not configured"), http.StatusServiceUnavailable)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name")+" is sending a "+kind+" email to "+recipientType+" ", id)
	if err := send(email); err != nil {
		log.Error("sendActionEmail: sending "+kind+" email to "+recipientType+" "+fmt.Sprint(id)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

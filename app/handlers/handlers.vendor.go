package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
	"gopkg.in/guregu/null.v4"
)

// Users ----------------------------------------------------------------------

// errVendorBlocked is returned when someone tries to sell for a vendor that the
// backoffice has blocked
var errVendorBlocked = errors.New("vendor is blocked")

// errVendorHasNoOwnEmail is returned when a mail should go to a vendor that
// only has the generated internal address
var errVendorHasNoOwnEmail = errors.New("vendor has no own email address")

// internalVendorEmail builds the address for a vendor without a mailbox of
// their own: the license ID followed by the VendorEmailPostfix setting
func internalVendorEmail(licenseID string) (string, error) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		return "", err
	}
	postfix := strings.ToLower(strings.TrimSpace(settings.VendorEmailPostfix))
	if postfix == "" || postfix == "@" {
		return "", errors.New("vendor email postfix is not configured in the settings")
	}
	if !strings.HasPrefix(postfix, "@") {
		postfix = "@" + postfix
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
	_, postfix, _ := strings.Cut(internal, "@")
	rawInternal := utils.ToLower(strings.TrimSpace(vendor.LicenseID.String)) + "@" + postfix
	if email == "" || email == internal || email == rawInternal {
		vendor.Email = internal
		vendor.HasOwnEmail = false
		return nil
	}
	vendor.Email = email
	vendor.HasOwnEmail = true
	return nil
}

type checkLicenseIDResponse struct {
	FirstName       string
	AccountProofUrl null.String
}

// CheckVendorsLicenseID godoc
//
//	 	@Summary 		Check for license id
//		@Description	Check if license id exists, return first name of vendor if it does
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//	    @Param		    licenseID path string true "License ID"
//		@Success		200	{string} checkLicenseIDResponse
//		@Response		200	{string} checkLicenseIDResponse
//		@Router			/vendors/check/{licenseID}/ [get]
func CheckVendorsLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("no licenseID provided under /vendors/check/{licenseID}/"), http.StatusBadRequest)
		return
	}

	users, err := database.Db.GetVendorByLicenseIDWithoutDisabled(licenseID)
	if err != nil {
		utils.ErrorJSON(w, errors.New("wrong license id. No vendor exists with this id"), http.StatusBadRequest)
		return
	}
	if users.IsBlocked {
		utils.ErrorJSON(w, errVendorBlocked, http.StatusForbidden)
		return
	}
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	response := checkLicenseIDResponse{
		FirstName:       users.FirstName,
		AccountProofUrl: users.AccountProofUrl,
	}
	if settings.UseVendorLicenseIdInShop {
		response.FirstName = licenseID
	}

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("checkVendorsLicenseID: ", err)
	}
}

// ListVendors godoc
//
//	 	@Summary 		List Vendors
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Security		KeycloakAuth
//		@Success		200	{array}	database.Vendor
//		@Router			/vendors/ [get]
func ListVendors(w http.ResponseWriter, r *http.Request) {
	vendors, err := database.Db.ListVendorsWithDisabled()
	respond(w, err, vendors)
}

// RecalculateAllVendorBalances godoc
//
//	@Summary		Recalculate all vendor balances
//	@Description	Recomputes each vendor's cached balance from their open payments
//	@Tags			Vendors
//	@Security		KeycloakAuth
//	@Success		200
//	@Router			/vendors/recalculate-balances/ [post]
func RecalculateAllVendorBalances(w http.ResponseWriter, r *http.Request) {
	if err := database.Db.RecalculateAllVendorBalances(); err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	vendors, err := database.Db.ListVendorsWithDisabled()
	respond(w, err, vendors)
}

// CreateVendor godoc
//
//	 	@Summary 		Create Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param		    data body database.Vendor true "Vendor Representation"
//		@Router			/vendors/ [post]
func CreateVendor(w http.ResponseWriter, r *http.Request) {
	var vendor database.Vendor
	err := utils.ReadJSON(w, r, &vendor)
	if err != nil {
		log.Error("CreateVendor: ReadJSON failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := resolveVendorEmail(&vendor); err != nil {
		log.Warn("CreateVendor: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name") + " is creating a vendor for " + vendor.Email)

	// Reject a duplicate license ID before touching Keycloak: GetOrCreateVendor
	// below creates a real Keycloak user and emails them a password-reset link,
	// side effects that can't be cleanly undone once the DB insert fails.
	if vendor.LicenseID.String != "" {
		_, checkErr := database.Db.GetVendorByLicenseID(vendor.LicenseID.String)
		if checkErr == nil {
			err := fmt.Errorf("%w: %q", database.ErrLicenseIDTaken, vendor.LicenseID.String)
			log.Warn("CreateVendor: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		} else if !ent.IsNotFound(checkErr) {
			log.Error("CreateVendor: checking license ID failed: ", checkErr)
			utils.ErrorJSON(w, checkErr, http.StatusInternalServerError)
			return
		}
	}

	// Create user in keycloak
	// An internal address has no mailbox, so there is no point in a welcome mail
	user, err := keycloak.KeycloakClient.GetOrCreateVendor(vendor.Email, vendor.HasOwnEmail)
	if err != nil {
		log.Error("CreateVendor: Create keycloak user failed ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info("Created user in keycloak: ", user)
	vendor.KeycloakID = user

	err = keycloak.KeycloakClient.AssignGroup(user, keycloak.KeycloakClient.VendorGroup)
	if err != nil {
		log.Error("CreateVendor: Assigning user to vendor group failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	id, err := database.Db.CreateVendor(vendor)
	if err != nil {
		if errors.Is(err, database.ErrLicenseIDTaken) {
			// Expected race with the pre-check above (e.g. two concurrent
			// creates); not a system fault, so don't alert on it.
			log.Warn("CreateVendor: ", err)
		} else {
			log.Error("CreateVendor: Create vendor in db failed: ", err)
		}
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, id)
}

// GetVendor godoc
//
//	 	@Summary 		Get Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//		@Param          id   path int  true  "Vendor ID"
//		@Router			/vendors/{id}/ [get]
func GetVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorWithBalanceUpdate(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, vendor)
}

type VendorOverview struct {
	ID           int
	FirstName    string
	LastName     string
	Email        string
	LicenseID    string
	UrlID        string
	LastPayout   null.Time `swaggertype:"string" format:"date-time"`
	Balance      int
	Address      string
	PLZ          string
	Locations    []*ent.Location
	Comments     []*ent.Comment
	Telephone    string
	OpenPayments []database.Payment
}

// GetVendorOverview godoc
//
//	 	@Summary 		Get Vendor overview
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200 {object} VendorOverview
//		@Security		KeycloakAuth
//		@Router			/vendors/me/ [get]
func GetVendorOverview(w http.ResponseWriter, r *http.Request) {

	// Get vendors email from keycloak header
	vendorEmail := r.Header.Get("X-Auth-User-Email")
	if vendorEmail == "" {
		utils.ErrorJSON(w, fmt.Errorf("user has no email defined"), http.StatusBadRequest)
		return
	}

	// Get vendor information from database
	vendor, err := database.Db.GetVendorByEmail(vendorEmail)
	if err != nil {
		if err.Error() == "no rows in result set" {
			utils.ErrorJSON(w, fmt.Errorf("user is not a vendor"), http.StatusBadRequest)
			return
		}
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor, err = database.Db.GetVendorWithBalanceUpdate(vendor.ID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get open payments of vendor from database
	minDate := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Now()
	payments, err := database.Db.ListPaymentsForPayout(minDate, maxDate, vendor.LicenseID.String)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create response
	response := VendorOverview{
		ID:           vendor.ID,
		FirstName:    vendor.FirstName,
		LastName:     vendor.LastName,
		Email:        vendor.Email,
		LicenseID:    vendor.LicenseID.String,
		UrlID:        vendor.UrlID,
		LastPayout:   vendor.LastPayout,
		Balance:      vendor.Balance,
		Locations:    vendor.Locations,
		Telephone:    vendor.Telephone,
		OpenPayments: payments,
	}

	// Return response
	respond(w, err, response)
}

// UpdateVendor godoc
//
//	 	@Summary 		Update Vendor
//		@Description	Warning: Unfilled fields will be set to default values
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param          id   path int  true  "Vendor ID"
//		@Param		    data body database.Vendor true "Vendor Representation"
//		@Param			locations query string false "What happens to the vendor's locations when the vendor gets disabled" Enums(keep, delete)
//		@Router			/vendors/{id}/ [put]
func UpdateVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("UpdateVendor: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	keepLocations, err := parseLocationsParam(r)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name")+" is updating vendor with id: ", vendorID)
	var vendor database.Vendor
	err = utils.ReadJSON(w, r, &vendor)
	if err != nil {
		log.Error("UpdateVendor: ReadJSON failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := resolveVendorEmail(&vendor); err != nil {
		log.Warn("UpdateVendor: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	oldVendor, err := database.Db.GetVendorSimple(vendorID)
	if err != nil {
		log.Error("UpdateVendor: get old vendor "+fmt.Sprint(vendorID)+"failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if !oldVendor.IsDeleted || !vendor.IsDeleted {
		keycloakId, err := keycloak.KeycloakClient.UpdateVendor(oldVendor.Email, vendor.Email, vendor.LicenseID.String, vendor.FirstName, vendor.LastName, vendor.HasOwnEmail)
		if err != nil {
			log.Error("UpdateVendor: update user in keycloak for "+fmt.Sprint(vendorID)+" failed: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		vendor.KeycloakID = keycloakId
	}

	err = database.Db.UpdateVendor(vendorID, vendor)
	if err != nil {
		log.Error("UpdateVendor: update vendor in db for "+fmt.Sprint(vendorID)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if keepLocations != nil && !oldVendor.IsDisabled && vendor.IsDisabled {
		err = database.Db.ReleaseVendorLocations(vendorID, *keepLocations)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusInternalServerError)
			return
		}
	}
	respond(w, err, vendor)
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

// parseLocationsParam reads the optional "locations" query parameter that
// decides what happens to a vendor's locations when the vendor is deleted or
// disabled: "keep" leaves them as unassigned locations, "delete" removes them.
// Without the parameter (nil) the locations stay with the vendor.
func parseLocationsParam(r *http.Request) (*bool, error) {
	switch r.URL.Query().Get("locations") {
	case "":
		return nil, nil
	case "keep":
		keep := true
		return &keep, nil
	case "delete":
		keep := false
		return &keep, nil
	default:
		return nil, errors.New("locations must be 'keep' or 'delete'")
	}
}

// DeleteVendor godoc
//
//		@Summary 		Delete Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param          id   path int  true  "Vendor ID"
//		@Param			locations query string false "What happens to the vendor's locations" Enums(keep, delete)
//		@Router			/vendors/{id}/ [delete]
func DeleteVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("DeleteVendor: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	keepLocations, err := parseLocationsParam(r)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name")+" is deleting vendor with id: ", vendorID)
	vendor, err := database.Db.GetVendor(vendorID)
	if err != nil {
		log.Error("DeleteVendor: GetVendor failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Delete user in keycloak
	err = keycloak.KeycloakClient.DeleteUser(vendor.Email)
	if err != nil {
		log.Info("DeleteVendor: Deleting user "+vendor.Email+" failed in keycloak failed: ", err)
		// ignore because not each legacy vendor is in keycloak
	}

	err = database.Db.DeleteVendor(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if keepLocations != nil {
		err = database.Db.ReleaseVendorLocations(vendorID, *keepLocations)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func UpdateVendorByLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("no licenseID provided under /vendors/license/{licenseID}/"), http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var updatedVendor database.Vendor
	err = utils.ReadJSON(w, r, &updatedVendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if err := resolveVendorEmail(&updatedVendor); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	keycloakId, err := keycloak.KeycloakClient.UpdateVendor(vendor.Email, updatedVendor.Email, vendor.LicenseID.String, updatedVendor.FirstName, updatedVendor.LastName, updatedVendor.HasOwnEmail)
	if err != nil {
		log.Error("UpdateVendor: update user in keycloak for "+fmt.Sprint(vendor.ID)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor.KeycloakID = keycloakId
	// flour doesn't know about blocking, so keep whatever the backoffice set
	updatedVendor.IsBlocked = vendor.IsBlocked
	updatedVendor.BlockedNote = vendor.BlockedNote
	err = database.Db.UpdateVendor(vendor.ID, updatedVendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name") + " is updating vendor via flour with license id: " + licenseID)
	respond(w, err, updatedVendor)
}

func GetVendorByLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("no licenseID provided under /vendors/license/{licenseID}/"), http.StatusBadRequest)
		return
	}
	vendor, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	respond(w, err, vendor)
}

// Online Map -----------------------------------------------------------------

// GetVendorLocations godoc
//
//	 	@Summary 		Get longitudes and latitudes of all vendors for online map
//		@Description	Get longitudes and latitudes of all vendors for online map
//		@Tags			Map
//		@Accept			json
//		@Produce		json
//		@Security		KeycloakAuth
//		@Success		200	{array}	database.LocationData
//		@Router			/map/ [get]
func GetVendorLocations(w http.ResponseWriter, r *http.Request) {
	locationData, err := database.Db.GetVendorLocations()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, locationData)
	if err != nil {
		log.Error("GetVendorLocations: ", err)
	}
}

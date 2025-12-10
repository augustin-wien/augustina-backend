package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
	"gopkg.in/guregu/null.v4"
)

// Users ----------------------------------------------------------------------

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
	vendors, err := database.Db.ListVendors()
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
	log.Info(r.Header.Get("X-Auth-User-Name") + " is creating a vendor for" + vendor.Email)

	// Create user in keycloak
	user, err := keycloak.KeycloakClient.GetOrCreateVendor(vendor.Email)
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
		log.Error("CreateVendor: Create vendor in db failed: ", err)
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
//		@Router			/vendors/{id}/ [put]
func UpdateVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("UpdateVendor: Can not read ID ", err)
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
	oldVendor, err := database.Db.GetVendorSimple(vendorID)
	if err != nil {
		log.Error("UpdateVendor: get old vendor "+fmt.Sprint(vendorID)+"failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if !oldVendor.IsDeleted || !vendor.IsDeleted {
		keycloakId, err := keycloak.KeycloakClient.UpdateVendor(oldVendor.Email, vendor.Email, vendor.LicenseID.String, vendor.FirstName, vendor.LastName)
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
	respond(w, err, vendor)
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
//		@Router			/vendors/{id}/ [delete]
func DeleteVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("DeleteVendor: Can not read ID ", err)
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
	keycloakId, err := keycloak.KeycloakClient.UpdateVendor(vendor.Email, updatedVendor.Email, vendor.LicenseID.String, updatedVendor.FirstName, updatedVendor.LastName)
	if err != nil {
		log.Error("UpdateVendor: update user in keycloak for "+fmt.Sprint(vendor.ID)+" failed: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendor.KeycloakID = keycloakId
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

package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

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
		if errors.Is(err, keycloak.ErrEmailTaken) {
			log.Warn("UpdateVendor: vendor "+fmt.Sprint(vendor.ID)+": ", err)
			utils.ErrorJSON(w, err, http.StatusConflict)
			return
		}
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

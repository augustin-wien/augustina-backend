package handlers

import (
	"errors"
	"net/http"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/integrations"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// OpenVendorNextcloudFolder godoc
//
//	 	@Summary 		Get the vendor's document folder in Nextcloud
//		@Description	Creates the vendor's folder in Nextcloud if it does not exist yet and returns a link that opens it
//		@Tags			Vendors
//		@Produce		json
//		@Success		200	{object}	VendorNextcloudFolder
//		@Failure		403
//		@Failure		404
//		@Failure		502
//		@Failure		503
//		@Security		KeycloakAuth
//	    @Param          licenseID   path string  true  "License ID"
//		@Router			/vendors/{licenseID}/nextcloud-folder/ [post]
func OpenVendorNextcloudFolder(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")

	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if !settings.NextcloudEnabled {
		utils.ErrorJSON(w, errors.New("the Nextcloud integration is disabled"), http.StatusForbidden)
		return
	}
	nextcloud := integrations.NextcloudFromSettings(settings)
	if !nextcloud.Configured() {
		utils.ErrorJSON(w, errors.New("the Nextcloud connection is not set up"), http.StatusServiceUnavailable)
		return
	}

	vendor, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		if ent.IsNotFound(err) {
			utils.ErrorJSON(w, database.ErrVendorNotFound, http.StatusNotFound)
			return
		}
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// The folder is named after the stored license ID, not the one from the request URL
	folderURL, err := nextcloud.EnsureVendorFolder(vendor.LicenseID.String)
	if err != nil {
		log.Error("OpenVendorNextcloudFolder: ", err)
		utils.ErrorJSON(w, errors.New("could not open the folder in Nextcloud"), http.StatusBadGateway)
		return
	}
	respond(w, nil, VendorNextcloudFolder{URL: folderURL})
}

// VendorNextcloudFolder is the response of OpenVendorNextcloudFolder
type VendorNextcloudFolder struct {
	URL string
}

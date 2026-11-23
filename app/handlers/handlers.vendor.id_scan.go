package handlers

import (
	"errors"
	"net/http"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/go-chi/chi/v5"
)

// ScanVendorIDCard godoc
//
//	 	@Summary 		Record a scan of the vendor's ID card
//		@Description	Stores the current time as the vendor's last ID card scan and returns the vendor
//		@Tags			Vendors
//		@Produce		json
//		@Success		200	{object}	database.Vendor
//		@Failure		403
//		@Failure		404
//		@Security		KeycloakAuth
//	    @Param          licenseID   path string  true  "License ID"
//		@Router			/vendors/{licenseID}/id-scan/ [post]
func ScanVendorIDCard(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")

	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if !settings.IDScanEnabled {
		utils.ErrorJSON(w, errors.New("ID card scanning is disabled"), http.StatusForbidden)
		return
	}

	vendor, err := database.Db.RecordVendorIDScan(licenseID)
	if errors.Is(err, database.ErrVendorNotFound) {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error("ScanVendorIDCard: ", err)
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	log.Info(r.Header.Get("X-Auth-User-Name") + " scanned the ID card of vendor " + licenseID)
	respond(w, nil, vendor)
}

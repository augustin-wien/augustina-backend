package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/go-chi/chi/v5"
)

// ListVendorLocations godoc
//
// @Summary List vendor locations
// @Description List vendor locations
// @ID listVendorLocations
// @Produce json
// @Success 200 {array} Location
// @Router /api/vendors/locations/ [get]
// @Security KeycloakAuth

func ListVendorLocations(w http.ResponseWriter, r *http.Request) {
	vendorID := chi.URLParam(r, "vendorid")
	if vendorID == "" {
		utils.ErrorJSON(w, fmt.Errorf("vendorId is required"), http.StatusBadRequest)
		return
	}
	vendorIDInt, err := strconv.Atoi(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	locations, err := database.Db.GetLocationsByVendorID(vendorIDInt)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, locations)
}

// CreateVendorLocation godoc
//
// @Summary Create vendor location
// @Description Create vendor location
// @ID createVendorLocation
// @Produce json
// @Router /api/vendors/locations/{id}/ [post]
// @Security KeycloakAuth

func CreateVendorLocation(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		log.Error("CreateVendorLocation: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var location ent.Location
	err = utils.ReadJSON(w, r, &location)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.CreateLocation(vendorID, location)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

// UpdateVendorLocation godoc
//
// @Summary Update vendor location
// @Description Update vendor location
// @ID updateVendorLocation
// @Produce json
// @Router /api/vendors/locations/{id}/ [put]
// @Security KeycloakAuth

func UpdateVendorLocation(w http.ResponseWriter, r *http.Request) {
	_, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		log.Error("CreateVendorLocation: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var location ent.Location
	err = utils.ReadJSON(w, r, &location)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.UpdateLocation(location)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

// DeleteVendorLocation godoc
//
// @Summary Delete vendor location
// @Description Delete vendor location
// @ID deleteVendorLocation
// @Produce json
// @Router /api/vendors/locations/{id}/ [delete]
// @Security KeycloakAuth
func DeleteVendorLocation(w http.ResponseWriter, r *http.Request) {
	_, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		log.Error("DeleteVendorLocation: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	locationID, err := strconv.Atoi(chi.URLParam(r, "locationID"))
	if err != nil {
		log.Error("DeleteVendorLocation: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.DeleteLocation(locationID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

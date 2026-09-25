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
// @Router /api/vendors/{vendorid}/locations/ [get]
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

// ListAllVendorLocations godoc
//
// @Summary List all locations with their vendor
// @Description List the locations of all vendors that are not deleted, ordered by license ID
// @ID listAllVendorLocations
// @Produce json
// @Success 200 {array} database.LocationOverview
// @Router /api/locations/ [get]
// @Security KeycloakAuth
func ListAllVendorLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := database.Db.ListLocationsWithVendor()
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
// @Router /api/vendors/{vendorid}/locations/{id}/ [patch]
// @Security KeycloakAuth

func UpdateVendorLocation(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		log.Error("UpdateVendorLocation: Can not read vendor ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	locationID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("UpdateVendorLocation: Can not read location ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var location ent.Location
	err = utils.ReadJSON(w, r, &location)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	// The URL is authoritative for which location is updated
	location.ID = locationID
	err = database.Db.UpdateLocation(vendorID, location)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
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
// @Router /api/vendors/{vendorid}/locations/{id}/ [delete]
// @Security KeycloakAuth
func DeleteVendorLocation(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "vendorid"))
	if err != nil {
		log.Error("DeleteVendorLocation: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	locationID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error("DeleteVendorLocation: Can not read ID ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.DeleteLocation(vendorID, locationID)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

// locationRequest is a location plus the vendor it should belong to; a null
// or missing vendorID leaves the location unassigned
type locationRequest struct {
	ent.Location
	VendorID *int `json:"vendorID"`
}

// readLocationRequest decodes the body and checks that the vendor, if any,
// exists and isn't deleted. It writes the error response itself and returns
// false when the request can't be used.
func readLocationRequest(w http.ResponseWriter, r *http.Request) (req locationRequest, ok bool) {
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return req, false
	}
	if req.VendorID != nil {
		vendor, err := database.Db.GetVendorSimple(*req.VendorID)
		if err != nil || vendor.IsDeleted {
			utils.ErrorJSON(w, fmt.Errorf("vendor %d not found", *req.VendorID), http.StatusBadRequest)
			return req, false
		}
	}
	return req, true
}

// CreateLocation godoc
//
// @Summary Create a location
// @Description Create a location, optionally assigned to a vendor
// @ID createLocation
// @Accept json
// @Produce json
// @Success 200 {integer} int "ID of the new location"
// @Router /api/locations/ [post]
// @Security KeycloakAuth
func CreateLocation(w http.ResponseWriter, r *http.Request) {
	req, ok := readLocationRequest(w, r)
	if !ok {
		return
	}
	id, err := database.Db.CreateStandaloneLocation(req.VendorID, req.Location)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, id)
}

// UpdateLocation godoc
//
// @Summary Update a location
// @Description Update a location and assign it to a vendor, or unassign it with a null vendorID
// @ID updateLocation
// @Accept json
// @Produce json
// @Router /api/locations/{id}/ [patch]
// @Security KeycloakAuth
func UpdateLocation(w http.ResponseWriter, r *http.Request) {
	locationID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	req, ok := readLocationRequest(w, r)
	if !ok {
		return
	}
	// The URL is authoritative for which location is updated
	req.Location.ID = locationID
	err = database.Db.UpdateLocationByID(req.VendorID, req.Location)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

// DeleteLocation godoc
//
// @Summary Delete a location
// @Description Delete a location, whichever vendor it belongs to
// @ID deleteLocation
// @Router /api/locations/{id}/ [delete]
// @Security KeycloakAuth
func DeleteLocation(w http.ResponseWriter, r *http.Request) {
	locationID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.DeleteLocationByID(locationID)
	if ent.IsNotFound(err) {
		utils.ErrorJSON(w, err, http.StatusNotFound)
		return
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	respond(w, err, nil)
}

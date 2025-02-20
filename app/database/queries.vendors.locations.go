package database

import (
	"context"

	"github.com/augustin-wien/augustina-backend/ent"
	entlocation "github.com/augustin-wien/augustina-backend/ent/location"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
)

// GetLocationsByVendorID fetches all locations associated with a given vendor ID.
func (db *Database) GetLocationsByVendorID(vendorID int) (locations []*ent.Location, err error) {

	locations, err = db.EntClient.Location.Query().Where(entlocation.HasVendorWith(entvendor.ID(vendorID))).All(context.Background())
	if err != nil {
		log.Error("GetLocationsByVendorID", err)
		return nil, err
	}
	return locations, nil
}

// CreateLocation creates a new location for a given vendor.
func (db *Database) CreateLocation(vendorID int, location ent.Location) (err error) {
	_, err = db.EntClient.Location.Create().SetVendorID(vendorID).SetName(location.Name).SetAddress(location.Address).SetLongitude(location.Longitude).SetLatitude(location.Latitude).SetZip(location.Zip).SetWorkingTime(location.WorkingTime).Save(context.Background())
	if err != nil {
		log.Error("CreateLocation", err)
	}
	return err
}

// UpdateLocation updates a location.
func (db *Database) UpdateLocation(location ent.Location) (err error) {
	_, err = db.EntClient.Location.UpdateOneID(location.ID).SetName(location.Name).SetAddress(location.Address).SetLongitude(location.Longitude).SetLatitude(location.Latitude).SetZip(location.Zip).SetWorkingTime(location.WorkingTime).Save(context.Background())
	if err != nil {
		log.Error("UpdateLocation", err)
	}
	return err
}

// DeleteLocation deletes a location.
func (db *Database) DeleteLocation(locationID int) (err error) {
	err = db.EntClient.Location.DeleteOneID(locationID).Exec(context.Background())
	if err != nil {
		log.Error("DeleteLocation", err)
	}
	return err
}

// Online Map -----------------------------------------------------------------

// LocationData is used to return the location data of a vendor for the online map
type LocationData struct {
	ID        int     `json:"id"`
	FirstName string  `json:"firstName"`
	LicenseID string  `json:"licenseID"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// GetVendorLocations returns a list of all longitudes and latitudes given by the vendors table
func (db *Database) GetVendorLocations() (locationData []LocationData, err error) {

	Locations, err := db.EntClient.Location.Query().WithVendor().All(context.Background())
	if err != nil {
		log.Error("GetVendorLocations: ", err)
		return locationData, err
	}
	for _, location := range Locations {
		locationData = append(locationData, LocationData{
			ID:        location.Edges.Vendor.ID,
			FirstName: location.Edges.Vendor.Firstname,
			LicenseID: location.Edges.Vendor.Licenseid,
			Longitude: location.Longitude,
			Latitude:  location.Latitude,
		})
	}
	return locationData, nil
}

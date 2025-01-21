package database

import (
	"context"

	"github.com/augustin-wien/augustina-backend/ent"
	entlocation "github.com/augustin-wien/augustina-backend/ent/location"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
	"gopkg.in/guregu/null.v4"
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
	ID        int         `json:"id"`
	FirstName string      `json:"firstName"`
	LicenseID null.String `json:"licenseID"`
	Longitude float64     `json:"longitude"`
	Latitude  float64     `json:"latitude"`
}

// GetVendorLocations returns a list of all longitudes and latitudes given by the vendors table
func (db *Database) GetVendorLocations() (locationData []LocationData, err error) {
	rows, err := db.Dbpool.Query(context.Background(), `
	SELECT Locations.Longitude, Locations.Latitude, Vendor, Vendor.FirstName, Vendor.LicenseID
		from Locations
		Join Vendor ON Vendor.id = Locations.vendor_locations
		where Locations.longitude != 0 and Locations.longitude != 0.1;
	`)
	if err != nil {
		log.Error("GetVendorLocations: ", err)
		return locationData, err
	}
	for rows.Next() {
		var nextLocationData LocationData
		err = rows.Scan(&nextLocationData.ID, &nextLocationData.LicenseID, &nextLocationData.FirstName, &nextLocationData.Longitude, &nextLocationData.Latitude)
		if err != nil {
			log.Error("GetVendorLocations: ", err)
			return locationData, err
		}
		locationData = append(locationData, nextLocationData)
	}
	return locationData, nil
}

package database

import (
	"context"

	"github.com/augustin-wien/augustina-backend/ent"
	entlocation "github.com/augustin-wien/augustina-backend/ent/location"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
	entworkingtime "github.com/augustin-wien/augustina-backend/ent/workingtime"
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

// WorkingTimeInput represents a simplified working time payload.
type WorkingTimeInput struct {
	Day      string  `json:"day"`
	OpenTime string  `json:"open_time"`
	CloseTime *string `json:"close_time"`
	Closed   bool    `json:"closed"`
}

// LocationInput represents incoming location payload used to create/update locations.
type LocationInput struct {
	ID           int                 `json:"id"`
	Name         string              `json:"name"`
	Address      string              `json:"address"`
	Longitude    float64             `json:"longitude"`
	Latitude     float64             `json:"latitude"`
	Zip          string              `json:"zip"`
	WorkingTime  string              `json:"working_time"`
	WorkingTimes []WorkingTimeInput  `json:"working_times"`
}

// CreateLocation creates a new location for a given vendor and attaches working_times if provided.
func (db *Database) CreateLocation(vendorID int, payload LocationInput) (err error) {
	loc, err := db.EntClient.Location.Create().SetVendorID(vendorID).SetName(payload.Name).SetAddress(payload.Address).SetLongitude(payload.Longitude).SetLatitude(payload.Latitude).SetZip(payload.Zip).Save(context.Background())
	if err != nil {
		log.Error("CreateLocation", err)
		return err
	}

	// If a legacy working_time string is provided, convert it into one working_times entry.
	if payload.WorkingTime != "" {
		_, err = db.EntClient.WorkingTime.Create().SetDay(entworkingtime.DayMonday).SetOpenTime(payload.WorkingTime).SetLocationID(loc.ID).Save(context.Background())
		if err != nil {
			log.Error("CreateLocation: create working_time", err)
			return err
		}
	}

	// If explicit working_times provided, create them.
	for _, wt := range payload.WorkingTimes {
		w := db.EntClient.WorkingTime.Create().SetDay(entworkingtime.Day(wt.Day)).SetOpenTime(wt.OpenTime).SetLocationID(loc.ID).SetClosed(wt.Closed)
		if wt.CloseTime != nil {
			w = w.SetCloseTime(*wt.CloseTime)
		}
		_, err = w.Save(context.Background())
		if err != nil {
			log.Error("CreateLocation: create working_time batch", err)
			return err
		}
	}

	return nil
}

// UpdateLocation updates a location and updates working_times when provided.
func (db *Database) UpdateLocation(payload LocationInput) (err error) {
	_, err = db.EntClient.Location.UpdateOneID(payload.ID).SetName(payload.Name).SetAddress(payload.Address).SetLongitude(payload.Longitude).SetLatitude(payload.Latitude).SetZip(payload.Zip).Save(context.Background())
	if err != nil {
		log.Error("UpdateLocation", err)
		return err
	}

	// If legacy working_time string provided, create/update a default working_time row.
	if payload.WorkingTime != "" {
		// For simplicity: always create a new default working_time linked to location.
		_, err = db.EntClient.WorkingTime.Create().SetDay(entworkingtime.DayMonday).SetOpenTime(payload.WorkingTime).SetLocationID(payload.ID).Save(context.Background())
		if err != nil {
			log.Error("UpdateLocation: create working_time", err)
			return err
		}
	}

	// If explicit working_times provided, append them. A more complete implementation would sync (insert/update/delete).
	for _, wt := range payload.WorkingTimes {
		w := db.EntClient.WorkingTime.Create().SetDay(entworkingtime.Day(wt.Day)).SetOpenTime(wt.OpenTime).SetLocationID(payload.ID).SetClosed(wt.Closed)
		if wt.CloseTime != nil {
			w = w.SetCloseTime(*wt.CloseTime)
		}
		_, err = w.Save(context.Background())
		if err != nil {
			log.Error("UpdateLocation: create working_time batch", err)
			return err
		}
	}

	return nil
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

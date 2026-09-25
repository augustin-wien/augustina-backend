package database

import (
	"context"
	"sort"

	"github.com/augustin-wien/augustina-backend/ent"
	entlocation "github.com/augustin-wien/augustina-backend/ent/location"
	"github.com/augustin-wien/augustina-backend/ent/schema"
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
	_, err = db.EntClient.Location.Create().SetVendorID(vendorID).SetName(location.Name).SetAddress(location.Address).SetLongitude(location.Longitude).SetLatitude(location.Latitude).SetZip(location.Zip).SetTelephone(location.Telephone).SetWorkingTime(location.WorkingTime).Save(context.Background())
	if err != nil {
		log.Error("CreateLocation", err)
	}
	return err
}

// UpdateLocation updates a location belonging to the given vendor. Returns an
// ent.NotFoundError if the location does not exist or belongs to another vendor.
func (db *Database) UpdateLocation(vendorID int, location ent.Location) (err error) {
	_, err = db.EntClient.Location.UpdateOneID(location.ID).Where(entlocation.HasVendorWith(entvendor.ID(vendorID))).SetName(location.Name).SetAddress(location.Address).SetLongitude(location.Longitude).SetLatitude(location.Latitude).SetZip(location.Zip).SetTelephone(location.Telephone).SetWorkingTime(location.WorkingTime).Save(context.Background())
	if err != nil && !ent.IsNotFound(err) {
		log.Error("UpdateLocation", err)
	}
	return err
}

// DeleteLocation deletes a location belonging to the given vendor. Returns an
// ent.NotFoundError if the location does not exist or belongs to another vendor.
func (db *Database) DeleteLocation(vendorID int, locationID int) (err error) {
	err = db.EntClient.Location.DeleteOneID(locationID).Where(entlocation.HasVendorWith(entvendor.ID(vendorID))).Exec(context.Background())
	if err != nil && !ent.IsNotFound(err) {
		log.Error("DeleteLocation", err)
	}
	return err
}

// Online Map -----------------------------------------------------------------

// LocationData is used to return the location data of a vendor for the online map.
// Locations without a vendor have ID 0 and HasVendor false.
type LocationData struct {
	ID           int     `json:"id"`
	FirstName    string  `json:"firstName"`
	LicenseID    string  `json:"licenseID"`
	Longitude    float64 `json:"longitude"`
	Latitude     float64 `json:"latitude"`
	LocationID   int     `json:"locationID"`
	LocationName string  `json:"locationName"`
	Address      string  `json:"address"`
	HasVendor    bool    `json:"hasVendor"`
}

// GetVendorLocations returns a list of all longitudes and latitudes given by the vendors table
func (db *Database) GetVendorLocations() (locationData []LocationData, err error) {

	Locations, err := db.EntClient.Location.Query().WithVendor().All(context.Background())
	if err != nil {
		log.Error("GetVendorLocations: ", err)
		return locationData, err
	}
	for _, location := range Locations {
		data := LocationData{
			Longitude:    location.Longitude,
			Latitude:     location.Latitude,
			LocationID:   location.ID,
			LocationName: location.Name,
			Address:      location.Address,
		}
		if v := location.Edges.Vendor; v != nil {
			data.ID = v.ID
			data.FirstName = v.Firstname
			data.LicenseID = v.Licenseid
			data.HasVendor = true
		}
		locationData = append(locationData, data)
	}
	return locationData, nil
}

// Location Overview ----------------------------------------------------------

// LocationOverview is a location together with the vendor it belongs to, used
// for the backoffice table of all locations. VendorID is nil for a location
// that is not assigned to any vendor.
type LocationOverview struct {
	ID               int                 `json:"id"`
	Name             string              `json:"name"`
	Address          string              `json:"address"`
	Zip              string              `json:"zip"`
	Telephone        string              `json:"telephone"`
	Longitude        float64             `json:"longitude"`
	Latitude         float64             `json:"latitude"`
	WorkingTime      *schema.WorkingTime `json:"working_time"`
	VendorID         *int                `json:"vendorID"`
	VendorLicenseID  string              `json:"vendorLicenseID"`
	VendorFirstName  string              `json:"vendorFirstName"`
	VendorLastName   string              `json:"vendorLastName"`
	VendorTelephone  string              `json:"vendorTelephone"`
	VendorIsDisabled bool                `json:"vendorIsDisabled"`
	VendorIsBlocked  bool                `json:"vendorIsBlocked"`
}

// ListLocationsWithVendor returns all locations without a vendor or of a
// vendor that is not deleted, ordered by the vendor's license ID with the
// unassigned locations last
func (db *Database) ListLocationsWithVendor() (locations []LocationOverview, err error) {
	ents, err := db.EntClient.Location.Query().
		Where(entlocation.Or(
			entlocation.Not(entlocation.HasVendor()),
			entlocation.HasVendorWith(entvendor.Isdeleted(false)),
		)).
		WithVendor().
		All(context.Background())
	if err != nil {
		log.Error("ListLocationsWithVendor: ", err)
		return nil, err
	}

	locations = make([]LocationOverview, 0, len(ents))
	for _, l := range ents {
		overview := LocationOverview{
			ID:          l.ID,
			Name:        l.Name,
			Address:     l.Address,
			Zip:         l.Zip,
			Telephone:   l.Telephone,
			Longitude:   l.Longitude,
			Latitude:    l.Latitude,
			WorkingTime: l.WorkingTime,
		}
		if v := l.Edges.Vendor; v != nil {
			overview.VendorID = &v.ID
			overview.VendorLicenseID = v.Licenseid
			overview.VendorFirstName = v.Firstname
			overview.VendorLastName = v.Lastname
			overview.VendorTelephone = v.Telephone
			overview.VendorIsDisabled = v.Isdisabled
			overview.VendorIsBlocked = v.Isblocked
		}
		locations = append(locations, overview)
	}
	sort.SliceStable(locations, func(i, j int) bool {
		a, b := locations[i], locations[j]
		if (a.VendorID == nil) != (b.VendorID == nil) {
			return b.VendorID == nil
		}
		if a.VendorLicenseID != b.VendorLicenseID {
			return a.VendorLicenseID < b.VendorLicenseID
		}
		return a.Name < b.Name
	})
	return locations, nil
}

// CreateStandaloneLocation creates a location that is assigned to the given
// vendor, or to no vendor at all if vendorID is nil
func (db *Database) CreateStandaloneLocation(vendorID *int, location ent.Location) (id int, err error) {
	l, err := db.EntClient.Location.Create().
		SetNillableVendorID(vendorID).
		SetName(location.Name).
		SetAddress(location.Address).
		SetLongitude(location.Longitude).
		SetLatitude(location.Latitude).
		SetZip(location.Zip).
		SetTelephone(location.Telephone).
		SetWorkingTime(location.WorkingTime).
		Save(context.Background())
	if err != nil {
		log.Error("CreateStandaloneLocation: ", err)
		return 0, err
	}
	return l.ID, nil
}

// UpdateLocationByID updates a location and moves it to the given vendor, or
// unassigns it if vendorID is nil. Returns an ent.NotFoundError if the location
// does not exist.
func (db *Database) UpdateLocationByID(vendorID *int, location ent.Location) (err error) {
	update := db.EntClient.Location.UpdateOneID(location.ID).
		SetName(location.Name).
		SetAddress(location.Address).
		SetLongitude(location.Longitude).
		SetLatitude(location.Latitude).
		SetZip(location.Zip).
		SetTelephone(location.Telephone).
		SetWorkingTime(location.WorkingTime)
	if vendorID != nil {
		update = update.SetVendorID(*vendorID)
	} else {
		update = update.ClearVendor()
	}
	_, err = update.Save(context.Background())
	if err != nil && !ent.IsNotFound(err) {
		log.Error("UpdateLocationByID: ", err)
	}
	return err
}

// DeleteLocationByID deletes a location regardless of its vendor. Returns an
// ent.NotFoundError if the location does not exist.
func (db *Database) DeleteLocationByID(locationID int) (err error) {
	err = db.EntClient.Location.DeleteOneID(locationID).Exec(context.Background())
	if err != nil && !ent.IsNotFound(err) {
		log.Error("DeleteLocationByID: ", err)
	}
	return err
}

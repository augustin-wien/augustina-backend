package database

import (
	"context"
	"fmt"

	"github.com/augustin-wien/augustina-backend/ent"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
	"github.com/augustin-wien/augustina-backend/utils"
	"gopkg.in/guregu/null.v4"
)

// GetVendorByLicenseID returns the vendor with the given licenseID
func (db *Database) GetVendorByLicenseID(licenseID string) (vendor Vendor, err error) {
	// Get vendor data
	v, err := db.EntClient.Vendor.Query().Where(entvendor.Licenseid(licenseID)).First(context.Background())
	if err != nil {
		return vendor, fmt.Errorf("GetVendorByLicenseID: Couldn't get vendor: %w", err)
	}
	vendor = db.VendorEntIntoVendor(*v)

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendorByLicenseID: couldn't get balance: ", err)
	}

	vendor, err = db.GetAdditionalVendorData(vendor)
	if err != nil {
		log.Error("GetVendorByLicenseID: couldn't get additional vendor data: ", err)
		return vendor, err
	}

	return vendor, nil

}

// ListVendors returns all users from the database but not all fields for better overview
func (db *Database) ListVendors() (vendors []Vendor, err error) {
	rows, err := db.Dbpool.Query(context.Background(), `
		SELECT vendor.ID, LicenseID, FirstName, LastName, LastPayout, Balance, IsDisabled
		FROM Vendor 
		JOIN Account ON Account.vendor = Vendor.id 
		WHERE Account.Type = 'Vendor' and IsDeleted = false
		ORDER BY LicenseID ASC
	`)
	if err != nil {
		log.Error("ListVendors", err)
		return vendors, err
	}
	defer rows.Close()

	for rows.Next() {
		var vendor Vendor
		err = rows.Scan(&vendor.ID, &vendor.LicenseID, &vendor.FirstName, &vendor.LastName, &vendor.LastPayout, &vendor.Balance, &vendor.IsDisabled)
		if err != nil {
			log.Error("ListVendors", err)
			return vendors, err
		}
		vendors = append(vendors, vendor)
	}

	return vendors, nil
}

// GetVendorByLicenseID returns the vendor with the given licenseID
func (db *Database) GetVendorByLicenseIDWithoutDisabled(licenseID string) (vendor Vendor, err error) {
	v, err := db.EntClient.Vendor.Query().Where(entvendor.Licenseid(licenseID)).Where(entvendor.Isdisabled(false)).First(context.Background())
	if err != nil {
		return vendor, fmt.Errorf("GetVendorByLicenseIDWithoutDisable: Couldn't get vendor: %w", err)
	}
	vendor = db.VendorEntIntoVendor(*v)

	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendorByLicenseIDWithoutDisable: couldn't get balance: ", err)
	}
	vendor, err = db.GetAdditionalVendorData(vendor)
	if err != nil {
		log.Error("GetVendorByLicenseIDWithoutDisable: couldn't get additional vendor data: ", err)
		return vendor, err
	}

	return vendor, nil
}

// GetVendorByEmail returns the vendor with the given licenseID
func (db *Database) GetVendorByEmail(mail string) (vendor Vendor, err error) {
	mail = utils.ToLower(mail)
	// Get vendor data
	ctx := context.Background()
	// Get vendor data
	v, err := db.EntClient.Vendor.Query().Where(entvendor.Email(mail)).First(ctx)
	if err != nil {
		return vendor, fmt.Errorf("GetVendorByEmail: Couldn't get vendor: %w", err)
	}
	vendor = db.VendorEntIntoVendor(*v)
	// Get vendor balance
	err = db.Dbpool.QueryRow(context.Background(), "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendorByEmail: Couldn't get balance: ", err)
	}
	vendor, err = db.GetAdditionalVendorData(vendor)
	if err != nil {
		log.Error("GetVendorByLicenseID: couldn't get additional vendor data: ", err)
		return vendor, err
	}

	return vendor, nil
}

// GetVendorSimple returns the vendor with the given vendorID
func (db *Database) GetVendorSimple(vendorID int) (vendor Vendor, err error) {
	ctx := context.Background()
	// Get vendor data
	v, err := db.EntClient.Vendor.Get(ctx, vendorID)
	if err != nil {
		return vendor, fmt.Errorf("failed querying user: %w", err)
	}
	vendor = db.VendorEntIntoVendor(*v)
	// Get vendor balance
	err = db.Dbpool.QueryRow(ctx, "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendor: couldn't get vendor ", err)
	}
	return vendor, nil
}

// GetVendor returns the vendor with the given id
func (db *Database) GetVendor(vendorID int) (vendor Vendor, err error) {
	ctx := context.Background()
	// Get vendor data
	v, err := db.EntClient.Vendor.Get(ctx, vendorID)
	if err != nil {
		return vendor, fmt.Errorf("failed querying user: %w", err)
	}
	vendor = db.VendorEntIntoVendor(*v)
	// Get vendor balance
	err = db.Dbpool.QueryRow(ctx, "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendor: couldn't get vendor ", err)
	}
	return vendor, nil
}

// GetVendorWithBalanceUpdate returns the vendor with the given id
func (db *Database) GetVendorWithBalanceUpdate(vendorID int) (vendor Vendor, err error) {
	ctx := context.Background()

	// Update Account balance by open payments
	_, err = db.UpdateAccountBalanceByOpenPayments(vendorID)
	if err != nil {
		log.Error("GetVendorWithBalanceUpdate: ", err)
	}

	// Get vendor data

	v, err := db.EntClient.Vendor.Get(ctx, vendorID)
	if err != nil {
		return vendor, fmt.Errorf("failed querying user: %w", err)
	}
	vendor = db.VendorEntIntoVendor(*v)

	// Get vendor balance
	err = db.Dbpool.QueryRow(ctx, "SELECT Balance FROM Account WHERE Vendor = $1", vendor.ID).Scan(&vendor.Balance)
	if err != nil {
		log.Error("GetVendorWithBalanceUpdate: Couldn't get balance ", err)
	}

	return vendor, nil
}

// CreateVendor creates a vendor and an associated account in the database
func (db *Database) CreateVendor(vendor Vendor) (vendorID int, err error) {
	vendor.Email = utils.ToLower(vendor.Email)
	// Create vendor
	v, err := db.EntClient.Vendor.Create().
		SetAccountproofurl(vendor.AccountProofUrl.String).
		SetEmail(vendor.Email).
		SetFirstname(vendor.FirstName).
		SetHasbankaccount(vendor.HasBankAccount).
		SetHassmartphone(vendor.HasSmartphone).
		SetIsdeleted(vendor.IsDeleted).
		SetIsdisabled(vendor.IsDisabled).
		SetKeycloakid(vendor.KeycloakID).
		SetLanguage(vendor.Language).
		SetLastname(vendor.LastName).
		SetLastpayout(vendor.LastPayout.Time).
		SetLicenseid(vendor.LicenseID.String).
		SetOnlinemap(vendor.OnlineMap).
		SetRegistrationdate(vendor.RegistrationDate).
		SetTelephone(vendor.Telephone).
		SetVendorsince(vendor.VendorSince).
		SetUrlid(vendor.UrlID).
		SetDebt(vendor.Debt).
		Save(context.Background())
	if err != nil {
		log.Error("CreateVendor: create vendor %s %+v", vendor.Email, err)
		return
	}
	vendorID = v.ID
	log.Info("CreateVendor: created vendor %v", vendorID)

	// Create vendor account
	_, err = db.Dbpool.Exec(context.Background(), "INSERT INTO Account (Name, Balance, Type, Vendor) values ($1, 0, $2, $3) RETURNING ID", vendor.LicenseID, "Vendor", v.ID)
	if err != nil {
		log.Error("CreateVendor: create vendor account %s %+v", vendor.Email, err)
		return
	}
	log.Info("CreateVendor: created vendor %s", vendor.Email)

	return
}

// UpdateVendor updates a vendor in the database
func (db *Database) UpdateVendor(id int, vendor Vendor) (err error) {
	vendor.Email = utils.ToLower(vendor.Email)
	ctx := context.Background()
	v := db.VendorIntoVendorEnt(vendor)
	_, err = db.EntClient.Vendor.UpdateOneID(id).SetAccountproofurl(v.Accountproofurl).SetEmail(v.Email).SetFirstname(v.Firstname).SetHasbankaccount(v.Hasbankaccount).SetHassmartphone(v.Hassmartphone).SetIsdeleted(v.Isdeleted).SetIsdisabled(v.Isdisabled).SetKeycloakid(v.Keycloakid).SetLanguage(v.Language).SetLastname(v.Lastname).SetLastpayout(v.Lastpayout).SetLicenseid(v.Licenseid).SetOnlinemap(v.Onlinemap).SetRegistrationdate(v.Registrationdate).SetTelephone(v.Telephone).SetUrlid(v.Urlid).SetDebt(v.Debt).Save(ctx)

	return err
}

// DeleteVendor deletes a user in the database and the associated account
func (db *Database) DeleteVendor(vendorID int) (err error) { 
	
	ctx := context.Background()
	v, err := db.EntClient.Vendor.Get(ctx, vendorID)
	if err != nil {
		log.Error("DeleteVendor get vendor: ", err)
		return
	}

    name := utils.RandomString(10)
	_, err = db.EntClient.Vendor.Update().
		Where(entvendor.ID(vendorID)).
		SetIsdeleted(true). // Update the isdeleted field
		SetLicenseid("del_"+v.Licenseid+"_"+name).
		Save(ctx)
	if err != nil {
		log.Error("DeleteVendor: ", err)
	}

	return
}

func (db *Database) GetAdditionalVendorData(vendor Vendor) (Vendor, error) {
	var err error
	// Fetch vendor locations using a separate function

	vendor.Locations, err = db.GetLocationsByVendorID(vendor.ID)
	if err != nil {
		log.Error("GetVendorByLicenseID: couldn't get locations: ", err)
		return vendor, err
	}
	vendor.Comments, err = db.GetVendorComments(vendor.ID)
	if err != nil {
		log.Error("GetVendorByLicenseID: couldn't get comments: ", err)
		return vendor, err
	}
	return vendor, nil

}

func (db *Database) VendorEntIntoVendor(v ent.Vendor) (vendor Vendor) {
	vendor = Vendor{
		ID:               v.ID,
		AccountProofUrl:  null.StringFrom(v.Accountproofurl),
		KeycloakID:       v.Keycloakid,
		UrlID:            v.Urlid,
		LicenseID:        null.StringFrom(v.Licenseid),
		FirstName:        v.Firstname,
		LastName:         v.Lastname,
		Email:            v.Email,
		LastPayout:       null.TimeFrom(v.Lastpayout),
		IsDisabled:       v.Isdisabled,
		IsDeleted:        v.Isdeleted,
		Locations:        v.Edges.Locations,
		Comments:         v.Edges.Comments,
		Language:         v.Language,
		Telephone:        v.Telephone,
		RegistrationDate: v.Registrationdate,
		VendorSince:      v.Vendorsince,
		OnlineMap:        v.Onlinemap,
		HasSmartphone:    v.Hassmartphone,
		HasBankAccount:   v.Hasbankaccount,
		Debt:             v.Debt,
	}
	return vendor
}

func (db *Database) VendorIntoVendorEnt(vendor Vendor) (v *ent.Vendor) {
	v = &ent.Vendor{
		ID:              vendor.ID,
		Accountproofurl: vendor.AccountProofUrl.String,
		Keycloakid:      vendor.KeycloakID,
		Urlid:           vendor.UrlID,
		Licenseid:       vendor.LicenseID.String,
		Firstname:       vendor.FirstName,
		Lastname:        vendor.LastName,
		Email:           vendor.Email,
		Lastpayout:      vendor.LastPayout.Time,
		Isdisabled:      vendor.IsDisabled,
		Isdeleted:       vendor.IsDeleted,
		// Locations:        vendor.Edges.Locations,
		// Comments:         vendor.Edges.Comments,
		Language:         vendor.Language,
		Telephone:        vendor.Telephone,
		Registrationdate: vendor.RegistrationDate,
		Vendorsince:      vendor.VendorSince,
		Onlinemap:        vendor.OnlineMap,
		Hassmartphone:    vendor.HasSmartphone,
		Hasbankaccount:   vendor.HasBankAccount,
		Debt:             vendor.Debt,
	}
	return v
}

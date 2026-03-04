package database

import (
	"context"
	"fmt"

	"github.com/augustin-wien/augustina-backend/ent"
	entaccount "github.com/augustin-wien/augustina-backend/ent/account"
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
	acc, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendor.ID)).First(context.Background())
	if err == nil {
		vendor.Balance = int(acc.Balance)
	} else {
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
	ctx := context.Background()
	// Get vendors with accounts of type 'Vendor'
	// The original query did a JOIN and filtered by Account.Type = 'Vendor'
	// This implies we only want vendors that HAVE such an account,
	// AND we only care about THAT account's balance.
	// We load only the relevant account.
	ents, err := db.EntClient.Vendor.Query().
		Where(entvendor.Isdisabled(false)). // Query had IsDeleted=false, but comment says IsDisabled? No, query says IsDeleted=false.
		Where(entvendor.Isdeleted(false)).
		Where(entvendor.HasAccountsWith(entaccount.Type("Vendor"))).
		WithAccounts(func(q *ent.AccountQuery) {
			q.Where(entaccount.Type("Vendor"))
		}).
		Order(ent.Asc(entvendor.FieldLicenseid)).
		All(ctx)

	if err != nil {
		log.Error("ListVendors", err)
		return vendors, err
	}

	for _, e := range ents {
		v := db.VendorEntIntoVendor(*e)
		// Set balance from the loaded account (there should be exactly one per vendor of type 'Vendor')
		if len(e.Edges.Accounts) > 0 {
			v.Balance = int(e.Edges.Accounts[0].Balance)
		}

		// The original query selected: ID, LicenseID, FirstName, LastName, LastPayout, Balance, IsDisabled
		// VendorEntIntoVendor copies all fields, so that's fine.
		vendors = append(vendors, v)
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
	acc, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendor.ID)).First(context.Background())
	if err == nil {
		vendor.Balance = int(acc.Balance)
	} else {
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
	acc, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendor.ID)).First(ctx)
	if err == nil {
		vendor.Balance = int(acc.Balance)
	} else {
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
	acc, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendor.ID)).First(ctx)
	if err == nil {
		vendor.Balance = int(acc.Balance)
	} else {
		log.Error("GetVendorSimple: couldn't get vendor balance ", err)
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
	acc, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendor.ID)).First(ctx)
	if err == nil {
		vendor.Balance = int(acc.Balance)
	} else {
		log.Error("GetVendor: couldn't get vendor balance ", err)
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
	acc, err := db.EntClient.Account.Query().Where(entaccount.VendorID(vendor.ID)).First(ctx)
	if err == nil {
		vendor.Balance = int(acc.Balance)
	} else {
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
		log.Errorf("CreateVendor: create vendor %s %+v", vendor.Email, err)
		return
	}
	vendorID = v.ID
	log.Info("CreateVendor: created vendor %v", vendorID)

	// Create vendor account
	_, err = db.EntClient.Account.Create().
		SetName(vendor.LicenseID.String).
		SetBalance(0).
		SetType("Vendor").
		SetVendorID(vendorID).
		Save(context.Background())
	if err != nil {
		log.Errorf("CreateVendor: create vendor account %s %+v", vendor.Email, err)
		return
	}
	log.Infof("CreateVendor: created vendor %s", vendor.Email)

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
		SetLicenseid("del_" + v.Licenseid + "_" + name).
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

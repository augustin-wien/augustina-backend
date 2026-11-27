package database

import (
	"context"
	"fmt"

	"github.com/augustin-wien/augustina-backend/ent"
)

func steckbriefFromEnt(v *ent.Vendor) VendorSteckbrief {
	return VendorSteckbrief{
		VendorID:    v.ID,
		FirstName:   v.Firstname,
		LastName:    v.Lastname,
		LicenseID:   v.Licenseid,
		Image:       v.Steckbriefimage,
		Description: v.Steckbriefdescription,
		Family:      v.Steckbrieffamily,
		Residence:   v.Steckbriefresidence,
		Education:   v.Steckbriefeducation,
		CareerGoal:  v.Steckbriefcareergoal,
		Support:     v.Steckbriefsupport,
	}
}

func (db *Database) getVendorEnt(vendorID int) (*ent.Vendor, error) {
	v, err := db.EntClient.Vendor.Get(context.Background(), vendorID)
	if ent.IsNotFound(err) || (err == nil && v.Isdeleted) {
		return nil, ErrVendorNotFound
	}
	return v, err
}

// GetVendorSteckbrief returns the Steckbrief of the vendor
func (db *Database) GetVendorSteckbrief(vendorID int) (VendorSteckbrief, error) {
	v, err := db.getVendorEnt(vendorID)
	if err != nil {
		return VendorSteckbrief{}, fmt.Errorf("GetVendorSteckbrief: %w", err)
	}
	return steckbriefFromEnt(v), nil
}

// UpdateVendorSteckbrief stores the text fields of the Steckbrief. The image and the
// read-only vendor fields are left as they are.
func (db *Database) UpdateVendorSteckbrief(vendorID int, s VendorSteckbrief) (VendorSteckbrief, error) {
	if _, err := db.getVendorEnt(vendorID); err != nil {
		return VendorSteckbrief{}, fmt.Errorf("UpdateVendorSteckbrief: %w", err)
	}
	v, err := db.EntClient.Vendor.UpdateOneID(vendorID).
		SetSteckbriefdescription(s.Description).
		SetSteckbrieffamily(s.Family).
		SetSteckbriefresidence(s.Residence).
		SetSteckbriefeducation(s.Education).
		SetSteckbriefcareergoal(s.CareerGoal).
		SetSteckbriefsupport(s.Support).
		Save(context.Background())
	if err != nil {
		return VendorSteckbrief{}, fmt.Errorf("UpdateVendorSteckbrief: %w", err)
	}
	return steckbriefFromEnt(v), nil
}

// SetVendorSteckbriefImage stores the path of the Steckbrief photo ("" removes it) and
// returns the updated Steckbrief together with the previous path
func (db *Database) SetVendorSteckbriefImage(vendorID int, path string) (s VendorSteckbrief, previous string, err error) {
	old, err := db.getVendorEnt(vendorID)
	if err != nil {
		return s, "", fmt.Errorf("SetVendorSteckbriefImage: %w", err)
	}
	v, err := db.EntClient.Vendor.UpdateOneID(vendorID).
		SetSteckbriefimage(path).
		Save(context.Background())
	if err != nil {
		return s, "", fmt.Errorf("SetVendorSteckbriefImage: %w", err)
	}
	return steckbriefFromEnt(v), old.Steckbriefimage, nil
}

// IsEmpty reports whether neither a photo nor any text has been entered
func (s VendorSteckbrief) IsEmpty() bool {
	return s.Image == "" && s.Description == "" && s.Family == "" && s.Residence == "" &&
		s.Education == "" && s.CareerGoal == "" && s.Support == ""
}

package database

import (
	"context"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	schema "github.com/augustin-wien/augustina-backend/ent/schema"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// createDevVendors creates one vendor per vendor state
func (db *Database) createDevVendors() (vendors devVendors, err error) {
	now := time.Now()
	create := func(v Vendor) (int, error) {
		if v.UrlID == "" {
			v.UrlID = "www.augustin.or.at/" + v.LicenseID.String
		}
		id, err := db.CreateVendor(v)
		if err != nil {
			log.Error("Dev data vendor creation failed ", zap.Error(err))
		}
		return id, err
	}

	if vendors.active, err = create(Vendor{
		KeycloakID:       "keycloakid1",
		LicenseID:        null.StringFrom("fl-123"),
		FirstName:        "firstname1",
		LastName:         "lastname1",
		Email:            "test_vendor@example.com",
		Language:         "de",
		Telephone:        "+43 660 123 45 67",
		RegistrationDate: now.AddDate(-3, 0, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-5, 0, 0).Format("2006-01-02"),
		OnlineMap:        true,
		HasSmartphone:    true,
		HasBankAccount:   true,
		AccountProofUrl:  null.StringFrom("https://example.com/account-proof/fl-123.pdf"),
	}); err != nil {
		return
	}
	if vendors.openBalance, err = create(Vendor{
		KeycloakID:       "keycloakid2",
		LicenseID:        null.StringFrom("fl-234"),
		FirstName:        "Recep",
		LastName:         "lastname2",
		Email:            "test_vendor2@example.com",
		Language:         "tr",
		RegistrationDate: now.AddDate(-1, -2, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-1, -2, 0).Format("2006-01-02"),
		OnlineMap:        true,
	}); err != nil {
		return
	}
	if vendors.negative, err = create(Vendor{
		KeycloakID:       "keycloakid3",
		LicenseID:        null.StringFrom("fl-345"),
		FirstName:        "Ioana",
		LastName:         "Popescu",
		Email:            "test_vendor3@example.com",
		Language:         "ro",
		Telephone:        "+43 699 987 65 43",
		RegistrationDate: now.AddDate(0, -8, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(0, -8, 0).Format("2006-01-02"),
		HasSmartphone:    true,
		Debt:             "25,00 € Vorschuss (Rückzahlung in Raten)",
	}); err != nil {
		return
	}
	if vendors.disabled, err = create(Vendor{
		KeycloakID:       "keycloakid4",
		LicenseID:        null.StringFrom("fl-456"),
		FirstName:        "Kofi",
		LastName:         "Mensah",
		Email:            "test_vendor4@example.com",
		Language:         "en",
		RegistrationDate: now.AddDate(-4, 0, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-4, 0, 0).Format("2006-01-02"),
		IsDisabled:       true,
	}); err != nil {
		return
	}
	if vendors.blocked, err = create(Vendor{
		KeycloakID:       "keycloakid5",
		LicenseID:        null.StringFrom("fl-567"),
		FirstName:        "Stefan",
		LastName:         "Novak",
		Email:            "test_vendor5@example.com",
		Language:         "de",
		RegistrationDate: now.AddDate(-2, 0, 0).Format("2006-01-02"),
		VendorSince:      now.AddDate(-2, 0, 0).Format("2006-01-02"),
		HasSmartphone:    true,
		HasBankAccount:   true,
	}); err != nil {
		return
	}
	if vendors.deleted, err = create(Vendor{
		KeycloakID: "keycloakid6",
		LicenseID:  null.StringFrom("fl-678"),
		FirstName:  "Gerda",
		LastName:   "Weiss",
		Email:      "test_vendor6@example.com",
		Language:   "de",
	}); err != nil {
		return
	}
	// Only the fields the vendor creation form requires
	vendors.fresh, err = create(Vendor{
		KeycloakID: "keycloakid7",
		LicenseID:  null.StringFrom("fl-789"),
		FirstName:  "Amira",
		LastName:   "Haddad",
		Email:      "test_vendor7@example.com",
	})
	return
}

// finishDevVendorStates blocks and deletes the vendors meant to end up that way
func (db *Database) finishDevVendorStates(vendors devVendors) error {
	ctx := context.Background()
	err := db.EntClient.Vendor.UpdateOneID(vendors.blocked).
		SetIsblocked(true).
		SetBlockednote("Ausweis abgelaufen, bitte neuen Ausweis vorlegen").
		Exec(ctx)
	if err != nil {
		return err
	}
	return db.EntClient.Vendor.UpdateOneID(vendors.deleted).
		SetIsdeleted(true).
		Exec(ctx)
}

func (db *Database) createDevLocations(vendors devVendors) (err error) {
	byVendor := map[int][]ent.Location{
		vendors.active: {
			{
				Name:      "Morning Market",
				Address:   "Marktplatz 1",
				Longitude: 16.3725,
				Latitude:  48.2082,
				Zip:       "1010",
				Telephone: "+43 1 512 34 56",
				WorkingTime: &schema.WorkingTime{
					Mode: "everyday",
					Everyday: []schema.TimeRange{
						{From: "08:00", To: "12:00"},
					},
				},
			},
			{
				Name:      "Evening Stand",
				Address:   "Kulturstraße 7",
				Longitude: 16.3790,
				Latitude:  48.2110,
				Zip:       "1020",
				WorkingTime: &schema.WorkingTime{
					Mode: "by_day",
					WeekDays: map[string][]schema.TimeRange{
						"mon": {{From: "09:00", To: "17:00"}},
						"tue": {{From: "09:00", To: "17:00"}},
						"wed": {{From: "09:00", To: "12:00"}, {From: "14:00", To: "18:00"}},
						"thu": {{From: "09:00", To: "17:00"}},
						"fri": {{From: "09:00", To: "17:00"}},
						"sat": {{FullDay: true}},
					},
				},
			},
		},
		vendors.openBalance: {
			{
				Name:        "Bahnhof Meidling",
				Address:     "Eichenstraße 25",
				Longitude:   16.3337,
				Latitude:    48.1745,
				Zip:         "1120",
				WorkingTime: &schema.WorkingTime{Mode: "whole_week", WholeWeek: true},
			},
		},
		vendors.negative: {
			{
				Name:      "Supermarkt Favoriten",
				Address:   "Favoritenstraße 120",
				Longitude: 16.3772,
				Latitude:  48.1823,
				Zip:       "1100",
				Telephone: "+43 699 987 65 43",
				WorkingTime: &schema.WorkingTime{
					Mode: "custom",
					WeekDays: map[string][]schema.TimeRange{
						"fri": {{From: "15:00", To: "19:00"}},
						"sat": {{From: "08:00", To: "13:00"}},
					},
				},
			},
		},
		vendors.disabled: {
			{
				Name:        "Stephansplatz",
				Address:     "Stephansplatz 3",
				Longitude:   16.3731,
				Latitude:    48.2085,
				Zip:         "1010",
				WorkingTime: &schema.WorkingTime{},
			},
		},
	}

	for vendorID, locations := range byVendor {
		for _, location := range locations {
			err = db.CreateLocation(vendorID, location)
			if err != nil {
				log.Error("Dev data location creation failed ", zap.Error(err))
				return err
			}
		}
	}

	// A location nobody is assigned to yet
	_, err = db.CreateStandaloneLocation(nil, ent.Location{
		Name:      "Westbahnhof (unbesetzt)",
		Address:   "Europaplatz 2",
		Longitude: 16.3378,
		Latitude:  48.1966,
		Zip:       "1150",
		WorkingTime: &schema.WorkingTime{
			Mode:     "everyday",
			Everyday: []schema.TimeRange{{From: "07:00", To: "10:00"}},
		},
	})
	return err
}

func (db *Database) createDevComments(vendors devVendors) (err error) {
	now := time.Now()
	var unresolved time.Time
	byVendor := map[int][]ent.Comment{
		vendors.active: {
			{
				Comment:    "Demo vendor is active and ready for use.",
				Warning:    false,
				CreatedAt:  now.Add(-72 * time.Hour),
				ResolvedAt: now.Add(-72 * time.Hour),
			},
			{
				Comment:    "Please verify the demo vendor's contract details.",
				Warning:    true,
				CreatedAt:  now.Add(-24 * time.Hour),
				ResolvedAt: now,
			},
			{
				Comment:    "Neue Fotos für den Ausweis mitbringen.",
				Warning:    false,
				CreatedAt:  now.Add(-2 * time.Hour),
				ResolvedAt: unresolved,
			},
		},
		vendors.negative: {
			{
				Comment:    "Vorschuss von 25 € ausgezahlt, Rückzahlung vereinbart.",
				Warning:    true,
				CreatedAt:  now.AddDate(0, -1, 0),
				ResolvedAt: unresolved,
			},
		},
		vendors.blocked: {
			{
				Comment:    "Ausweis abgelaufen, bis zur Verlängerung gesperrt.",
				Warning:    true,
				CreatedAt:  now.AddDate(0, 0, -3),
				ResolvedAt: unresolved,
			},
		},
	}

	for vendorID, comments := range byVendor {
		for _, comment := range comments {
			err = db.CreateVendorComment(vendorID, comment)
			if err != nil {
				log.Error("Dev data comment creation failed ", zap.Error(err))
				return err
			}
		}
	}
	return nil
}

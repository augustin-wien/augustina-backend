package database

import "go.uber.org/zap"

// CreateTestData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	db.createDevUsers()
	db.createDevSettings()
	return nil
}

func (db *Database) createDevUsers() (err error) {
	vendor := Vendor{
		KeycloakID: "keycloakid1",
		UrlID: "urlid1",
		LicenseID: "licenseid1",
		FirstName: "firstname1",
		LastName: "lastname1",
		Email: "email1",
	}
	_, err = db.CreateVendor(vendor)
	if err != nil {
		log.Error("Dev data user creation failed ", zap.Error(err))
	}
	log.Info("Dev data user creation succeeded")
	return nil
}

func (db *Database) createDevSettings() (err error) {
	settings := Settings{}
	db.UpdateSettings(settings)
	return nil
}

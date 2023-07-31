package database

import "go.uber.org/zap"

// CreateTestData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	db.createDevUsers()
	db.createDevSettings()
	return nil
}

func (db *Database) createDevUsers() (err error) {
	user := User{
		KeycloakID: "keycloakid1",
		UrlID: "urlid1",
		LicenseID: "licenseid1",
		FirstName: "firstname1",
		LastName: "lastname1",
		IsVendor: true,
		IsAdmin: true,
	}
	db.CreateUser(user)
	if err != nil {
		log.Error("Dev data user failed", zap.Error(err))
	}
	return nil
}

func (db *Database) createDevSettings() (err error) {
	settings := Settings{}
	db.UpdateSettings(settings)
	return nil
}

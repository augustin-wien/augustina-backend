package database

import (
	"reflect"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// CreateTestData creates test data for the application
func (db *Database) CreateDevData() (err error) {
	db.createDevUsers()
	db.CreateDevItems()
	return nil
}

func (db *Database) createDevUsers() (err error) {
	vendor := Vendor{
		KeycloakID: "keycloakid1",
		UrlID:      "urlid1",
		LicenseID:  "licenseid1",
		FirstName:  "firstname1",
		LastName:   "lastname1",
		Email:      "email1",
	}
	_, err = db.CreateVendor(vendor)
	if err != nil {
		log.Error("Dev data vendor creation failed ", zap.Error(err))
	}
	log.Info("Dev data vendor creation succeeded")
	return nil
}

func (db *Database) CreateDevItems() (err error) {
	newspaper := Item{
		Name:        "Zeitung",
		Description: "Aktuelle Zeitungsausgabe",
		Price:       300,
		Archived:    false,
	}

	calendar := Item{
		Name:        "Kalender",
		Description: "Kalender für das Jahr 2024",
		Price:       800,
		LicenseItem: null.IntFrom(2),
		Archived:    false,
	}

	_, err = db.CreateItem(newspaper)
	if err != nil {
		pg := err.(*pgconn.PgError)
		if reflect.TypeOf(err) == reflect.TypeOf(&pgconn.PgError{}) && pg.Code == "23505" {
			log.Info("Postgres unique_violation error detected. Details are :", pg.Detail)
			log.Info("You may want to hit 'docker compose down -v' to reset the database")
		}
		log.Info("Error type is: ", reflect.TypeOf(err))
		log.Error("Dev newspaper creation failed ", err)
		return err
	}

	_, err = db.CreateItem(calendar)
	if err != nil {
		log.Error("Dev calendar creation failed ", err)
		return err
	}
	log.Info("Dev newspaper & calendar creation succeeded")
	return err
}

func (db *Database) CreateAccountBalances() (err error) {
	return err
}

package database

import (
	"augustin/config"
	"augustin/utils"
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var log = utils.GetLogger()

// Database struct
type Database struct {
	Dbpool       *pgxpool.Pool
	IsProduction bool
}

// Db is the global database connection pool that is used by all handlers
var Db Database

// Connect to production database and store it in the global Db variable
func (db *Database) InitDb() (err error) {
	err = db.initDb(true, true)
	if err != nil {
		return err
	}
	// Check if database has been setup, variable is true in the first place and will be set false afterwards
	if !config.Config.DatabaseSetup {
		log.Info("Database already setup.")
	} else {

		err := db.InitiateSettings()
		if err != nil {
			log.Error("Settings creation failed ", zap.Error(err))
		}

		// Check if default accounts exist
		err = db.InitiateAccounts()
		if err != nil {
			log.Error("Default accounts creation failed ", zap.Error(err))
		}

		if config.Config.Development {
			err = db.CreateDevData()
			if err != nil {
				log.Error("Dev data creation failed ", zap.Error(err))
			}
		}

		// Set DATABASE_SETUP to false to prevent database setup on next startup
		// TODO make this work or figure out a better solution
		err = os.Setenv("DATABASE_SETUP", "false")
		if err != nil {
			log.Error("Unable to set variable DATABASE_SETUP to 'complete'")
		}

		log.Info("Database setup successfully completed.")
	}
	return err
}

// Connect to an empty testing database and store it in the global Db variable
func (db *Database) InitEmptyTestDb() (err error) {
	err = db.initDb(false, false)
	if err != nil {
		return err
	}
	err = db.EmptyDatabase()
	if err != nil {
		return err
	}
	db.InitiateSettings()
	db.InitiateAccounts()
	return err
}

// initDb initializes the database connection pool and stores it in the global Db variable
func (db *Database) initDb(isProduction bool, logInfo bool) (err error) {

	// Create connection pool
	var extra_key string
	if !isProduction {
		extra_key = "_TEST"
	}
	dbpool, err := pgxpool.New(
		context.Background(),
		"postgres://"+
			utils.GetEnv("DB_USER", "user")+
			":"+
			utils.GetEnv("DB_PASS", "password")+
			"@"+
			utils.GetEnv("DB_HOST"+extra_key, "localhost")+
			":"+
			utils.GetEnv("DB_PORT"+extra_key, "5432")+
			"/"+
			utils.GetEnv("DB_NAME", "product_api")+
			"?sslmode=disable",
	)
	if err != nil {
		log.Error("Unable to create connection pool", zap.Error(err))
		return
	}

	// Store connection pool in global Db variable
	db.Dbpool = dbpool
	db.IsProduction = isProduction

	// Check if database is reachable
	var greeting string
	greeting, err = db.GetHelloWorld()
	if err != nil {
		log.Error("InitDb failed", err)
		return
	}
	if logInfo {
		log.Info("InitDb succesfull: ", greeting)
	}
	return
}

// CloseDbPool closes the database connection pool
func (db *Database) CloseDbPool() {
	db.Dbpool.Close()
}

// EmptyDatabase truncates all tables in the database
func (db *Database) EmptyDatabase() (err error) {
	if db.IsProduction {
		log.Fatal("Cannot empty production database")
		return
	}
	_, err = db.Dbpool.Exec(context.Background(), "SELECT truncate_tables('user');")
	if err != nil {
		log.Error(err)
	}
	return
}

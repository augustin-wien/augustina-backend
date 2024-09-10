package database

import (
	"augustin/config"
	"augustin/utils"
	"context"

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

// InitDb connects to production database and stores it in the global Db variable
func (db *Database) InitDb() (err error) {
	err = db.initDb(true, true)
	if err != nil {
		return err
	}

	err = initData(db)

	return err
}

// InitEmptyTestDb connects to an empty testing database and store it in the global Db variable
func (db *Database) InitEmptyTestDb() (err error) {
	log.Info("Initializing empty test database")
	err = db.initDb(false, false)
	if err != nil {
		return err
	}
	err = db.EmptyDatabase()
	if err != nil {
		return err
	}

	err = initData(db)

	return
}

// initData initializes the database with default data
func initData(db *Database) (err error) {
	// Initializes DBSettings if not already initialized
	err = db.InitiateDBSettings()
	if err != nil {
		log.Error("Settings creation failed ", zap.Error(err))
	}

	// Get DBSettings
	var dbSettings DBSettings
	dbSettings, err = db.GetDBSettings()
	if err != nil {
		log.Error("Settings retrieval failed ", zap.Error(err))
	}

	if !dbSettings.IsInitialized {

		// Create default settings
		err = db.InitiateSettings()
		if err != nil {
			log.Error("Settings creation failed ", zap.Error(err))
		}
		// Create default accounts
		err = db.InitiateAccounts()
		if err != nil {
			log.Error("Default accounts creation failed ", zap.Error(err))
		}

		// Create default items
		err = db.InitiateItems()
		if err != nil {
			log.Error("Default items creation failed ", zap.Error(err))
		}

		if db.IsProduction {
			err = db.UpdateInitialSettings()
			if err != nil {
				log.Error("Updating initial Settings failed ", zap.Error(err))
			}

			if config.Config.CreateDemoData {
				err = db.CreateDevData()
				if err != nil {
					log.Error("Dev data creation failed ", zap.Error(err))
				}
			}
		}

		// Update DBSettings to initialized
		err = db.UpdateDBSettings(DBSettings{IsInitialized: true})
		if err != nil {
			log.Error("Updating DBSettings failed ", zap.Error(err))
		}
		log.Info("Database successfully initialized")
	} else {
		log.Info("Database already initialized")
	}

	return
}

// initDb initializes the database connection pool and stores it in the global Db variable
func (db *Database) initDb(isProduction bool, logInfo bool) (err error) {

	// Create connection pool
	var extraKey string
	if !isProduction {
		extraKey = "_TEST"
	}
	dbpool, err := pgxpool.New(
		context.Background(),
		"postgres://"+
			utils.GetEnv("DB_USER", "user")+
			":"+
			utils.GetEnv("DB_PASS", "password")+
			"@"+
			utils.GetEnv("DB_HOST"+extraKey, "localhost")+
			":"+
			utils.GetEnv("DB_PORT"+extraKey, "5432")+
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
		log.Info("Connection to database successful: ", greeting)
	}
	return
}

// CloseDbPool closes the database connection pool
func (db *Database) CloseDbPool() {
	db.Dbpool.Close()
}

// EmptyDatabase truncates all tables in the database
func (db *Database) EmptyDatabase() (err error) {
	log.Info("Emptying database executed")
	if db.IsProduction {
		log.Fatal("Cannot empty production database")
		return
	}
	// Show number of accounts existing in database before truncation
	var count int
	err = db.Dbpool.QueryRow(context.Background(), `SELECT COUNT(*) FROM Account;`).Scan(&count)
	if err != nil {
		log.Error("EmptyDatabase: ", err)
		return
	}
	log.Info("Number of accounts before truncation: ", count)


	_, err = db.Dbpool.Exec(context.Background(), `SELECT truncate_tables('user');`)
	log.Info("Database emptied")
	if err != nil {
		log.Error("EmptyDatabase: ", err)
	}

	// Show number of accounts existing in database after truncation
	err = db.Dbpool.QueryRow(context.Background(), `SELECT COUNT(*) FROM Account;`).Scan(&count)
	if err != nil {
		log.Error("EmptyDatabase: ", err)
	}
	log.Info("Number of accounts after truncation: ", count)
	return
}

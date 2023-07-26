package database

import (
	"augustin/utils"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var log = utils.GetLogger()

// Database struct
type Database struct {
	Dbpool *pgxpool.Pool
	IsProduction bool
}

// Db is the global database connection pool that is used by all handlers
var Db Database

// Connect to production database and store it in the global Db variable
func (db *Database) InitDb() (err error) {
	err = db.initDb(true, true)
	return err
}

// Connect to an empty testing database and store it in the global Db variable
func (db *Database) InitEmptyTestDb() (err error) {
	err = db.initDb(false, false)
	if err != nil {
		return err
	}
	err = db.EmptyDatabase()
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
			utils.GetEnv("DB_HOST" + extra_key, "localhost")+
			":"+
			utils.GetEnv("DB_PORT" + extra_key, "5432")+
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
	return
}

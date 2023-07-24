package database

import (
	"augustin/utils"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var log = utils.InitLog()

// Database struct
type Database struct {
	Dbpool *pgxpool.Pool
	IsProduction bool
}

// Db is the global database connection pool that is used by all handlers
var Db Database

// Connect to production database and store it in the global Db variable
func InitDb() (err error) {
	err = initDb(true, true)
	return err
}

// Connect to testing database and store it in the global Db variable
func InitEmptyTestDb() (err error) {
	err = initDb(false, false)
	if err != nil {
		return err
	}
	err = EmptyDatabase()
	return err
}

// initDb initializes the database connection pool and stores it in the global Db variable
func initDb(production bool, info bool) (err error) {

	// Create connection pool
	var extra_key string
	if !production {
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
	Db.Dbpool = dbpool
	Db.IsProduction = production

	// Check if database is reachable
	var greeting string
	greeting, err = Db.GetHelloWorld()
	if err != nil {
		log.Error("InitDb failed", zap.Error(err))
		return
	}
	if info {
		log.Info("InitDb succesfull: ", greeting)
	}
	return
}

// CloseDbPool closes the database connection pool
func (db *Database) CloseDbPool() {
	db.Dbpool.Close()
}

// EmptyDatabase truncates all tables in the database
func EmptyDatabase() (err error) {
	if Db.IsProduction {
		log.Fatal("Cannot empty production database")
		return
	}
	_, err = Db.Dbpool.Exec(context.Background(), "SELECT truncate_tables('user');")
	return
}

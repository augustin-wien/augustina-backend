package database

import (
	"augustin/utils"
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var log = utils.InitLog()

// Database struct
type Database struct {
	// Db, config can be added here
	Dbpool *pgxpool.Pool
	IsProduction bool
}

// Db is the global database connection pool that is used by all handlers
var Db Database

// Connect to production database and store it in the global Db variable
func InitDb() {
	initDb(true)
}

// Connect to testing database and store it in the global Db variable
func InitTestDb() {
	initDb(false)
	err := EmptyDatabase()
	if err != nil {
		log.Error("Unable to empty database: %v\n", zap.Error(err))
		os.Exit(1)
	}
}

// initDb initializes the database connection pool and stores it in the global Db variable
func initDb(production bool) {
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
	log.Info("Actually received", utils.GetEnv("DB_PORT" + extra_key, "5432"))
	if err != nil {
		log.Error("Unable to create connection pool", zap.Error(err))
		os.Exit(1)
	}
	Db = Database{Dbpool: dbpool, IsProduction: production}
	var greeting string
	greeting, err = Db.GetHelloWorld()
	if err != nil {
		log.Errorf("InitDb failed: %v\n", err)
		os.Exit(1)
	}
	log.Info("InitDb succesfull: ", greeting)

}

// CloseDbPool closes the database connection pool
func (db *Database) CloseDbPool() {
	db.Dbpool.Close()
}

// EmptyDatabase truncates all tables in the database
func EmptyDatabase() error {
	if Db.IsProduction {
		log.Fatal("Cannot empty production database")
	}
	_, err := Db.Dbpool.Exec(context.Background(), "SELECT truncate_tables('user');")
	return err

}

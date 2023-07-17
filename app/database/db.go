package database

import (
	"augustin/utils"
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

// Database struct
type Database struct {
	// Db, config can be added here
	Dbpool *pgxpool.Pool
}

// Db is the global database connection pool that is used by all handlers
var Db Database

// InitDb initializes the database connection pool and stores it in the global Db variable
func InitDb() {
	dbpool, err := pgxpool.New(
		context.Background(),
		"postgres://"+
			utils.GetEnv("DB_USER", "user")+
			":"+
			utils.GetEnv("DB_PASS", "password")+
			"@"+
			utils.GetEnv("DB_HOST", "localhost")+
			":"+
			utils.GetEnv("DB_PORT", "5432")+
			"/"+
			utils.GetEnv("DB_NAME", "product_api")+
			"?sslmode=disable",
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	Db = Database{Dbpool: dbpool}

	var greeting string
	greeting, err = Db.GetHelloWorld()
	if err != nil {
		fmt.Fprintf(os.Stderr, "InitDb failed: %v\n", err)
		os.Exit(1)
	}
	log.Infof("InitDb succesfull: %v", greeting)
}

// CloseDbPool closes the database connection pool
func (db *Database) CloseDbPool() {
	db.Dbpool.Close()
}

// TestDbType is a struct that contains a database connection pool and a Database struct
type TestDbType struct {
	Dbpool *pgxpool.Pool
	Db     Database
}

// CreateDbTestInstance creates a new database connection pool for testing purposes and includes some usefull functions
func CreateDbTestInstance() TestDbType {
	dbpool, err := pgxpool.New(
		context.Background(),
		"postgres://"+
			utils.GetEnv("DB_USER", "user")+
			":"+
			utils.GetEnv("DB_PASS", "password")+
			"@"+
			utils.GetEnv("DB_HOST_TEST", "localhost")+
			":"+
			utils.GetEnv("DB_PORT", "5433")+
			"/"+
			utils.GetEnv("DB_NAME", "product_api")+
			"?sslmode=disable",
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	TestDb := TestDbType{
		Dbpool: dbpool,
		Db:     Database{Dbpool: dbpool},
	}
	return TestDb
}

// EmptyDatabase truncates all tables in the database
func (t *TestDbType) EmptyDatabase() error {
	_, err := t.Dbpool.Exec(context.Background(), "truncate table accounts, items, paymenttypes, payments, settings cascade")
	if err != nil {
		log.Fatalf("Truncate failed: %v\n", err)
	}
	return err
}

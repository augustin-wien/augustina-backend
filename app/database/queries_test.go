package database

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	log "github.com/sirupsen/logrus"
)

var (
	host     = "localhost"
	user     = "postgres"
	password = "postgres"
	dbName   = "users_test"
	port     = "5435"
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
)

var resource *dockertest.Resource

var dbpool *pgxpool.Pool

var TestDB Database

func TestMain(m *testing.M) {
	// connect to docker; fail if docker not running
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not connect to docker; is it running? %s", err)
	}
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	log.Println("Starting PostgreSQL Docker container")
	// set up our docker options, specifying the image and so forth
	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14.5",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
			"listen_addresses = '*'",
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: "5435"},
			},
		},
	}

	// get a resource (docker image)
	resource, err = pool.RunWithOptions(&opts, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Printf("Error: could not start resource: %s", err)
		err = pool.Purge(resource)
		if err != nil {
			log.Fatalf("could not purge resource: %s", err)
		}
	}
	log.Println("PostgreSQL Docker container started")

	// start the image and wait until it's ready
	if err := pool.Retry(func() error {
		var err error
		dbpool, err = pgxpool.New(
			context.Background(),
			fmt.Sprintf(dsn, host, port, user, password, dbName),
		)
		if err != nil {
			log.Println("Error:", err)
			return err
		}
		return dbpool.Ping(context.Background())
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to database: %s", err)
	}

	// populate the database with empty tables
	tableSQL, err := os.ReadFile("./testdata/testdata.sql")

	if err != nil {
		log.Errorf("Could not read sql file %v\n", err)
		return
	}

	defer pool.Purge(resource)

	if dbpool == nil {
		log.Fatal("dbpool is nil")
		log.Fatalf("Testdata.sql: %s", string(tableSQL))
		return
	}

	_, err = dbpool.Exec(context.Background(), string(tableSQL))
	if err != nil {
		log.Errorf("Could not populate database with empty table %v\n", err)
		return
	}
	if err != nil {
		log.Fatalf("error creating tables: %s", err)
	}

	// initialize the database
	TestDB = Database{Dbpool: dbpool}

	// run tests
	code := m.Run()

	// clean up
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Exit(code)
}

func Test_pingDB(t *testing.T) {
	err := dbpool.Ping(context.Background())
	if err != nil {
		t.Error("can't ping database")
	}
}

func Test_DatabaseTablesCreated(t *testing.T) {
	var tableCount int
	err := dbpool.QueryRow(context.Background(), "select count(*) from information_schema.tables where table_schema = 'public'").Scan(&tableCount)
	if err != nil {
		t.Error("can't get table count")
	}

	// checks if there are 5 tables in the database, which is the number of tables in testdata.sql
	if tableCount != 5 {
		t.Error("table count not equal to 5")
	}
}

func Test_GetHelloWorld(t *testing.T) {
	var greeting string
	greeting, err := TestDB.GetHelloWorld()
	// err := dbpool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		t.Error("can't get hello world")
	}
	if reflect.TypeOf(greeting).Kind() != reflect.String {
		t.Error("Hello World not of type string")
	}
	if string(greeting) == "'Hello, world!'" {
		t.Errorf("Hello World not equal to 'Hello, world!', instead %s", greeting)
	}
}

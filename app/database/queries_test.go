package database

import (
	"augustin/structs"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestDbType is a struct that contains a database connection pool and a Database struct
var TestDb TestDbType

// TestMain is the main function for the database tests and initializes the database
func TestMain(m *testing.M) {
	log.Info("Starting db tests")
	// initialize database
	TestDb = CreateDbTestInstance()

	var greeting string
	greeting, err := TestDb.Db.GetHelloWorld()
	if err != nil {
		fmt.Fprintf(os.Stderr, "InitDb failed: %v\n", err)
		os.Exit(1)
	}
	log.Infof("InitDb succesfull: %v", greeting)

	// run tests
	code := m.Run()

	// clean up
	TestDb.EmptyDatabase()
	os.Exit(code)
}

// Test_pingDB tests if the database can be pinged
func Test_pingDB(t *testing.T) {
	if TestDb.Dbpool == nil {
		t.Error("dbpool is nil")
	}
	err := TestDb.Dbpool.Ping(context.Background())
	if err != nil {
		t.Error("can't ping database")
	}
}

func Test_DatabaseTablesCreated(t *testing.T) {
	var tableCount int
	err := TestDb.Dbpool.QueryRow(context.Background(), "select count(*) from information_schema.tables where table_schema = 'public'").Scan(&tableCount)
	if err != nil {
		t.Error("can't get table count")
	}

	// checks if there are 5 tables in the database, which is the number of tables in testdata.sql
	if tableCount != 6 {
		t.Error("table count not equal to 5")
	}
}

func Test_GetHelloWorld(t *testing.T) {
	var greeting string
	greeting, err := TestDb.Db.GetHelloWorld()
	// err := dbpool.QueryRow(context.Background(), "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		t.Error("can't get hello world")
	}
	if reflect.TypeOf(greeting).Kind() != reflect.String {
		t.Error("Hello World not of type string")
	}

	require.Equal(t, string(greeting), "Hello, world!")
}

func Test_UpdateSettings(t *testing.T) {
	// Define settings
	err := TestDb.Db.UpdateSettings(structs.Settings{
		Color: "red",
		Logo:  "/img/Augustin-Logo-Rechteck.jpg",
		Items: []structs.Item{
			{
				Name:  "calendar",
				Price: 3.14,
			},
			{
				Name:  "cards",
				Price: 12.12,
			},
		},
	},
	)

	if err != nil {
		t.Errorf("UpdateSettings failed: %v\n", err)
	}
}

func Test_GetVendorSettings(t *testing.T) {
	var vendorsettings string
	vendorsettings, err := TestDb.Db.GetVendorSettings()

	if err != nil {
		t.Error("can't select vendorsettings")
	}
	if reflect.TypeOf(vendorsettings).Kind() != reflect.String {
		t.Error("Vendorsettings not of type string")
	}

	require.Equal(t, string(vendorsettings), `{"credit":1.61,"qrcode":"/img/Augustin-QR-Code.png","idnumber":"123456789"}`)
}

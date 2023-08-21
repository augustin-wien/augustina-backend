package database

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMain is the main function for the database tests and initializes the database
func TestMain(m *testing.M) {
	Db.InitEmptyTestDb()
	os.Exit(m.Run())
}

// Test_pingDB tests if the database can be pinged
func Test_pingDB(t *testing.T) {
	if Db.Dbpool == nil {
		t.Error("dbpool is nil")
	}
	err := Db.Dbpool.Ping(context.Background())
	if err != nil {
		t.Error("can't ping database")
	}
}

func Test_DatabaseTablesCreated(t *testing.T) {
	var tableCount int
	err := Db.Dbpool.QueryRow(context.Background(), "select count(*) from information_schema.tables where table_schema = 'public'").Scan(&tableCount)
	if err != nil {
		t.Error("can't get table count")
	}

	// Check if there is at least one table
	if tableCount < 1 {
		t.Error("No tables exist")
	}
}

func Test_GetHelloWorld(t *testing.T) {
	var greeting string
	greeting, err := Db.GetHelloWorld()
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
	err := Db.UpdateSettings(Settings{
		Color: "red",
		Logo:  "/img/Augustin-Logo-Rechteck.jpg",
	},
	)

	if err != nil {
		t.Errorf("UpdateSettings failed: %v\n", err)
	}
}

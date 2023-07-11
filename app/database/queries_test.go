package database

import (
	"augustin/structs"
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
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
		fmt.Println("test1", dbpool.Ping(context.Background()))
		return dbpool.Ping(context.Background())
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to database: %s", err)
	}

	// populate the database with empty tables
	tableSQL, err := os.ReadFile("./testdata/testdata.sql")

	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
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

	require.Equal(t, string(greeting), "Hello, world!")
}

func Test_UpdateSettings(t *testing.T) {
	// Define settings
	err := TestDB.UpdateSettings(structs.Settings{
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
	vendorsettings, err := TestDB.GetVendorSettings()

	if err != nil {
		t.Error("can't select vendorsettings")
	}
	if reflect.TypeOf(vendorsettings).Kind() != reflect.String {
		t.Error("Vendorsettings not of type string")
	}

	require.Equal(t, string(vendorsettings), `{"credit":1.61,"qrcode":"/img/Augustin-QR-Code.png","idnumber":"123456789"}`)
}

// func Test_Payments(t *testing.T) {
// 	godotenv.Load("../.env")
// 	// Initialize test case
// 	s := handlers.CreateNewServer()
// 	s.MountHandlers()

// 	// Reset tables
// 	_, err := dbpool.Query(context.Background(), "truncate Payments, PaymentTypes, Accounts")
// 	if err != nil {
// 		t.Errorf("Truncate payments failed: %v\n", err)
// 	}

// 	// Set up a payment type
// 	payment_type_id, err := TestDB.CreatePaymentType(
// 		structs.PaymentType{Name: pgtype.Text{String: "Test type", Valid: true}},
// 	)
// 	if err != nil {
// 		t.Errorf("CreatePaymentType failed: %v\n", err)
// 	}

// 	// Set up a payment account
// 	account_id, err := TestDB.CreateAccount(
// 		structs.Account{Name: pgtype.Text{String: "Test account", Valid: true}},
// 	)
// 	if err != nil {
// 		t.Errorf("CreateAccount failed: %v\n", err)
// 	}

// 	// Create payments via API
// 	f := structs.PaymentBatch{
// 		Payments: []structs.Payment{
// 			{
// 				Sender:   account_id,
// 				Receiver: account_id,
// 				Type:     payment_type_id,
// 				Amount:   pgtype.Float4{Float32: 3.14, Valid: true},
// 			},
// 		},
// 	}
// 	var body bytes.Buffer
// 	err = json.NewEncoder(&body).Encode(f)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	req, _ := http.NewRequest("POST", "/api/payments/", &body)
// 	response := executeRequest(req, s)

// 	// Check the response
// 	checkResponseCode(t, http.StatusOK, response.Code)

// 	// Get payments via API
// 	req2, err := http.NewRequest("GET", "/api/payments/", nil)
// 	response2 := executeRequest(req2, s)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Check the response
// 	checkResponseCode(t, http.StatusOK, response2.Code)

// 	// Unmarshal response
// 	var payments []structs.Payment
// 	err = json.Unmarshal(response2.Body.Bytes(), &payments)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Test payments response
// 	require.Equal(t, payments[0].Amount.Float32, float32(3.14))
// 	require.Equal(t, payments[0].Sender, account_id)
// 	require.Equal(t, payments[0].Receiver, account_id)
// 	require.Equal(t, payments[0].Type, payment_type_id)
// 	require.Equal(t, payments[0].Timestamp.Time.Day(), time.Now().Day())
// 	require.Equal(t, payments[0].Timestamp.Time.Hour(), time.Now().Hour())
// 	require.Equal(t, len(payments), 1)
// }

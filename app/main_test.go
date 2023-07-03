// Documentation see here: https://go-chi.io/#/pages/testing
package main

import (
	"augustin/database"
	"augustin/structs"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// WARNING: The tests use the main database, do not run tests in production

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)

	return rr
}

// checkResponseCode is a simple utility to check the response code
// of the response
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}


func TestHelloWorld(t *testing.T) {
	// Initialize database
	database.InitDb()

	// Create a New Server Struct
	s := CreateNewServer()

	// Mount Handlers
	s.MountHandlers()

	// Create a New Request
	req, _ := http.NewRequest("GET", "/api/hello/", nil)

	// Execute Request
	response := executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, http.StatusOK, response.Code)

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, "Hello, world!", response.Body.String())
}

func TestHelloWorldAuth(t *testing.T) {
	// Initialize database
	// TODO: This is not a test database, but the real one!
	err := godotenv.Load("../.env")
	if err != nil {
		log.Error("Error loading .env file")
	}
	database.InitDb()

	// Create a New Server Struct
	s := CreateNewServer()

	// Mount Handlers
	s.MountHandlers()

	// Create a New Request
	req, _ := http.NewRequest("GET", "/api/auth/hello/", nil)

	// Execute Request
	response := executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, 401, response.Code)

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, "Unauthorized\n", response.Body.String())
}
func TestPayments(t *testing.T) {
	// Initialize test case
	database.InitDb()
	s := CreateNewServer()
	s.MountHandlers()

	// Reset tables
	_, err := database.Db.Dbpool.Query(context.Background(), "truncate Payments, PaymentTypes, Accounts")
	if err != nil {
		log.Errorf("Truncate payments failed: %v\n", err)
	}

	// Set up a payment type
	payment_type_id, err := database.Db.CreatePaymentType(
		structs.PaymentType{Name: pgtype.Text{String: "Test type", Valid: true}},
	)
	if err != nil {
		log.Errorf("CreatePaymentType failed: %v\n", err)
	}

	// Set up a payment account
	account_id, err := database.Db.CreateAccount(
		structs.Account{Name: pgtype.Text{String: "Test account", Valid: true}},
	)
	if err != nil {
		log.Errorf("CreateAccount failed: %v\n", err)
	}

	// Create payments via API
	f := structs.PaymentBatch{
		Payments: []structs.Payment{
			{
				Sender:   account_id,
				Receiver: account_id,
				Type:     payment_type_id,
				Amount:   pgtype.Float4{Float32: 3.14, Valid: true},
			},
		},
	}
	var body bytes.Buffer
	err = json.NewEncoder(&body).Encode(f)
	if err != nil {
		log.Fatal(err)
	}
	req, _ := http.NewRequest("POST", "/api/payments/", &body)
	response := executeRequest(req, s)

	// Check the response
	checkResponseCode(t, http.StatusOK, response.Code)

	// Get payments via API
	req2, err := http.NewRequest("GET", "/api/payments/", nil)
	response2 := executeRequest(req2, s)
	if err != nil {
		log.Fatal(err)
	}

	// Check the response
	checkResponseCode(t, http.StatusOK, response2.Code)

	// Unmarshal response
	var payments []structs.Payment
	err = json.Unmarshal(response2.Body.Bytes(), &payments)
	if err != nil {
		panic(err)
	}

	// Test payments response
	require.Equal(t, payments[0].Amount.Float32, float32(3.14))
	require.Equal(t, payments[0].Sender, account_id)
	require.Equal(t, payments[0].Receiver, account_id)
	require.Equal(t, payments[0].Type, payment_type_id)
	require.Equal(t, payments[0].Timestamp.Time.Day(), time.Now().Day())
	require.Equal(t, payments[0].Timestamp.Time.Hour(), time.Now().Hour())
	require.Equal(t, len(payments), 1)

}

func TestSettings(t *testing.T) {
	database.InitDb()
	s := CreateNewServer()
	s.MountHandlers()

	// Define settings
	database.Db.UpdateSettings(structs.Settings{
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

	// Create a New Request
	req, _ := http.NewRequest("GET", "/api/settings/", nil)

	// Execute Request
	response := executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, http.StatusOK, response.Code)

	// Unmarshal response
	var settings structs.Settings
	err := json.Unmarshal(response.Body.Bytes(), &settings)
	if err != nil {
		panic(err)
	}

	// Test response
	require.Equal(t, settings.Color, "red")
	require.Equal(t, settings.Logo, "/img/Augustin-Logo-Rechteck.jpg")
	require.Equal(t, settings.Items[0].Name, "calendar")
	require.Equal(t, settings.Items[0].Price, float32(3.14))
	require.Equal(t, settings.Items[1].Name, "cards")
	require.Equal(t, settings.Items[1].Price, float32(12.12))
}

func TestVendor(t *testing.T) {
	// Create a New Server Struct
	s := CreateNewServer()
	// Mount Handlers
	s.MountHandlers()

	// Create a New Request
	req, _ := http.NewRequest("GET", "/api/vendor/", nil)

	// Execute Request
	response := executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, http.StatusOK, response.Code)
	marshal_struct, _ := json.Marshal(&structs.Vendor{Credit: 1.61, QRcode: "/img/Augustin-QR-Code.png", IDnumber: "123456789"})

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, string(marshal_struct), response.Body.String())
}

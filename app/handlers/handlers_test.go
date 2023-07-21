package handlers

import (
	"augustin/database"
	"augustin/structs"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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

func TestMain(m *testing.M) {
	// Connect to testing database
	database.InitTestDb()

	// Run tests
	code := m.Run()

	// Connect back to production database
	database.InitDb()

	// Exit with the code from the tests
	os.Exit(code)
}

func TestHelloWorld(t *testing.T) {
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

func Test_Payments(t *testing.T) {
	// Initialize test case
	s := CreateNewServer()
	s.MountHandlers()

	// Set up a payment type
	payment_type_id, err := database.Db.CreatePaymentType(
		structs.PaymentType{
			Name: "Test type",
		},
	)
	if err != nil {
		t.Errorf("CreatePaymentType failed: %v\n", err)
	}

	// Set up a payment account
	account_id, err := database.Db.CreateAccount(
		structs.Account{Name: "Test account"},
	)
	if err != nil {
		t.Errorf("CreateAccount failed: %v\n", err)
	}

	// Create payments via API
	f := structs.PaymentBatch{
		Payments: []structs.Payment{
			{
				Sender:   account_id,
				Receiver: account_id,
				Type:     payment_type_id,
				Amount:   float32(3.14),
			},
		},
	}
	var body bytes.Buffer
	err = json.NewEncoder(&body).Encode(f)
	if err != nil {
		log.Fatal("smth", zap.Error(err))
	}
	req, _ := http.NewRequest("POST", "/api/payments/", &body)
	response := executeRequest(req, s)

	// Check the response
	checkResponseCode(t, http.StatusOK, response.Code)

	// Get payments via API
	req2, err := http.NewRequest("GET", "/api/payments/", nil)
	response2 := executeRequest(req2, s)
	if err != nil {
		log.Fatal("smth", zap.Error(err))
	}

	// Check the response
	checkResponseCode(t, http.StatusOK, response2.Code)

	// Unmarshal response
	var payments []structs.Payment
	err = json.Unmarshal(response2.Body.Bytes(), &payments)
	if err != nil {
		panic(err)
	}
	require.Equal(t, 1, len(payments))
	if t.Failed() {
		return
	}

	// Test payments response
	require.Equal(t, payments[0].Amount, float32(3.14))
	require.Equal(t, payments[0].Sender, account_id)
	require.Equal(t, payments[0].Receiver, account_id)
	require.Equal(t, payments[0].Type, payment_type_id)
	require.Equal(t, payments[0].Timestamp.Time.Day(), time.Now().Day())
	require.Equal(t, payments[0].Timestamp.Time.Hour(), time.Now().Hour())

}

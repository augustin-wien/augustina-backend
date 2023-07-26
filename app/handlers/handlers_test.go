package handlers

import (
	"augustin/database"
	"augustin/structs"
	"augustin/utils"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var router *chi.Mux

// TestMain is executed before all tests and initializes an empty database
func TestMain(m *testing.M) {
	database.Db.InitEmptyTestDb()
	router = GetRouter()
	os.Exit(m.Run())
}


func TestHelloWorld(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/hello/", nil)
	response := utils.SubmitRequest(req, router)

	// Check the response code
	utils.CheckResponse(t, http.StatusOK, response.Code)

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, "\"Hello, world!\"", response.Body.String())
}

func TestPayments(t *testing.T) {

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
	response := executeRequest(req, router)

	// Check the response
	checkResponseCode(t, http.StatusOK, response.Code)

	// Get payments via API
	req2, err := http.NewRequest("GET", "/api/payments/", nil)
	response2 := executeRequest(req2, router)
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

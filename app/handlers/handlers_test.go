package handlers

import (
	"augustin/database"
	"augustin/utils"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

var r *chi.Mux



// TestMain is executed before all tests and initializes an empty database
func TestMain(m *testing.M) {
	database.Db.InitEmptyTestDb()
	r = GetRouter()
	os.Exit(m.Run())
}


func TestHelloWorld(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/hello/", nil)
	response := utils.SubmitRequest(req, r)

	// Check the response code
	utils.CheckResponse(t, http.StatusOK, response.Code)

	// We can use testify/require to assert values, as it is more convenient
	require.Equal(t, "\"Hello, world!\"", response.Body.String())
}

// TestItems tests CRUD operations on items (including images)
func TestItems(t *testing.T) {
	f := `{
		"Name": "Test item",
		"Price": 3.14,
		"IsEditable": true
	}`
	utils.TestRequest(t, r, "POST", "/api/item/", f, 200)
}

func TestPayments(t *testing.T) {

	// Set up a payment type
	payment_type_id, err := database.Db.CreatePaymentType(
		database.PaymentType{
			Name: "Test type",
		},
	)
	utils.CheckError(t, err)

	// Set up a payment account
	account_id, err := database.Db.CreateAccount(
		database.Account{Name: "Test account"},
	)
	utils.CheckError(t, err)

	// Create payments via API
	f := database.PaymentBatch{
		Payments: []database.Payment{
			{
				Sender:   account_id,
				Receiver: account_id,
				Type:     payment_type_id,
				Amount:   float32(3.14),
			},
		},
	}
	utils.TestRequest(t, r, "POST", "/api/payments/", f, 200)
	response2 := utils.TestRequest(t, r, "GET", "/api/payments/", nil, 200)

	// Unmarshal response
	var payments []database.Payment
	err = json.Unmarshal(response2.Body.Bytes(), &payments)
	utils.CheckError(t, err)
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

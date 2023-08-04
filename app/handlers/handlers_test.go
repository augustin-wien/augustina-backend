package handlers

import (
	"augustin/database"
	"augustin/utils"
	"encoding/json"
	"os"
	"testing"

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
	res := utils.TestRequest(t, r, "GET", "/api/hello/", nil, 200)
	require.Equal(t, "\"Hello, world!\"", res.Body.String())
}

// TestUsers tests CRUD operations on users
func TestUsers(t *testing.T) {
	// Create
	json_vendor := `{
		"keycloakID": "test",
		"urlID": "test",
		"licenseID": "test",
		"firstName": "test1234",
		"lastName": "test"
	}`
	res := utils.TestRequestStr(t, r, "POST", "/api/vendors/", json_vendor, 200)
	vendorID := res.Body.String()
	res = utils.TestRequest(t, r, "GET", "/api/vendors/", nil, 200)
	var vendors []database.Vendor
	err := json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))
	require.Equal(t, "test1234", vendors[0].FirstName)

	// Update
	json_vendor = `{"firstName": "nameAfterUpdate"}`
	utils.TestRequestStr(t, r, "PUT", "/api/vendors/"+vendorID+"/", json_vendor, 200)
	res = utils.TestRequest(t, r, "GET", "/api/vendors/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 1, len(vendors))
	require.Equal(t, "nameAfterUpdate", vendors[0].FirstName)

	// Delete
	utils.TestRequest(t, r, "DELETE", "/api/vendors/"+vendorID+"/", nil, 204)
	res = utils.TestRequest(t, r, "GET", "/api/vendors/", nil, 200)
	err = json.Unmarshal(res.Body.Bytes(), &vendors)
	utils.CheckError(t, err)
	require.Equal(t, 0, len(vendors))
}


// TestItems tests CRUD operations on items (including images)
// func TestItems(t *testing.T) {
// 	f := `{
// 		"Name": "Test item",
// 		"Price": 3.14,
// 		"IsEditable": true
// 	}`
// 	utils.TestRequest(t, r, "POST", "/api/item/", f, 200)
// }

// func TestPayments(t *testing.T) {

// 	// Set up a payment type
// 	payment_type_id, err := database.Db.CreatePaymentType(
// 		database.PaymentType{
// 			Name: "Test type",
// 		},
// 	)
// 	utils.CheckError(t, err)

// 	// Set up a payment account
// 	account_id, err := database.Db.CreateAccount(
// 		database.Account{Name: "Test account"},
// 	)
// 	utils.CheckError(t, err)

// 	// Create payments via API
// 	f := database.PaymentBatch{
// 		Payments: []database.Payment{
// 			{
// 				Sender:   account_id,
// 				Receiver: account_id,
// 				Type:     payment_type_id,
// 				Amount:   float32(3.14),
// 			},
// 		},
// 	}
// 	utils.TestRequest(t, r, "POST", "/api/payments/", f, 200)
// 	response2 := utils.TestRequest(t, r, "GET", "/api/payments/", nil, 200)

// 	// Unmarshal response
// 	var payments []database.Payment
// 	err = json.Unmarshal(response2.Body.Bytes(), &payments)
// 	utils.CheckError(t, err)
// 	require.Equal(t, 1, len(payments))
// 	if t.Failed() {
// 		return
// 	}

// 	// Test payments response
// 	require.Equal(t, payments[0].Amount, float32(3.14))
// 	require.Equal(t, payments[0].Sender, account_id)
// 	require.Equal(t, payments[0].Receiver, account_id)
// 	require.Equal(t, payments[0].Type, payment_type_id)
// 	require.Equal(t, payments[0].Timestamp.Time.Day(), time.Now().Day())
// 	require.Equal(t, payments[0].Timestamp.Time.Hour(), time.Now().Hour())

// }

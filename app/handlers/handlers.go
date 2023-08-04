//	@title			Augustin Swagger
//	@version		0.0.1
//	@description	This swagger describes every endpoint of this project.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	GNU Affero General Public License
//	@license.url	https://www.gnu.org/licenses/agpl-3.0.txt

//	@host		localhost:3000
//	@BasePath	/api
// @accept json

//	@securityDefinitions.basic	BasicAuth

//	@externalDocs.description	OpenAPI
//	@externalDocs.url			https://swagger.io/resources/open-api/

package handlers

import (
	"augustin/utils"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"

	_ "github.com/swaggo/files"        // swagger embed files
	_ "github.com/swaggo/http-swagger" // http-swagger middleware

	"augustin/database"
	"augustin/paymentprovider"
)

type TransactionOrder struct {
	Amount int
}

type TransactionOrderResponse struct {
	SmartCheckoutURL string
}

type TransactionVerification struct {
	TransactionID string
}

type TransactionVerificationResponse struct {
	Verification bool
}

var log = utils.GetLogger()

// ReturnHelloWorld godoc
//
//	@Summary		Return HelloWorld
//	@Description	Return HelloWorld as sample API call
//	@Tags			core
//	@Accept			json
//	@Produce		json
//	@Router			/hello/ [get]
//
// HelloWorld API Handler fetching data from database
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, greeting)
}

// UpdateItem requires a multipart form
// https://www.sobyte.net/post/2022-03/go-multipart-form-data/
func UpdateItem(w http.ResponseWriter, r *http.Request) (err error) {

	// Read multipart form
	r.ParseMultipartForm(32 << 20)
	mForm := r.MultipartForm

	// Handle normal fields
	var item database.Item
	fields := mForm.Value
	err = mapstructure.Decode(fields, &item)
	if err != nil {
		panic(err)
	}

	// Get file from image field
	file, header, err := r.FormFile("Image")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Debugging
	name := strings.Split(header.Filename, ".")
	log.Infof("Uploading %s\n", name[0])

	// Save file
	path := "/img/" + header.Filename
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	io.Copy(f, file)
	item.Image = path

	// Save item to database
	err = database.Db.UpdateItem(item)
	if err != nil {
		panic(err)
	}

	return err
}

// GetPayments godoc
//
//	@Summary	Get all payments
//	@Tags		core
//	@Accept		json
//	@Produce	json
//	@Success	200	{array}	database.Payment
//	@Router		/payments/ [get]
func GetPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := database.Db.GetPayments()
	if err != nil {
		log.Errorf("GetPayments DB Error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	marshal_struct, err := json.Marshal(payments)
	if err != nil {
		log.Errorf("JSON conversion failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(marshal_struct))
}

// CreatePayments godoc
//
//	@Summary		Create a set of payments
//	@Description	{"Payments":[{"Sender": 1, "Receiver":1, "Type":1,"Amount":1.00}]}
//	@Tags			core
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}	database.PaymentType
//	@Router			/payments/ [post]
func CreatePayments(w http.ResponseWriter, r *http.Request) {
	var paymentBatch database.PaymentBatch
	err := utils.ReadJSON(w, r, &paymentBatch)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.CreatePayments(paymentBatch.Payments)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
}

// CreateTransactionOrder godoc
//
//	@Summary		Create a transaction order
//	@Description	Post your amount like {"Amount":100}, which equals 100 cents
//	@Tags			core
//	@accept			json
//	@Produce		json
//	@Param			amount body TransactionOrder true "Amount in cents"
//	@Success		200	{array}	TransactionOrderResponse
//	@Router			/vivawallet/transaction_order/ [post]
func CreateTransactionOrder(w http.ResponseWriter, r *http.Request) {
	var transactionOrder TransactionOrder
	err := utils.ReadJSON(w, r, &transactionOrder)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create a new payment order
	accessToken, err := paymentprovider.AuthenticateToVivaWallet()
	if err != nil {
		log.Error("Authentication failed: ", err)
	}
	log.Info("Access token: ", accessToken)
	orderCode, err := paymentprovider.CreatePaymentOrder(accessToken, transactionOrder.Amount)
	if err != nil {
		log.Error("Creating payment order failed: ", err)
	}
	log.Info("Order code: ", orderCode)

	// Create response
	url := "https://demo.vivapayments.com/web/checkout?ref=" + strconv.Itoa(orderCode)
	response := TransactionOrderResponse{
		SmartCheckoutURL: url,
	}
	utils.WriteJSON(w, http.StatusOK, response)

}

// VerifyTransaction godoc
//
//	@Summary		Verify a transaction
//	@Description	Accepts {"TransactionID":"1234567890"} and returns {"Verification":true}, if successful
//	@Tags			core
//	@accept			json
//	@Produce		json
//	@Param			transactionID body TransactionVerification true "Transaction ID"
//	@Success		200	{array}	TransactionVerificationResponse
//	@Router			/vivawallet/transaction_verification/ [post]
func VerifyTransaction(w http.ResponseWriter, r *http.Request) {
	var transactionVerification TransactionVerification
	err := utils.ReadJSON(w, r, &transactionVerification)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get access token
	accessToken, err := paymentprovider.AuthenticateToVivaWallet()
	if err != nil {
		log.Error("Authentication failed: ", err)
	}

	// Verify transaction
	verification, err := paymentprovider.VerifyTransactionID(accessToken, transactionVerification.TransactionID)
	if err != nil {
		log.Info("Verifying transaction failed: ", err)
		return
	}

	// Create response
	response := TransactionVerificationResponse{
		Verification: verification,
	}
	utils.WriteJSON(w, http.StatusOK, response)

}

// getSettings godoc
//
//	@Summary		Return settings
//	@Description	Return settings about the web-shop
//	@Tags			core
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}	database.Settings
//	@Router			/settings/ [get]
func getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, settings)
}

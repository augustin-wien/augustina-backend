package handlers

import (
	"augustin/utils"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mitchellh/mapstructure"

	"augustin/database"

	_ "github.com/swaggo/files"        // swagger embed files
	_ "github.com/swaggo/http-swagger" // http-swagger middleware

	"augustin/paymentprovider"
)

var log = utils.GetLogger()

// respond takes care of writing the response to the client
func respond(w http.ResponseWriter, err error, payload interface{}) {
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, payload)
}

type TransactionOrder struct {
	Amount int
}

type TransactionOrderResponse struct {
	SmartCheckoutURL string
}

type TransactionVerification struct {
	OrderCode int
}

type TransactionVerificationResponse struct {
	Verification bool
}

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

// Users ----------------------------------------------------------------------

// ListVendors godoc
//
//	 	@Summary 		List Vendors
//		@Tags			vendors
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Vendor
//		@Router			/vendors/ [get]
func ListVendors(w http.ResponseWriter, r *http.Request) {
	users, err := database.Db.ListVendors()
	respond(w, err, users)
}

// CreateVendor godoc
//
//	 	@Summary 		Create Vendor
//		@Tags			vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//	    @Param		    data body database.Vendor true "Vendor Representation"
//		@Router			/vendors/ [post]
func CreateVendor(w http.ResponseWriter, r *http.Request) {
	var vendor database.Vendor
	err := utils.ReadJSON(w, r, &vendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Info("id ", vendor.ID, "fn ", vendor.FirstName, "bl ", vendor.Balance)

	id, err := database.Db.CreateVendor(vendor)
	respond(w, err, id)
}

// UpdateVendor godoc
//
//		 	@Summary 		Update Vendor
//			@Description	Warning: Unfilled fields will be set to default values
//			@Tags			vendors
//			@Accept			json
//			@Produce		json
//			@Success		200
//	        @Param          id   path int  true  "Vendor ID"
//		    @Param		    data body database.Vendor true "Vendor Representation"
//			@Router			/vendors/{id}/ [put]
func UpdateVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	var vendor database.Vendor
	err = utils.ReadJSON(w, r, &vendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.UpdateVendor(vendorID, vendor)
	respond(w, err, vendor)
}

// DeleteVendor godoc
//
//		 	@Summary 		Delete Vendor
//			@Tags			vendors
//			@Accept			json
//			@Produce		json
//			@Success		200
//	        @Param          id   path int  true  "Vendor ID"
//			@Router			/vendors/{id}/ [delete]
func DeleteVendor(w http.ResponseWriter, r *http.Request) {
	vendorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.DeleteVendor(vendorID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Items (that can be sold) ---------------------------------------------------

// ListItems godoc
//
//	 	@Summary 		List Items
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Item
//		@Router			/items/ [get]
func ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := database.Db.ListItems()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, items)
}

// CreateItem godoc
//
//	 	@Summary 		Create Item
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Item true "Item Representation"
//		@Success		200	 {int}	id
//		@Router			/items/ [post]
func CreateItem(w http.ResponseWriter, r *http.Request) {
	var item database.Item
	err := utils.ReadJSON(w, r, &item)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	id, err := database.Db.CreateItem(item)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, id)
}

func UpdateItemImage(w http.ResponseWriter, r *http.Request) (path string, err error) {
	// Get file from image field
	file, header, err := r.FormFile("Image")
	if err != nil {
		return // No file passed, which is ok
	}
	defer file.Close()

	// Debugging
	name := strings.Split(header.Filename, ".")
	if len(name) != 2 {
		log.Error(err)
		utils.ErrorJSON(w, errors.New("invalid filename"), http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Generate unique filename
	i := 0
	for {
		path = "/img/" + name[0] + "_" + strconv.Itoa(i) + "." + name[1]
		_, err = os.Stat(".." + path)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		i += 1
		if i > 1000 {
			log.Error(err)
			utils.ErrorJSON(w, errors.New("too many files with same name"), http.StatusBadRequest)
			return
		}
	}

	// Save file with unique name
	err = os.WriteFile(".."+path, buf.Bytes(), 0666)
	if err != nil {
		log.Error(err)
	}
	return
}

// UpdateItem godoc
//
//	 	@Summary 		Update Item
//		@Description	Requires multipart form (for image)
//		@Tags			Items
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Item true "Item Representation"
//		@Success		200
//		@Router			/items/{id}/ [put]
//
// UpdateItem requires a multipart form
// https://www.sobyte.net/post/2022-03/go-multipart-form-data/
func UpdateItem(w http.ResponseWriter, r *http.Request) {
	ItemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Read multipart form
	r.ParseMultipartForm(32 << 20)
	mForm := r.MultipartForm
	if mForm == nil {
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	// Handle normal fields
	var item database.Item
	fields := mForm.Value                   // Values are stored in []string
	fields_clean := make(map[string]string) // Values are stored in string
	for key, value := range fields {
		fields_clean[key] = value[0]
	}
	err = mapstructure.Decode(fields_clean, &item)
	if err != nil {
		log.Error(err)
	}

	path, _ := UpdateItemImage(w, r)
	if path != "" {
		item.Image = path
	}

	// Save item to database
	err = database.Db.UpdateItem(ItemID, item)
	if err != nil {
		log.Error(err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
	}

}

// DeleteItem godoc
//
//		 	@Summary 		Delete Item
//			@Tags			Items
//			@Accept			json
//			@Produce		json
//			@Success		200
//	        @Param          id   path int  true  "Item ID"
//			@Router			/items/{id} [delete]
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	ItemID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.DeleteItem(ItemID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Orders ---------------------------------------------------------------------

type CreateOrderRequest struct {
	Entries []database.OrderEntry
	User    string
	Vendor  int32
}

type CreateOrderResponse struct {
	SmartCheckoutURL string
}

// CreatePaymentOrder godoc
//
//	 	@Summary 		Create Payment Order
//		@Description	Submits payment order & saves it to database
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//	    @Param		    data body CreateOrderRequest true "Payment Order"
//		@Success		200 {object} CreateOrderResponse
//		@Router			/orders/ [post]
func CreatePaymentOrder(w http.ResponseWriter, r *http.Request) {

	// Read payment order from request
	var order database.Order
	err := utils.ReadJSON(w, r, &order)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Identify accounts
	var buyerAccount database.Account
	if order.User.Valid {
		buyerAccount, err = database.Db.GetAccountByUser(order.User.String)
	} else {
		buyerAccount, err = database.Db.GetAccountByType("UserAnon")
	}
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	vendorAccount, err := database.Db.GetAccountByVendor(order.Vendor)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	orgaAccount, err := database.Db.GetAccountByType("Orga")
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Extend order entries
	for _, entry := range order.Entries {
		// Get item from database
		item, err := database.Db.GetItem(entry.Item)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Define flow of money from buyer to vendor
		entry.Sender = buyerAccount.ID
		log.Info("Sender ", entry.Sender)
		entry.Receiver = vendorAccount.ID

		// If there is a license item, prepend it before the actual item
		if item.LicenseItem.Valid {
			licenseItem, err := database.Db.GetItem(int(item.LicenseItem.Int64))
			if err != nil {
				utils.ErrorJSON(w, err, http.StatusBadRequest)
				return
			}
			// Define flow of money from vendor to orga
			licenseItemEntry := database.OrderEntry{
				Item:     int(item.LicenseItem.Int64),
				Quantity: entry.Quantity,
				Price:    licenseItem.Price,
				Sender:   vendorAccount.ID,
				Receiver: orgaAccount.ID,
			}
			order.Entries = append([]database.OrderEntry{licenseItemEntry}, order.Entries...)
		}

	}

	// Submit order to vivawallet
	accessToken, err := paymentprovider.AuthenticateToVivaWallet()
	if err != nil {
		log.Error("Authentication failed: ", err)
	}
	OrderCode, err := paymentprovider.CreatePaymentOrder(accessToken, order.GetTotal())
	if err != nil {
		log.Error("Creating payment order failed: ", err)
	}

	// Save order to database
	order.OrderCode = strconv.Itoa(OrderCode)
	log.Info("Order vendor: ", order.Vendor)
	_, err = database.Db.CreateOrder(order)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create response
	url := "https://demo.vivapayments.com/web/checkout?ref=" + strconv.Itoa(OrderCode)
	response := CreateOrderResponse{
		SmartCheckoutURL: url,
	}
	utils.WriteJSON(w, http.StatusOK, response)
}

// VerifyPaymentOrder godoc
//
//	 	@Summary 		Verify Payment Order
//		@Description	Verifies order and creates payments
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//		@Success		200 {object} database.Order
//		@Param			s query string true "Order Code" Format(3043685539722561)
//		@Param			t query string true "Transaction ID" Format(882d641c-01cc-442f-b894-2b51250340b5)
//		@Router			/orders/verify/ [post]
func VerifyPaymentOrder(w http.ResponseWriter, r *http.Request) {

	// Get transaction ID from URL parameter
	OrderCode := r.URL.Query().Get("s")
	TransactionID := r.URL.Query().Get("t")

	// Get payment order from database
	order, err := database.Db.GetOrderByOrderCode(OrderCode)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Return success message if already verified
	if order.Verified {
		utils.WriteJSON(w, http.StatusOK, order)
		return
	}

	// Get access token
	accessToken, err := paymentprovider.AuthenticateToVivaWallet()
	if err != nil {
		log.Error("Authentication failed: ", err)
	}

	// Verify transaction
	isVerified, err := paymentprovider.VerifyTransactionID(accessToken, TransactionID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if !isVerified {
		utils.ErrorJSON(w, errors.New("transaction not verified"), http.StatusBadRequest)
		return
	}

	// Add additional entries in order (e.g. transaction fees)
	// TODO: order.Entries = append(order.Entries, MyEntry)

	// Store verification in db
	order.TransactionID = TransactionID
	order.Verified = isVerified
	err = database.Db.UpdateOrderAndCreatePayments(order.ID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create response
	utils.WriteJSON(w, http.StatusOK, order)
}

// Payments (from one account to another account) -----------------------------

// ListPayments godoc
//
//	 	@Summary 		Get list of all payments
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Payment
//		@Router			/payments [get]
func ListPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := database.Db.ListPayments()
	respond(w, err, payments)
}

type CreatePaymentsRequest struct {
	Payments []database.Payment
}

// CreatePayments godoc
//
//	 	@Summary 		Create a set of payments
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Param			amount body CreatePaymentsRequest true " Create Payment"
//		@Success		200
//		@Router			/payments [post]
func CreatePayments(w http.ResponseWriter, r *http.Request) {
	var paymentBatch CreatePaymentsRequest
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

// VivaWallet MVP (to be replaced by PaymentOrder API) ------------------------

// VivaWalletCreateTransactionOrder godoc
//
//	@Summary		Create a transaction order
//	@Description	Post your amount like {"Amount":100}, which equals 100 cents
//	@Tags			core
//	@accept			json
//	@Produce		json
//	@Param			amount body TransactionOrder true "Amount in cents"
//	@Success		200	{array}	TransactionOrderResponse
//	@Router			/vivawallet/transaction_order/ [post]
func VivaWalletCreateTransactionOrder(w http.ResponseWriter, r *http.Request) {
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
	orderCode, err := paymentprovider.CreatePaymentOrder(accessToken, transactionOrder.Amount)
	if err != nil {
		log.Error("Creating payment order failed: ", err)
	}

	// Create response
	url := "https://demo.vivapayments.com/web/checkout?ref=" + strconv.Itoa(orderCode)
	response := TransactionOrderResponse{
		SmartCheckoutURL: url,
	}
	utils.WriteJSON(w, http.StatusOK, response)

}

// VivaWalletVerifyTransaction godoc
//
//	@Summary		Verify a transaction
//	@Description	Accepts {"OrderCode":"1234567890"} and returns {"Verification":true}, if successful
//	@Tags			core
//	@accept			json
//	@Produce		json
//	@Param			OrderCode body TransactionVerification true "Transaction ID"
//	@Success		200	{array}	TransactionVerificationResponse
//	@Router			/vivawallet/transaction_verification/ [post]
func VivaWalletVerifyTransaction(w http.ResponseWriter, r *http.Request) {
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
	verification, err := paymentprovider.VerifyTransactionID(accessToken, strconv.Itoa(transactionVerification.OrderCode))
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

// Settings -------------------------------------------------------------------

// getSettings godoc
//
//	 	@Summary 		Return settings
//		@Description	Return configuration data of the system
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Settings
//		@Router			/settings/ [get]
func getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, settings)
}

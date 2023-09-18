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
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/guregu/null.v4"

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

type transactionOrder struct {
	Amount int
}

type transactionOrderResponse struct {
	SmartCheckoutURL string
}

type transactionVerification struct {
	OrderCode int
}

type transactionVerificationResponse struct {
	Verification bool
}

// HelloWorld godoc
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

func updateItemImage(w http.ResponseWriter, r *http.Request) (path string, err error) {
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
		i++
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
	fields := mForm.Value                  // Values are stored in []string
	fieldsClean := make(map[string]string) // Values are stored in string
	for key, value := range fields {
		fieldsClean[key] = value[0]
	}
	err = mapstructure.Decode(fieldsClean, &item)
	if err != nil {
		log.Error(err)
	}

	path, _ := updateItemImage(w, r)
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

type createOrderRequestEntry struct {
	Item     int
	Quantity int
}

type createOrderRequest struct {
	Entries []createOrderRequestEntry
	User    string
	Vendor  int32
}

type createOrderResponse struct {
	SmartCheckoutURL string
}

// CreatePaymentOrder godoc
//
//	 	@Summary 		Create Payment Order
//		@Description	Submits payment order to provider & saves it to database. Entries need to have an item id and a quantity (for entries without a price like tips, the quantity is the amount of cents). If no user is given, the order is anonymous.
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//	    @Param		    data body createOrderRequest true "Payment Order"
//		@Success		200 {object} createOrderResponse
//		@Router			/orders/ [post]
func CreatePaymentOrder(w http.ResponseWriter, r *http.Request) {

	// Read payment order from request
	var order database.Order
	err := utils.ReadJSON(w, r, &order)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get accounts
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
	vendorAccount, err := database.Db.GetAccountByVendorID(order.Vendor)
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
	for idx, entry := range order.Entries {
		// Get item from database
		item, err := database.Db.GetItem(entry.Item)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Define flow of money from buyer to vendor
		order.Entries[idx].Sender = buyerAccount.ID
		order.Entries[idx].Receiver = vendorAccount.ID
		order.Entries[idx].Price = item.Price // Take current item price

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

	// Submit order to vivawallet (disabled in tests)
	var OrderCode int
	if database.Db.IsProduction {
		accessToken, err := paymentprovider.AuthenticateToVivaWallet()
		if err != nil {
			log.Error("Authentication failed: ", err)
		}
		OrderCode, err = paymentprovider.CreatePaymentOrder(accessToken, order.GetTotal())
		if err != nil {
			log.Error("Creating payment order failed: ", err)
		}
	}

	// Save order to database
	order.OrderCode.String = strconv.Itoa(OrderCode)
	order.OrderCode.Valid = true // This means that it is not null
	_, err = database.Db.CreateOrder(order)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create response
	url := "https://demo.vivapayments.com/web/checkout?ref=" + strconv.Itoa(OrderCode)
	response := createOrderResponse{
		SmartCheckoutURL: url,
	}
	utils.WriteJSON(w, http.StatusOK, response)
}

type verifyOrderResponse struct {
	ID            int
	OrderCode     string
	TransactionID string
	Verified      bool
	Timestamp     string
	User          string
	Vendor        int
	Entries       []database.OrderEntry
}

// VerifyPaymentOrder godoc
//
//	 	@Summary 		Verify Payment Order
//		@Description	Verifies order and creates payments
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//		@Success		200 {object} verifyOrderResponse
//		@Param			s query string true "Order Code" Format(3043685539722561)
//		@Param			t query string true "Transaction ID" Format(882d641c-01cc-442f-b894-2b51250340b5)
//		@Router			/orders/verify/ [post]
func VerifyPaymentOrder(w http.ResponseWriter, r *http.Request) {

	// Get transaction ID from URL parameter
	OrderCode := r.URL.Query().Get("s")
	if OrderCode == "" {
		utils.ErrorJSON(w, errors.New("missing parameter s"), http.StatusBadRequest)
		return
	}
	TransactionID := r.URL.Query().Get("t")
	if TransactionID == "" {
		utils.ErrorJSON(w, errors.New("missing parameter t"), http.StatusBadRequest)
		return
	}

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

	var isVerified bool
	if database.Db.IsProduction {
		// Get access token
		accessToken, err := paymentprovider.AuthenticateToVivaWallet()
		if err != nil {
			log.Error("Authentication failed: ", err)
		}

		// Verify transaction
		isVerified, err = paymentprovider.VerifyTransactionID(accessToken, TransactionID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
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
	err = database.Db.VerifyOrderAndCreatePayments(order.ID)
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
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Payment
//		@Router			/payments/ [get]
func ListPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := database.Db.ListPayments()
	respond(w, err, payments)
}



// CreatePayment godoc
//
//	 	@Summary 		Create a payment
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body database.Payment true " Create Payment"
//		@Success		200
//		@Router			/payments/ [post]
func CreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment database.Payment
	err := utils.ReadJSON(w, r, &payment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	paymentID, err := database.Db.CreatePayment(payment)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	utils.WriteJSON(w, http.StatusOK, paymentID)
}

type createPaymentsRequest struct {
	Payments []database.Payment
}

// CreatePayments godoc
//
//	 	@Summary 		Create a set of payments
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body createPaymentsRequest true " Create Payment"
//		@Success		200 {int} id
//		@Router			/payments/batch/ [post]
func CreatePayments(w http.ResponseWriter, r *http.Request) {
	var paymentBatch createPaymentsRequest
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

type createPaymentPayoutRequest struct {
	VendorLicenseID string
	Amount          int
}

// CreatePaymentPayout godoc
//
//	 	@Summary 		Create a payment from a vendor account to cash
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body createPaymentPayoutRequest true " Create Payment"
//		@Success		200 {int} id
//		@Router			/payments/payout/ [post]
func CreatePaymentPayout(w http.ResponseWriter, r *http.Request) {

		// Read data from request
		var payoutData createPaymentPayoutRequest
		err := utils.ReadJSON(w, r, &payoutData)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Get vendor
		vendor, err := database.Db.GetVendorByLicenseID(payoutData.VendorLicenseID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Get vendor account
		vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Get cash account
		cashAccount, err := database.Db.GetAccountByType("Cash")
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Check if vendor has enough money
		log.Info("Account Payout ", vendorAccount.Balance, payoutData.Amount)
		if vendorAccount.Balance < payoutData.Amount {
			utils.ErrorJSON(w, errors.New("payout amount bigger than vendor account balance"), http.StatusBadRequest)
			return
		}

		// Create payment
		payment := database.Payment{
			Sender:   vendorAccount.ID,
			Receiver: cashAccount.ID,
			Amount:   payoutData.Amount,
		}
		paymentID, err := database.Db.CreatePayment(payment)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		// Update last payout date
		// TODO: Should be transaction together with above DB request
		vendor.LastPayout = null.NewTime(time.Now(), true)
		err = database.Db.UpdateVendor(vendor.ID, vendor)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}

		utils.WriteJSON(w, http.StatusOK, paymentID)

}


// VivaWallet MVP (to be replaced by PaymentOrder API) ------------------------

// VivaWalletCreateTransactionOrder godoc
//
//	@Summary		Create a transaction order
//	@Description	Post your amount like {"Amount":100}, which equals 100 cents
//	@Tags			core
//	@accept			json
//	@Produce		json
//	@Param			amount body transactionOrder true "Amount in cents"
//	@Success		200	{array}	transactionOrderResponse
//	@Router			/vivawallet/transaction_order/ [post]
func VivaWalletCreateTransactionOrder(w http.ResponseWriter, r *http.Request) {
	var transactionOrder transactionOrder
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
	response := transactionOrderResponse{
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
//	@Param			OrderCode body transactionVerification true "Transaction ID"
//	@Success		200	{array}	transactionVerificationResponse
//	@Router			/vivawallet/transaction_verification/ [post]
func VivaWalletVerifyTransaction(w http.ResponseWriter, r *http.Request) {
	var transactionVerification transactionVerification
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
	response := transactionVerificationResponse{
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

// updateSettings godoc
//
//	 	@Summary 		Update settings
//		@Description	Update configuration data of the system
//		@Tags			core
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Settings true "Settings Representation"
//		@Success		200
//		@Router			/settings/ [put]
func updateSettings(w http.ResponseWriter, r *http.Request) {
	var settings database.Settings
	err := utils.ReadJSON(w, r, &settings)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = database.Db.UpdateSettings(settings)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil)
}

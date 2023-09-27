package handlers

import (
	"augustin/config"
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

// HelloWorld godoc
//
//	@Summary		Return HelloWorld
//	@Description	Return HelloWorld as sample API call
//	@Tags			Core
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

// HelloWorldAuth godoc
//
//	@Summary		Return HelloWorld
//	@Description	Return HelloWorld as sample API call
//	@Tags			Core, Auth
//	@Accept			json
//	@Produce		json
//	@Security		KeycloakAuth
//	@Router			/auth/hello/ [get]
//
// HelloWorld API Handler fetching data from database
func HelloWorldAuth(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, greeting)
}

// Users ----------------------------------------------------------------------

type checkLicenseIDResponse struct {
	FirstName string
}

// CheckVendorsLicenseID godoc
//
//	 	@Summary 		Check for license id
//		@Description	Check if license id exists, return first name of vendor if it does
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//	    @Param		    licenseID path string true "License ID"
//		@Success		200	{string} checkLicenseIDResponse
//		@Response		200	{string} checkLicenseIDResponse
//		@Router			/vendors/check/{licenseID}/ [get]
func CheckVendorsLicenseID(w http.ResponseWriter, r *http.Request) {
	licenseID := chi.URLParam(r, "licenseID")
	if licenseID == "" {
		utils.ErrorJSON(w, errors.New("No licenseID provided under /vendors/check/{licenseID}/"), http.StatusBadRequest)
		return
	}

	users, err := database.Db.GetVendorByLicenseID(licenseID)
	if err != nil {
		utils.ErrorJSON(w, errors.New("Wrong license id. No vendor exists with this id."), http.StatusBadRequest)
		return
	}

	response := checkLicenseIDResponse{FirstName: users.FirstName}
	utils.WriteJSON(w, http.StatusOK, response)
}

// ListVendors godoc
//
//	 	@Summary 		List Vendors
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Security		KeycloakAuth
//		@Success		200	{array}	database.Vendor
//		@Router			/vendors/ [get]
func ListVendors(w http.ResponseWriter, r *http.Request) {
	users, err := database.Db.ListVendors()
	respond(w, err, users)
}

// CreateVendor godoc
//
//	 	@Summary 		Create Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
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
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	// Create user in keycloak
	// randomPassword := utils.RandomString(10)
	// user, err := keycloak.KeycloakClient.CreateUser(vendor.Email, vendor.Email, vendor.Email, randomPassword)
	// if err != nil {
	// 	utils.ErrorJSON(w, err, http.StatusBadRequest)
	// 	return
	// }
	// keycloak.KeycloakClient.AssignRole(user, "vendor")
	respond(w, err, id)
}

// UpdateVendor godoc
//
//	 	@Summary 		Update Vendor
//		@Description	Warning: Unfilled fields will be set to default values
//		@Tags			vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param          id   path int  true  "Vendor ID"
//		@Param		    data body database.Vendor true "Vendor Representation"
//		@Router			/vendors/{id}/ [put]
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
	// todo update keycloak user
	respond(w, err, vendor)
}

// DeleteVendor godoc
//
//		@Summary 		Delete Vendor
//		@Tags			Vendors
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//	    @Param          id   path int  true  "Vendor ID"
//		@Router			/vendors/{id}/ [delete]
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
//		@Success		200	 {integer}	id
//		@Security		KeycloakAuth
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
//		@Param			id path int true "Item ID"
//	    @Param		    data body database.Item true "Item Representation"
//		@Success		200
//		@Security		KeycloakAuth
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
	fields := mForm.Value               // Values are stored in []string
	fieldsClean := make(map[string]any) // Values are stored in string
	for key, value := range fields {
		if key == "Price" {
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error(err)
			}
		} else {
			fieldsClean[key] = value[0]
		}
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
	utils.WriteJSON(w, http.StatusOK, nil)
}

// DeleteItem godoc
//
//		 	@Summary 		Delete Item
//			@Tags			Items
//			@Accept			json
//			@Produce		json
//			@Success		200
//			@Security		KeycloakAuth
//	     @Param          id   path int  true  "Item ID"
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
	Entries         []createOrderRequestEntry
	User            string
	VendorLicenseID string
}

type createOrderResponse struct {
	SmartCheckoutURL string
}

// PaymentOrders ---------------------------------------------------------------------

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
	var requestData createOrderRequest
	var order database.Order
	err := utils.ReadJSON(w, r, &requestData)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	order.Entries = make([]database.OrderEntry, len(requestData.Entries))
	for idx, entry := range requestData.Entries {
		order.Entries[idx].Item = entry.Item
		order.Entries[idx].Quantity = entry.Quantity
	}

	order.User.String = requestData.User
	vendor, err := database.Db.GetVendorByLicenseID(requestData.VendorLicenseID)
	order.Vendor = vendor.ID

	var settings database.Settings
	if settings, err = database.Db.GetSettings(); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get accounts
	var buyerAccountID int
	authenticatedUserID := r.Header.Get("X-Auth-User")
	if authenticatedUserID != "" {
		buyerAccount, err := database.Db.GetOrCreateAccountByUserID(authenticatedUserID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		buyerAccountID = buyerAccount.ID
	} else {
		buyerAccountID, err = database.Db.GetAccountTypeID("UserAnon")
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
	orgaAccountID, err := database.Db.GetAccountTypeID("Orga")
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
		order.Entries[idx].Sender = buyerAccountID
		order.Entries[idx].Receiver = vendorAccount.ID
		order.Entries[idx].Price = item.Price // Take current item price
		order.Entries[idx].IsSale = true      // Will be used for sales payment

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
				Receiver: orgaAccountID,
			}
			order.Entries = append([]database.OrderEntry{licenseItemEntry}, order.Entries...)
		}

	}

	if order.GetTotal() >= settings.MaxOrderAmount {
		utils.ErrorJSON(w, errors.New("Order amount is too high"), http.StatusBadRequest)
		return
	}

	// Submit order to vivawallet (disabled in tests)
	var OrderCode int
	if database.Db.IsProduction {
		accessToken, err := paymentprovider.AuthenticateToVivaWallet()
		if err != nil {
			log.Error("Authentication failed: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		OrderCode, err = paymentprovider.CreatePaymentOrder(accessToken, order)
		if err != nil {
			log.Error("Creating payment order failed: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
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
	url := config.Config.VivaWalletSmartCheckoutURL + strconv.Itoa(OrderCode)
	response := createOrderResponse{
		SmartCheckoutURL: url,
	}
	utils.WriteJSON(w, http.StatusOK, response)
}

type VerifyPaymentOrderResponse struct {
	TimeStamp time.Time
}

// VerifyPaymentOrder godoc
//
//	 	@Summary 		Verify Payment Order
//		@Description	Verifies order and creates payments
//		@Tags			Orders
//		@Accept			json
//		@Produce		json
//		@Success		200 {object} VerifyPaymentOrderResponse
//		@Param			s query string true "Order Code" Format(3043685539722561)
//		@Param			t query string true "Transaction ID" Format(882d641c-01cc-442f-b894-2b51250340b5)
//		@Router			/orders/verify/ [get]
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
	log.Info("Order: ", order)
	log.Info("Order timestamp: ", order.Timestamp)

	if database.Db.IsProduction {
		// Verify transaction
		_, err := paymentprovider.VerifyTransactionID(TransactionID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}
	// Make sure that transaction timestamp is not older than 15 minutes (900 seconds) to time.Now()
	if time.Now().Sub(order.Timestamp) > 900*time.Second {
		utils.ErrorJSON(w, errors.New("Transaction timestamp is older than 15 minutes"), http.StatusBadRequest)
		return
	}

	var verifyPaymentOrderResponse VerifyPaymentOrderResponse
	verifyPaymentOrderResponse.TimeStamp = order.Timestamp

	// Create response
	utils.WriteJSON(w, http.StatusOK, verifyPaymentOrderResponse)
}

// Payments (from one account to another account) -----------------------------

// ListPayments godoc
//
//	 	@Summary 		Get list of all payments
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Success		200	{array}	database.Payment
//		@Security		KeycloakAuth
//		@Security		KeycloakAuth
//		@Router			/payments/ [get]
func ListPayments(w http.ResponseWriter, r *http.Request) {

	// Get minDate and maxDate parameters
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")
	var err error
	var minDate, maxDate time.Time
	if minDateRaw != "" {
		minDate, err = time.Parse(time.RFC3339, minDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}
	if maxDateRaw != "" {
		maxDate, err = time.Parse(time.RFC3339, maxDateRaw)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}

	// Get payments with optional parameters
	payments, err := database.Db.ListPayments(minDate, maxDate)
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
//		@Success		200 {integer} id
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
//		@Success		200 {integer} id
//		@Security		KeycloakAuth
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

	// Check that amount is bigger than 0
	if payoutData.Amount <= 0 {
		utils.ErrorJSON(w, errors.New("payout amount must be bigger than 0"), http.StatusBadRequest)
		return
	}

	// Check if vendor has enough money
	if vendorAccount.Balance < payoutData.Amount {
		utils.ErrorJSON(w, errors.New("payout amount bigger than vendor account balance"), http.StatusBadRequest)
		return
	}

	// Get authenticated user
	authenticatedUserID := r.Header.Get("X-Auth-User")

	// Create payment
	payment := database.Payment{
		Sender:   vendorAccount.ID,
		Receiver: cashAccount.ID,
		Amount:   payoutData.Amount,
		AuthorizedBy: authenticatedUserID,
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

// VivaWalletCreateTransactionOrder godoc
//
//	@Summary		Webhook for VivaWallet successful transaction
//	@Description	Webhook for VivaWallet successful transaction
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionDetailRequest true "Payment Successful Response"
//	@Router			/webhooks/vivawallet/success [post]
func VivaWalletWebhookSuccess(w http.ResponseWriter, r *http.Request) {
	var paymentSuccessful paymentprovider.TransactionDetailRequest
	err := utils.ReadJSON(w, r, &paymentSuccessful)
	if err != nil {
		log.Info("Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentSuccessfulResponse(paymentSuccessful)
	if err != nil {
		log.Error(err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil)
}

// VivaWalletWebhookFailure godoc
//
//	@Summary		Webhook for VivaWallet failed transaction
//	@Description	Webhook for VivaWallet failed transaction
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionDetailRequest true "Payment Failure Response"
//	@Router			/webhooks/vivawallet/failure [post]
func VivaWalletWebhookFailure(w http.ResponseWriter, r *http.Request) {
	var paymentFailure paymentprovider.TransactionDetailRequest
	err := utils.ReadJSON(w, r, &paymentFailure)
	if err != nil {
		log.Info("Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentFailureResponse(paymentFailure)
	if err != nil {
		log.Error(err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil)
}

// VivaWalletWebhookPrice godoc
//
//	@Summary		Webhook for VivaWallet transaction prices
//	@Description	Webhook for VivaWallet transaction prices
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionPriceRequest true "Payment Price Response"
//	@Router			/webhooks/vivawallet/price [post]
func VivaWalletWebhookPrice(w http.ResponseWriter, r *http.Request) {

	log.Info("VivaWalletWebhookPrice entered")

	data, err := io.ReadAll(r.Body)

	if err != nil {

		log.Error("Reading body failed for VivaWalletWebhookPrice: ", err)

		utils.ErrorJSON(w, err, http.StatusBadRequest)

	}

	log.Info("VivaWalletWebhookPrice full request: ", string(data))
	// var paymentPrice paymentprovider.TransactionPriceRequest
	// err := utils.ReadJSON(w, r, &paymentPrice)
	// if err != nil {
	// 	log.Info("Reading JSON failed for webhook: ", err)
	// 	utils.ErrorJSON(w, err, http.StatusBadRequest)
	// 	return
	// }

	// err = paymentprovider.HandlePaymentPriceResponse(paymentPrice)
	// if err != nil {
	// 	log.Error(err)
	// 	return
	// }

	// utils.WriteJSON(w, http.StatusOK, nil)
}

// VivaWalletVerificationKey godoc
//
//	@Summary		Return VivaWallet verification key
//	@Description	Return VivaWallet verification key
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200	{array}	paymentprovider.VivaWalletVerificationKeyResponse
//	@Router			/webhooks/vivawallet/price [get]
//	@Router 		/webhooks/vivawallet/success [get]
//	@Router 		/webhooks/vivawallet/failure [get]
func VivaWalletVerificationKey(w http.ResponseWriter, r *http.Request) {
	key := config.Config.VivaWalletVerificationKey
	if key == "" {
		log.Error("VIVA_WALLET_VERIFICATION_KEY not set or can't be found")
		utils.ErrorJSON(w, errors.New("VIVA_WALLET_VERIFICATION_KEY not set or can't be found"), http.StatusBadRequest)
		return
	}
	response := paymentprovider.VivaWalletVerificationKeyResponse{Key: key}
	utils.WriteJSON(w, http.StatusOK, response)
}

// Settings -------------------------------------------------------------------

// getSettings godoc
//
//	 	@Summary 		Return settings
//		@Description	Return configuration data of the system
//		@Tags			Core
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

func updateSettingsLogo(w http.ResponseWriter, r *http.Request) (path string, err error) {

	// Get file from image field
	file, header, err := r.FormFile("Logo")
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
	if name[1] != "png" {
		log.Error(err)
		utils.ErrorJSON(w, errors.New("file type must be png"), http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Save file with name "logo"
	path = "/img/logo.png"
	err = os.WriteFile(".."+path, buf.Bytes(), 0666)
	if err != nil {
		log.Error(err)
	}
	return
}

// updateSettings godoc
//
//	 	@Summary 		Update settings
//		@Description	Update configuration data of the system. Requires multipart form. Logo has to be a png and will always be saved under "img/logo.png"
//		@Tags			Core
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Settings true "Settings Representation"
//		@Success		200
//		@Security		KeycloakAuth
//		@Router			/settings/ [put]
func updateSettings(w http.ResponseWriter, r *http.Request) {

	var err error

	// Read multipart form
	r.ParseMultipartForm(32 << 20)
	mForm := r.MultipartForm
	if mForm == nil {
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	// Handle normal fields
	var settings database.Settings
	fields := mForm.Value               // Values are stored in []string
	fieldsClean := make(map[string]any) // Values are stored in string
	for key, value := range fields {
		if key == "MaxOrderAmount" {
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
				return
			}
		} else {
			fieldsClean[key] = value[0]
		}
	}
	err = mapstructure.Decode(fieldsClean, &settings)
	if err != nil {
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	path, _ := updateSettingsLogo(w, r)
	if path != "" {
		settings.Logo = "img/logo.png"
	}

	// Save settings to database
	err = database.Db.UpdateSettings(settings)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil)
}

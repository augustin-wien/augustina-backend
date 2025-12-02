package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/go-chi/chi/v5"
	"gopkg.in/guregu/null.v4"

	"github.com/augustin-wien/augustina-backend/database"

	_ "github.com/swaggo/files"        // swagger embed files
	_ "github.com/swaggo/http-swagger" // http-swagger middleware

	"github.com/augustin-wien/augustina-backend/paymentprovider"
)

var log = utils.GetLogger()

// respond takes care of writing the response to the client
func respond(w http.ResponseWriter, err error, payload interface{}) {
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		log.Error("respond: ", err)
	}
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
	err = utils.WriteJSON(w, http.StatusOK, greeting)
	if err != nil {
		// use request-scoped logger when available so request_id is included
		logger := utils.LoggerFromContext(r.Context())
		logger.Error("HelloWorld: ", err)
	}
}

// HelloWorldAuth godoc
//
//	@Summary		Return HelloWorld
//	@Description	Return HelloWorld as sample API call
//	@Tags			Core
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
	err = utils.WriteJSON(w, http.StatusOK, greeting)
	if err != nil {
		// use request-scoped logger when available so request_id is included
		logger := utils.LoggerFromContext(r.Context())
		logger.Error("HelloWorldAuth: ", err)
	}
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
	CustomerEmail   null.String
}

type createOrderResponse struct {
	SmartCheckoutURL string
}

// PaymentOrders ---------------------------------------------------------------------

// hasDuplicitValues checks if a map has duplicate values
// Credit to: https://stackoverflow.com/a/57237165/19932351
func hasDuplicitValues(m map[int]int) bool {
	// Create empty map
	x := make(map[int]struct{})

	// Iterate over map
	for _, v := range m {
		// Add value to map by using it as key
		if _, has := x[v]; has {
			// Return true if value is already in map
			return true
		}
		// Add empty struct to map
		x[v] = struct{}{}
	}

	return false
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
	var requestData createOrderRequest
	var order database.Order
	err := utils.ReadJSON(w, r, &requestData)
	if err != nil {
		log.Error("CreatePaymentOrder: ReadJSON: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Security checks for entries
	for _, entry := range requestData.Entries {

		// 1. Check: Quantity has to be > 0 for any item except donation
		if entry.Quantity <= 0 && entry.Item != 2 {
			utils.ErrorJSON(w, errors.New("nice try! Quantity has to be greater than 0"), http.StatusBadRequest)
			return
		}

		// 2. Check: All items have to exist
		item, err := database.Db.GetItem(entry.Item)
		if err != nil {
			utils.ErrorJSON(w, errors.New("nice try! Item does not exist"), http.StatusBadRequest)
			return
		}

		// 3. Check: Transaction costs (id == 3) are not allowed to be in entries
		if entry.Item == 3 {
			utils.ErrorJSON(w, errors.New("nice try! You are not allowed to purchase this item"), http.StatusBadRequest)
			return
		}

		// 4. Check: If there is a item that needs a customerEmail, the user has to be given

		if item.LicenseItem.Valid {
			if !requestData.CustomerEmail.Valid || requestData.CustomerEmail.String == "" {
				utils.ErrorJSON(w, errors.New("you are not allowed to purchase this item without a customer email"), http.StatusBadRequest)
				return
			}
			order.CustomerEmail = requestData.CustomerEmail
		}
	}

	// 5. Check: If there is more than one entry, each item id has to be unique
	if len(requestData.Entries) > 1 {
		// Create map with item ids as keys
		uniqueItemIDs := make(map[int]int)
		for idx, entry := range requestData.Entries {
			uniqueItemIDs[idx] = entry.Item
		}
		// Check if there are duplicate item ids
		if hasDuplicitValues(uniqueItemIDs) {
			utils.ErrorJSON(w, errors.New("nice try! You are not supposed to have duplicate item ids in your order request"), http.StatusBadRequest)
			return
		}
	}

	// 6. Check: If item 2 (donation) is ordered without another item
	if len(requestData.Entries) == 1 && requestData.Entries[0].Item == 2 {
		// Throw error
		utils.ErrorJSON(w, errors.New("nice try! You are not allowed to purchase this item without another item"), http.StatusBadRequest)
		return
	}

	// Create slice of order entries depending on size of requestData.Entries
	order.Entries = make([]database.OrderEntry, len(requestData.Entries))

	// Add entries to each ordered item
	for idx, entry := range requestData.Entries {
		order.Entries[idx].Item = entry.Item
		order.Entries[idx].Quantity = entry.Quantity
	}

	// Get vendor id from license id
	vendor, err := database.Db.GetVendorByLicenseID(requestData.VendorLicenseID)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	order.Vendor = vendor.ID

	var settings *ent.Settings
	if settings, err = database.Db.GetSettings(); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Add user to order
	// TODO-Question: This line is not necessary anymore, since the user is already in the request?
	order.User.String = requestData.User

	// Get accounts
	var buyerAccountID int
	authenticatedUserID := r.Header.Get("X-Auth-User-Name")
	if authenticatedUserID != "" {
		buyerAccount, err := database.Db.GetOrCreateAccountByUserID(authenticatedUserID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		buyerAccountID = buyerAccount.ID
	} else {
		buyerAccountID, err = database.Db.GetAccountTypeID("UserAnon")
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
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

	// Amount of added license items
	// Since for each license item an additional entry is added,
	// Therefore, the index of the for loop has to be increased by 1
	licenseItemAdded := 0

	// Extend order entries
	for idx, entry := range order.Entries {
		// Increase index depending on how many license items were added
		idx = idx + licenseItemAdded
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
			// Get license item from database
			licenseItem, err := database.Db.GetItem(int(item.LicenseItem.Int64))
			if err != nil {
				utils.ErrorJSON(w, err, http.StatusBadRequest)
				return
			}
			// Define flow of money from vendor to orga
			licenseItemEntry := database.OrderEntry{
				Item:         int(item.LicenseItem.Int64),
				Quantity:     entry.Quantity,
				Price:        licenseItem.Price,
				Sender:       vendorAccount.ID,
				Receiver:     orgaAccount.ID,
				SenderName:   vendorAccount.Name,
				ReceiverName: orgaAccount.Name,
			}
			// Prepend license item without overwriting next entries
			order.Entries = append([]database.OrderEntry{licenseItemEntry}, order.Entries...)
			// Add customer email to order
			order.CustomerEmail = requestData.CustomerEmail
			// Increase licenseItemAdded by one
			licenseItemAdded++

		}

	}
	// ignore MaxOrderAmount if its 0
	if settings.MaxOrderAmount != 0 && order.GetTotal() >= settings.MaxOrderAmount {
		utils.ErrorJSON(w, errors.New("order amount is too high"), http.StatusBadRequest)
		return
	}
	// Submit order to vivawallet (disabled in tests)
	var OrderCode string = "0"
	if database.Db.IsProduction {

		if config.Config.DEBUG_payments {
			log.Info("DEBUG_payments is enabled, skipping payment order creation")
			OrderCode = strconv.Itoa(utils.GenerateRandomNumber()) // Set OrderCode to a random number for testing purposes
		} else {
			accessToken, err := paymentprovider.AuthenticateToVivaWallet()
			if err != nil {
				log.Error("Authentication failed: ", err)
				utils.ErrorJSON(w, err, http.StatusBadRequest)
				return
			}
			OrderCode, err = paymentprovider.CreatePaymentOrder(accessToken, order, requestData.VendorLicenseID)
			if err != nil {
				log.Errorf("Creating payment order failed for %+v with order id %+v failed", requestData.VendorLicenseID, order.ID, err)
				utils.ErrorJSON(w, err, http.StatusBadRequest)
				return
			}
		}
	}

	// Save order to database
	order.OrderCode.String = OrderCode
	order.OrderCode.Valid = true // This means that it is not null
	_, err = database.Db.CreateOrder(order)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Check if VivaWalletSmartCheckoutURL is set
	if config.Config.VivaWalletSmartCheckoutURL == "" {
		utils.ErrorJSON(w, errors.New("VivaWalletSmartCheckoutURL is not set"), http.StatusBadRequest)
		return
	}

	// Create response
	checkoutURL := config.Config.VivaWalletSmartCheckoutURL + OrderCode
	if config.Config.DEBUG_payments {
		log.Info("DEBUG_payments is enabled, using test URL")
		checkoutURL = "http://localhost:5173/success?t=" + OrderCode + "&s=" + OrderCode + "&lang=en-GB&eventId=0&eci=1"
	}
	// Add color code to URL
	if settings.Color == "" {
		log.Info("Color code is not set")
	} else {

		var colorCode string
		// Check if color code is valid with # at the beginning
		if settings.Color[0] == '#' {
			// Remove # from color code due to VivaWallet's policy
			colorCode = settings.Color[1:]
		} else {
			log.Info("Color code is not valid: ", settings.Color)
		}
		// Make color code lowercase
		colorCode = strings.ToLower(colorCode)

		// Add color code and necessary attachment to URL
		colorCodeAttachment := fmt.Sprintf("%s%s", "&color=", colorCode)

		// Add color code to URL
		checkoutURL = fmt.Sprintf("%s%s", checkoutURL, colorCodeAttachment)
	}

	response := createOrderResponse{
		SmartCheckoutURL: checkoutURL,
	}
	log.Debugf("CreatePaymentOrder: Created order with OrderCode %s for vendor %s", OrderCode, requestData.VendorLicenseID)
	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("CreatePaymentOrder: ", err)
	}
}

// VerifyPaymentOrderResponse is the response to VerifyPaymentOrder
type VerifyPaymentOrderResponse struct {
	TimeStamp        time.Time
	FirstName        string
	PurchasedItems   []database.OrderEntry
	TotalSum         int
	PDFDownloadLinks *[]database.PDFDownloadLinks
}

// paymentsResponse is the payload returned by ListPaymentsForPayout
type paymentsResponse struct {
	Payments []database.Payment `json:"payments"`
	Balance  int                `json:"balance"`
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
		log.Error("VerifyPaymentOrder: GetOrderByOrderCode: ", err, OrderCode)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	log.Infof("VerifyPaymentOrder: Verifying order with OrderCode %s and TransactionID %s", OrderCode, TransactionID)
	if database.Db.IsProduction && !config.Config.Development && !config.Config.DEBUG_payments {
		// Verify transaction
		_, err := paymentprovider.VerifyTransactionID(TransactionID, true)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}

	if config.Config.Development {
		// Verify transaction
		log.Infof("VerifyPaymentOrder: Verifying transaction in development mode for TransactionID %s", TransactionID)
		err = database.Db.VerifyOrderAndCreatePayments(order.ID, 0)
		if err != nil {
			log.Error("VerifyPaymentOrder: VerifyOrderAndCreatePayments: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
		}
	}

	// Make sure that transaction timestamp is not older than 15 minutes (900 seconds) to time.Now()
	// if time.Since(order.Timestamp) > 900*time.Second {
	// 	utils.ErrorJSON(w, errors.New("transaction timestamp is older than 15 minutes"), http.StatusBadRequest)
	// 	return
	// }

	var verifyPaymentOrderResponse VerifyPaymentOrderResponse

	// Declare timestamp from order
	verifyPaymentOrderResponse.TimeStamp = order.Timestamp

	for _, entry := range order.Entries {
		if entry.IsSale {
			verifyPaymentOrderResponse.PurchasedItems = append(verifyPaymentOrderResponse.PurchasedItems, entry)
		} else {
			continue
		}
	}

	// Declare total sum from order
	verifyPaymentOrderResponse.TotalSum = order.GetTotal()
	verifyPaymentOrderResponse.PDFDownloadLinks = order.GetPDFDownloadLinks()

	// Get first name of vendor from vendor id in order
	vendor, err := database.Db.GetVendor(order.Vendor)
	if err != nil {
		log.Error("Getting vendor's first name failed: ", err)
		return
	}
	settings, err := database.Db.GetSettings()
	if err != nil {
		log.Error("Getting settings failed: ", err)
		return
	}
	if settings.UseVendorLicenseIdInShop {
		verifyPaymentOrderResponse.FirstName = vendor.LicenseID.String
	} else {
		// Declare first name from vendor
		verifyPaymentOrderResponse.FirstName = vendor.FirstName
	}

	// Create response
	err = utils.WriteJSON(w, http.StatusOK, verifyPaymentOrderResponse)
	if err != nil {
		log.Error("VerifyPaymentOrder: ", err)
	}
}

// Payments (from one account to another account) -----------------------------

func parseBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}

// ListPaymentsForPayout godoc
//
//	 	@Summary 		Get list of all payments for payout
//		@Description 	Payments that do not have an associated payout
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			vendor query string false "Vendor LicenseID"
//		@Success	200	{object}	paymentsResponse
//		@Security		KeycloakAuth
//		@Security		KeycloakAuth
//		@Router			/payments/forpayout/ [get]
func ListPaymentsForPayout(w http.ResponseWriter, r *http.Request) {
	var err error
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")
	vendor := r.URL.Query().Get("vendor")
	// check if vendor exists
	if vendor != "" {
		v, err := database.Db.GetVendorByLicenseID(vendor)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		// get vendor with updated balance and use its balance in response
		vendorObj, err := database.Db.GetVendorWithBalanceUpdate(v.ID)
		if err != nil {
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
		// store vendorObj for later balance extraction via vendorObj.Balance
		_ = vendorObj
	}
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
	payments, err := database.Db.ListPaymentsForPayout(minDate, maxDate, vendor)
	if err != nil {
		respond(w, err, nil)
		return
	}

	balance := 0
	if vendor != "" {
		// try to get vendor balance (GetVendorWithBalanceUpdate already updated balance)
		if vObj, err := database.Db.GetVendorByLicenseID(vendor); err == nil {
			if vFull, err := database.Db.GetVendorWithBalanceUpdate(vObj.ID); err == nil {
				balance = vFull.Balance
			}
		}
	}

	resp := paymentsResponse{
		Payments: payments,
		Balance:  balance,
	}
	respond(w, nil, resp)
}

// ListPayments godoc
//
//		 	@Summary 		Get list of all payments
//			@Description 	Filter by date, vendor, payouts, sales. If payouts set true, all payments are removed that are not payouts. Same for sales. So sales and payouts can't be true at the same time.
//			@Tags			Payments
//			@Accept			json
//			@Produce		json
//			@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//			@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//			@Param			vendor query string false "Vendor LicenseID"
//	     @Param			payouts query bool false "Payouts only"
//	     @Param          sales query bool false "Sales only"
//			@Success		200	{array}	database.Payment
//			@Security		KeycloakAuth
//			@Security		KeycloakAuth
//			@Router			/payments/ [get]
func ListPayments(w http.ResponseWriter, r *http.Request) {
	var err error

	// Get filter parameters
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")
	payoutRaw := r.URL.Query().Get("payouts")
	salesRaw := r.URL.Query().Get("sales")
	vendor := r.URL.Query().Get("vendor")

	// Parse filter parameters
	payout, err := parseBool(payoutRaw)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	sales, err := parseBool(salesRaw)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
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

	// Get payments with filter parameters
	payments, err := database.Db.ListPayments(minDate, maxDate, vendor, payout, sales, false)
	respond(w, err, payments)
}

type ItemStatistics struct {
	ID          int
	Name        string
	SumAmount   int
	SumQuantity int
}

// PaymentsStatistics is the response to ListPaymentsStatistics
type PaymentsStatistics struct {
	From  time.Time
	To    time.Time
	Items []ItemStatistics
}

// ListPaymentsStatistics godoc
//
//	 	@Summary 		Calculate statistics of items & payments
//		@Description 	Filter by date, get statistical information, sorted by item.
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			from query string false "Minimum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Param			to query string false "Maximum date (RFC3339, UTC)" example(2006-01-02T15:04:05Z)
//		@Success		200	{array}	PaymentsStatistics
//		@Security		KeycloakAuth
//		@Security		KeycloakAuth
//		@Router			/payments/statistics/ [get]
func ListPaymentsStatistics(w http.ResponseWriter, r *http.Request) {
	var err error

	// Get filter parameters
	minDateRaw := r.URL.Query().Get("from")
	maxDateRaw := r.URL.Query().Get("to")

	// Parse filter parameters
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

	// Get items
	items, err := database.Db.ListItemsWithDisabled(false, false)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get payments with filter parameters
	payments, err := database.Db.ListPayments(minDate, maxDate, "", false, false, false)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Create map of items
	itemsMap := make(map[int]ItemStatistics)
	for _, item := range items {

		itemsMap[item.ID] = ItemStatistics{
			ID:          item.ID,
			Name:        item.Name,
			SumAmount:   0,
			SumQuantity: 0,
		}
	}

	// Create sums per item
	for _, payment := range payments {
		if !payment.Item.Valid {
			continue
		}
		itemID := int(payment.Item.Int64)
		if entry, ok := itemsMap[itemID]; ok {
			// Check if item is a donation
			if itemID == 2 {
				entry.SumQuantity += 1
			} else {
				entry.SumQuantity += payment.Quantity
			}
			entry.SumAmount += payment.Amount
			itemsMap[itemID] = entry
		} else {
			utils.ErrorJSON(w, errors.New("item not found"), http.StatusBadRequest)
			return
		}
	}

	// Create payment statistics
	var paymentsStatistics PaymentsStatistics
	paymentsStatistics.From = minDate
	paymentsStatistics.To = maxDate
	for _, item := range itemsMap {
		paymentsStatistics.Items = append(paymentsStatistics.Items, item)
	}

	respond(w, err, paymentsStatistics)
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
		log.Error("CreatePayment: ReadJSON: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	paymentID, err := database.Db.CreatePayment(payment)
	if err != nil {
		log.Error("CreatePayment: CreatePayment: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, paymentID)
	if err != nil {
		log.Error("CreatePayment: ", err)
	}
}

type createPaymentsRequest struct {
	Payments []database.Payment
}

// CreatePayments godoc
//
//	 	@Summary 		Create a set of payments
//		@Description 	TODO: This handler is not working right now and to be done for manually setting payments
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body createPaymentsRequest true "Create Payment"
//		@Success		200 {integer} id
//		@Security		KeycloakAuth
//		@Router			/payments/ [post]
func CreatePayments(w http.ResponseWriter, r *http.Request) {
	var paymentBatch createPaymentsRequest
	err := utils.ReadJSON(w, r, &paymentBatch)
	if err != nil {
		log.Error("CreatePayments: parse JSON ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = database.Db.CreatePayments(paymentBatch.Payments)
	if err != nil {
		log.Error("CreatePayments: db", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
}

type createPaymentPayoutRequest struct {
	VendorLicenseID string
	From            time.Time
	To              time.Time
}

// CreatePaymentPayout godoc
//
//	 	@Summary 		Create a payment from a vendor account to cash
//		@Tags			Payments
//		@Accept			json
//		@Produce		json
//		@Param			amount body createPaymentPayoutRequest true "Create Payment"
//		@Success		200 {integer} id
//		@Security		KeycloakAuth
//		@Router			/payments/payout/ [post]
func CreatePaymentPayout(w http.ResponseWriter, r *http.Request) {

	// Read data from request
	var payoutData createPaymentPayoutRequest
	err := utils.ReadJSON(w, r, &payoutData)
	if err != nil {
		log.Error("CreatePaymentPayout: parse JSON ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get vendor
	vendor, err := database.Db.GetVendorByLicenseID(payoutData.VendorLicenseID)
	if err != nil {
		log.Error("CreatePaymentPayout: get vendor ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get vendor account
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	if err != nil {
		log.Error("CreatePaymentPayout: get vendor account ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Get amount of money for payout
	paymentsToBePaidOut, err := database.Db.ListPaymentsForPayout(payoutData.From, payoutData.To, payoutData.VendorLicenseID)
	if err != nil {
		log.Error("CreatePaymentPayout: list payments for payout ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	var amount int
	for _, payment := range paymentsToBePaidOut {
		if payment.Receiver == vendorAccount.ID {
			amount += payment.Amount
		}
		if payment.Sender == vendorAccount.ID {
			amount -= payment.Amount
		}
	}

	// Check that amount is bigger than 0
	if amount <= 0 {
		utils.ErrorJSON(w, errors.New("payout amount must be bigger than 0"), http.StatusBadRequest)
		return
	}

	// Check if vendor has enough money
	// if vendorAccount.Balance < amount {
	// 	log.Error("CreatePaymentPayout: payout amount bigger than vendor account balance")
	// 	utils.ErrorJSON(w, errors.New("payout amount bigger than vendor account balance"), http.StatusBadRequest)
	// 	return
	// }

	// Get authenticated user
	authenticatedUserID := r.Header.Get("X-Auth-User-Name")

	// Execute payout
	paymentID, err := database.Db.CreatePaymentPayout(vendor, vendorAccount.ID, authenticatedUserID, amount, paymentsToBePaidOut)
	if err != nil {
		log.Error("CreatePaymentPayout: db", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Return success with paymentID
	err = utils.WriteJSON(w, http.StatusOK, paymentID)
	if err != nil {
		log.Error("CreatePaymentPayout: finish:", err)
	}
	log.Infof("Payout of %d cents for vendor %v was successful", amount, vendor.LicenseID)

}

type webhookResponse struct {
	Status string
}

// VivaWalletWebhookSuccess godoc
//
//	@Summary		Webhook for VivaWallet successful transaction
//	@Description	Webhook for VivaWallet successful transaction
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionSuccessRequest true "Payment Successful Response"
//	@Router			/webhooks/vivawallet/success/ [post]
func VivaWalletWebhookSuccess(w http.ResponseWriter, r *http.Request) {

	// Message to console that handler was entered
	log.Info("Transaction Success Webhook entered")

	var paymentSuccessful paymentprovider.TransactionSuccessRequest
	err := utils.ReadJSON(w, r, &paymentSuccessful)
	if err != nil {
		log.Info("VivaWalletWebhookSuccess: Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentSuccessfulResponse(paymentSuccessful)
	if err != nil {
		log.Error("VivaWalletWebhookSuccess: handle payment failed: ", err)
		return
	}

	var response webhookResponse
	response.Status = "OK"

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletWebhookSuccess: write json: ", err)
	}
}

// VivaWalletWebhookFailure godoc
//
//	@Summary		Webhook for VivaWallet failed transaction
//	@Description	Webhook for VivaWallet failed transaction
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200
//	@Param			data body paymentprovider.TransactionSuccessRequest true "Payment Failure Response"
//	@Router			/webhooks/vivawallet/failure/ [post]
func VivaWalletWebhookFailure(w http.ResponseWriter, r *http.Request) {
	var paymentFailure paymentprovider.TransactionSuccessRequest
	err := utils.ReadJSON(w, r, &paymentFailure)
	if err != nil {
		log.Info("VivaWalletWebhookFailure: Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentFailureResponse(paymentFailure)
	if err != nil {
		log.Error("VivaWalletWebhookFailure: ", err)
		return
	}

	var response webhookResponse
	response.Status = "OK"

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletWebhookFailure: ", err)
	}
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
//	@Router			/webhooks/vivawallet/price/ [post]
func VivaWalletWebhookPrice(w http.ResponseWriter, r *http.Request) {

	// Message to console that handler was entered
	log.Info("Transaction Price Webhook entered")

	var paymentPrice paymentprovider.TransactionPriceRequest
	err := utils.ReadJSON(w, r, &paymentPrice)
	if err != nil {
		log.Info("VivaWalletWebhookPrice: Reading JSON failed for webhook: ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	err = paymentprovider.HandlePaymentPriceResponse(paymentPrice)
	if err != nil {
		log.Error("VivaWalletWebhookPrice: handle payment price response failed: ", err, paymentPrice)
		return
	}

	var response webhookResponse
	response.Status = "OK"

	err = utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletWebhookPrice: failed to write json: ", err)
	}
}

// VivaWalletVerificationKey godoc
//
//	@Summary		Return VivaWallet verification key
//	@Description	Return VivaWallet verification key
//	@Tags			VivaWallet Webhooks
//	@accept			json
//	@Produce		json
//	@Success		200	{array}	paymentprovider.VivaWalletVerificationKeyResponse
//	@Router			/webhooks/vivawallet/price/ [get]
//	@Router 		/webhooks/vivawallet/success/ [get]
//	@Router 		/webhooks/vivawallet/failure/ [get]
func VivaWalletVerificationKey(w http.ResponseWriter, r *http.Request) {
	key := config.Config.VivaWalletVerificationKey
	if key == "" {
		log.Error("VIVA_WALLET_VERIFICATION_KEY not set or can't be found")
		utils.ErrorJSON(w, errors.New("VIVA_WALLET_VERIFICATION_KEY not set or can't be found"), http.StatusBadRequest)
		return
	}
	response := paymentprovider.VivaWalletVerificationKeyResponse{Key: key}
	err := utils.WriteJSON(w, http.StatusOK, response)
	if err != nil {
		log.Error("VivaWalletVerificationKey: ", err)
	}
}

// GetPDF godoc
//
//	@Summary		Get PDF path
//	@Description	Get PDF path
//	@Tags			PDF
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/pdf/ [get]
func GetPDF(w http.ResponseWriter, r *http.Request) {
	pdf, err := database.Db.GetPDF()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, pdf)
	if err != nil {
		log.Error("GetPDF: ", err)
	}
}

// GetPDFDownload godoc
//
//	@Summary		Get PDF download path
//	@Description	Get PDF download path
//	@Tags			PDF
//	@Accept			json
//	@Produce		json
//	@Success		200
//	@Router			/pdf/download/ [get]
func downloadPDF(w http.ResponseWriter, r *http.Request) {
	// Get id from URL
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Error("DownloadPDF: No id passed")
		utils.ErrorJSON(w, errors.New("missing parameter id"), http.StatusBadRequest)
		return
	}
	tx, err := database.Db.Dbpool.Begin(context.Background())
	if err != nil {
		log.Error("UpdatePdfDownload: failed to start transaction ", err)
		return
	}
	defer func() {
		err = database.DeferTx(tx, err)
		if err != nil {
			log.Error("DownloadPDF: failed to defer transaction ", err)
		}
	}()

	// Get PDF from database
	pdfDownload, err := database.Db.GetPDFDownloadTx(tx, id)
	if err != nil {
		log.Error("DownloadPDF: Failed to get PDF download from database ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfDownload.Timestamp.IsZero() {
		log.Error("DownloadPDF: Timestamp is zero")
		utils.ErrorJSON(w, errors.New("timestamp is zero"), http.StatusBadRequest)
		return
	}
	// check for expiration < 6 weeks
	if time.Until(pdfDownload.Timestamp).Hours() < -6*7*24 {
		log.Error("DownloadPDF: PDF is expired")
		utils.ErrorJSON(w, errors.New("pdf is expired"), http.StatusBadRequest)
		return
	}
	// Get PDF from database
	pdf, err := database.Db.GetPDFByID(int64(pdfDownload.PDF))
	if err != nil {
		log.Error("DownloadPDF: Failed to get PDF from database ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	pdfDownload.DownloadCount = pdfDownload.DownloadCount + 1
	pdfDownload.LastDownload = time.Now()
	err = database.Db.UpdatePdfDownloadTx(tx, pdfDownload)
	pdfDownload, err = database.Db.GetPDFDownloadTx(tx, id)

	if err != nil {
		log.Error("DownloadPDF: Failed to update downloadpdf ", err)
	}
	// send file
	http.ServeFile(w, r, pdf.Path)
}

func validatePDFLink(w http.ResponseWriter, r *http.Request) {
	// Get id from URL
	id := chi.URLParam(r, "id")
	if id == "" {
		log.Error("DownloadPDF: No id passed")
		utils.ErrorJSON(w, errors.New("missing parameter id"), http.StatusBadRequest)
		return
	}

	// Get PDF from database
	pdfDownload, err := database.Db.GetPDFDownload(id)
	if err != nil {
		log.Error("DownloadPDF: Failed to get PDF from database ", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if pdfDownload.Timestamp.IsZero() {
		log.Error("DownloadPDF: Timestamp is zero")
		utils.ErrorJSON(w, errors.New("timestamp is zero"), http.StatusBadRequest)
		return
	}
	// check for expiration < 6 weeks
	if time.Until(pdfDownload.Timestamp).Hours() < -6*7*24 {
		log.Error("DownloadPDF: PDF is expired")
		utils.ErrorJSON(w, errors.New("pdf is expired"), http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, "valid")
	if err != nil {
		log.Error("validatePDFLink: ", err)
	}
}

package paymentprovider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/augustin-wien/augustina-backend/ent"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/integrations"
	"github.com/augustin-wien/augustina-backend/utils"

	b64 "encoding/base64"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

var log = utils.GetLogger()

// AuthenticateToVivaWallet authenticates to VivaWallet and returns an access token
func AuthenticateToVivaWallet() (string, error) {
	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAccountsURL
	if apiURL == "" {
		return "", errors.New("viva wallet accounts url is not set")
	}
	resource := "/connect/token"
	jsonPost := []byte(`grant_type=client_credentials`)
	u, err := url.ParseRequestURI(apiURL)
	if err != nil {
		log.Error("Parsing URL failed: ", err)
		return "", err
	}
	u.Path = resource
	urlStr := u.String()

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Error("building request failed: ", err)
		return "", err
	}

	// Encode client credentials to base64

	if config.Config.VivaWalletSmartCheckoutClientID == "" || config.Config.VivaWalletSmartCheckoutClientKey == "" {
		err := errors.New("viva wallet smart checkout client credentials not set")
		log.Error("AuthenticateToVivaWallet: ", err)
		return "", err
	}
	clientID := config.Config.VivaWalletSmartCheckoutClientID
	clientKey := config.Config.VivaWalletSmartCheckoutClientKey

	// join id and key with a colon
	joinedIDKey := clientID + ":" + clientKey

	// encode to base64
	encodedIDKey := b64.StdEncoding.EncodeToString([]byte(joinedIDKey))

	// Create Header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodedIDKey)

	// Create a new client with a 10 second timeout
	client := http.Client{Timeout: 10 * time.Second}

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		log.Error("AuthenticateToVivaWallet: impossible to send request: ", err)
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("AuthenticateToVivaWallet: reading body failed: ", err)
		return "", err
	}

	// Unmarshal response body to struct
	var authResponse AuthenticationResponse
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		log.Error("AuthenticateToVivaWallet: unmarshalling body failed: ", err)
		return "", err
	}

	return authResponse.AccessToken, nil
}

// CreatePaymentOrder creates a payment order and returns the order code
func CreatePaymentOrder(accessToken string, order database.Order, vendorLicenseID string) (string, error) {
	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAPIURL
	if apiURL == "" {
		return "", errors.New("viva wallet api url is not set")
	}
	resource := "/checkout/v2/orders"
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String()

	// Create a new sample customer
	// TODO once registration is possible: Check if user is "UserAnon" and if not change this to customer fields
	customer := Customer{
		Email:       "",
		Fullname:    "",
		CountryCode: "AT",
		RequestLang: "de-AT",
	}

	// Create string slice listing every item name in order
	items := []string{}

	// Iterate through the order entries and retrieve item names
	for _, entry := range order.Entries {
		item, err := database.Db.GetItem(entry.Item) // Get item by ID
		if err != nil {
			log.Error("AuthenticateToVivaWallet: Item could not be found", zap.Error(err))
		}
		items = append(items, item.Name)
	}

	if config.Config.VivaWalletSourceCode == "" {
		return "", errors.New("viva wallet source code is not set")
	}

	// Create a new sample payment order
	paymentOrderRequest := PaymentOrderRequest{
		Amount:              order.GetTotal(),
		CustomerTrns:        strings.Join(items, ", ") + ", " + vendorLicenseID,
		Customer:            customer,
		PaymentTimeout:      300,
		Preauth:             false,
		AllowRecurring:      false,
		MaxInstallments:     0,
		PaymentNotification: true,
		TipAmount:           0,
		DisableExactAmount:  false,
		DisableCash:         false,
		DisableWallet:       false,
		SourceCode:          config.Config.VivaWalletSourceCode,
		MerchantTrns:        "Ein gutes Leben für alle!",
		Tags:                items,
	}

	// Create a new post request
	jsonPost, err := json.Marshal(paymentOrderRequest)
	if err != nil {
		log.Error("AuthenticateToVivaWallet: marshalling payment order failed: ", err)
		return "", err
	}

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Error("building request failed: ", err)
		return "", err
	}
	// Create Header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create a new client with a 10 second timeout
	client := http.Client{Timeout: 10 * time.Second}
	// Send the request
	res, err := client.Do(req)
	if err != nil {
		log.Error("impossible to send request: ", err)
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		body, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			log.Error("reading body failed: ", readErr)
			return "", fmt.Errorf("request failed: status %d (failed to read body: %v)", res.StatusCode, readErr)
		}
		return "", errors.New("request failed: status " + strconv.Itoa(res.StatusCode) + " " + string(body))
	}

	// Read the successful response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("reading body failed: ", err)
		return "", err
	}

	log.Debugw("VivaWallet CreatePaymentOrder response", "body", string(body))

	// Unmarshal response body to struct
	var orderCode PaymentOrderResponse
	err = json.Unmarshal(body, &orderCode)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return "", err
	}

	if orderCode.OrderCode == 0 {
		log.Errorw("VivaWallet returned empty OrderCode", "body", string(body))
		return "", errors.New("VivaWallet returned empty OrderCode")
	}

	return strconv.FormatInt(orderCode.OrderCode, 10), err

}

// HandlePaymentSuccessfulResponse handles the webhook response for a successful payment
func HandlePaymentSuccessfulResponse(paymentSuccessful TransactionSuccessRequest) (err error) {

	// Set everything up for the request
	var transactionVerificationResponse TransactionVerificationResponse

	// Retry verification to handle eventual consistency
	for i := 0; i < 5; i++ {
		transactionVerificationResponse, err = VerifyTransactionID(paymentSuccessful.EventData.TransactionID, false)
		if err != nil {
			log.Error("HandlePaymentSuccessfulResponse: TransactionID could not be verified: ", err, " for transaction ID ", paymentSuccessful.EventData.TransactionID)
		} else {
			// Check if OrderCode matches
			if transactionVerificationResponse.OrderCode == paymentSuccessful.EventData.OrderCode {
				break // Match found, proceed
			}
			log.Warnf("HandlePaymentSuccessfulResponse: order code mismatch (attempt %d): %d vs %d", i+1, transactionVerificationResponse.OrderCode, paymentSuccessful.EventData.OrderCode)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		log.Error("HandlePaymentSuccessfulResponse: Verifying transaction ID failed: ", err, " for transaction ID ", paymentSuccessful.EventData.TransactionID)
		return err
	}

	// 1. Check: Verify that webhook request and API response match all three fields

	if transactionVerificationResponse.OrderCode != paymentSuccessful.EventData.OrderCode {
		log.Errorf("HandlePaymentSuccessfulResponse: order code mismatch: %d vs %d with transaction id %s", transactionVerificationResponse.OrderCode, paymentSuccessful.EventData.OrderCode, paymentSuccessful.EventData.TransactionID)
		return errors.New("HandlePaymentSuccessfulResponse: order code mismatch")
	}

	if transactionVerificationResponse.Amount != paymentSuccessful.EventData.Amount {
		transactionToFloat64 := fmt.Sprintf("%f", transactionVerificationResponse.Amount)
		webhookToFloat64 := fmt.Sprintf("%f", paymentSuccessful.EventData.Amount)
		return errors.New("HandlePaymentSuccessfulResponse: amount mismatch: " + transactionToFloat64 + " vs " + webhookToFloat64 + " with transaction id " + paymentSuccessful.EventData.TransactionID)
	}

	if transactionVerificationResponse.StatusID != paymentSuccessful.EventData.StatusID {
		return errors.New("HandlePaymentSuccessfulResponse: status id mismatch")
	}

	// 2. Check: Verify that order can be found by ordercode and order is not already set verified in database
	var order database.Order
	// Retry getting order from database to avoid race conditions
	for i := 0; i < 5; i++ {
		order, err = database.Db.GetOrderByOrderCode(strconv.FormatInt(paymentSuccessful.EventData.OrderCode, 10))
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		log.Error("HandlePaymentSuccessfulResponse: Getting order from database failed: ", err, " for order code ", paymentSuccessful.EventData.OrderCode)
	}

	if order.Verified {
		return errors.New("order already verified")
	}

	// 3. Check: Verify amount matches with the ones in the database

	// Sum up all prices of orderentries and compare with amount
	var sum float64
	for _, entry := range order.Entries {

		// Check for TransactionCostsName
		if config.Config.TransactionCostsName == "" {
			return errors.New("transaction costs name is not set")
		}

		// Check if entry is transaction costs, which are not included in the sum
		var transactionCostItem database.Item
		transactionCostItem, err = database.Db.GetItemByName(config.Config.TransactionCostsName)
		if err != nil {
			return err
		}
		if entry.Item == transactionCostItem.ID {
			continue // Skip transaction costs
		}
		item, err := database.Db.GetItem(entry.Item) // Get item by ID
		if err != nil {
			log.Error("HandlePaymentSuccessfulResponse: Item could not be found", zap.Error(err))
		}

		if item.IsLicenseItem {
			continue // Skip license items
		}

		sum += float64(entry.Price * entry.Quantity)
	}
	// Amount would mismatch without converting to float64
	// Note: Bad consistency by VivaWallet representing amount in cents and int vs euro and float
	sum = float64(sum) / 100

	if sum != paymentSuccessful.EventData.Amount {
		return errors.New("amount mismatch: " + fmt.Sprintf("%f", sum) + " vs " + fmt.Sprintf("%f", paymentSuccessful.EventData.Amount) + " with transaction id " + paymentSuccessful.EventData.TransactionID)
	}

	// Since every check passed, now set verification status of order and create payments
	log.Info("Order has been verified and payments are being created")
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, paymentSuccessful.EventData.TransactionTypeID)
	if err != nil {
		log.Error("Verifying order and creating payments failed: ", err)
		return err
	}
	// flour
	if config.Config.FlourWebhookURL != "" {
		log.Info("Flour Webhook set, sending webhook for order", order.ID)
		// Send webhook to Flour
		go func(id int, timestamp time.Time, items []database.OrderEntry, vendorID int, totalSum int) {
			vendor, err := database.Db.GetVendor(vendorID)
			if err != nil {
				log.Error("Flour webhook: Getting vendor failed: ", err)
				return
			}

			err = integrations.SendPaymentToFlour(id, timestamp, items, vendor, totalSum)
			if err != nil {
				log.Error("Sending payment to Flour failed: ", err)
			}
		}(order.ID, order.Timestamp, order.Entries, order.Vendor, int(sum))
	}

	// Create transaction costs for Paypal
	//err = CreatePaypalTransactionCosts(paymentSuccessful, order)

	return
}

// CreatePaypalTransactionCosts creates transaction costs for Paypal payments
func CreatePaypalTransactionCosts(paymentSuccessful TransactionSuccessRequest, order database.Order) (err error) {
	// Check if VivaWalletTransactionTypeIDPaypal is set
	if config.Config.VivaWalletTransactionTypeIDPaypal == 0 {
		return errors.New("viva wallet transaction type id for paypal is not set")
	}

	// Check if order has been payed via Paypal i.e. TransactionTypeId == 48
	// Check TransactionTypeId here: https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter
	if paymentSuccessful.EventData.TransactionTypeID == config.Config.VivaWalletTransactionTypeIDPaypal {

		// // Check if PaypalPercentageCosts and PaypalFixCosts are set
		// if config.Config.PaypalPercentageCosts == 0 {
		// 	return errors.New("Env variable PaypalPercentageCosts is not set")
		// }

		// if config.Config.PaypalFixCosts == 0 {
		// 	return errors.New("Env variable PaypalFixCosts is not set")
		// }

		// // Convert percentage to multiply it with total sum i.e. 0.05 for 5% transaction costs
		// convertedPercentageCosts := (config.Config.PaypalPercentageCosts) / 100

		// // Calculate transaction costs i.e. 0.034 * 100ct + 35 = 38.4ct
		// paypalAmount := convertedPercentageCosts*float64(order.GetTotal()) + config.Config.PaypalFixCosts

		// // Given after research that Paypal rounds down on 3.4 ct to 3 ct we use math.Round
		// paypalAmount = math.Round(paypalAmount)

		// // Create order entries for transaction costs
		// // WARNING: int() always rounds down in case you stop using math.Round
		// err = CreateTransactionCostEntries(order, int(paypalAmount), "Paypal")
		// if err != nil {
		// 	return err
		// }
	}

	return

}

// VerifyTransactionID verifies that the transactionID belongs to VivaWallet and returns the transaction details
func VerifyTransactionID(transactionID string, checkDBStatus bool) (transactionVerificationResponse TransactionVerificationResponse, err error) {

	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAPIURL
	if apiURL == "" {
		return transactionVerificationResponse, errors.New("viva wallet api url is not set")
	}
	// Use transactionId from webhook to get transaction details
	resource := "/checkout/v2/transactions/" + transactionID
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String()

	// Create a new get request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		log.Error("building request failed: ", err)
		return transactionVerificationResponse, err
	}

	// Get access token
	accessToken, err := AuthenticateToVivaWallet()
	if err != nil {
		log.Error("authentication failed: ", err)
		return transactionVerificationResponse, err
	}

	// Create Header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create a new client with a 10 second timeout
	client := http.Client{Timeout: 10 * time.Second}
	// Send the request
	res, err := client.Do(req)
	if err != nil {
		log.Error("sending request failed: ", err)
		return transactionVerificationResponse, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		body, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			log.Error("reading body failed: ", readErr)
			return transactionVerificationResponse, readErr
		}
		return transactionVerificationResponse, errors.New("request failed: status " + strconv.Itoa(res.StatusCode) + " " + string(body))
	}

	// Read the response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("VerifyTransactionID: reading body failed: ", err)
		return transactionVerificationResponse, err
	}

	// Unmarshal response body to struct
	err = json.Unmarshal(body, &transactionVerificationResponse)
	if err != nil {
		log.Error("VerifyTransactionID: Unmarshalling body failed: ", err)
		return transactionVerificationResponse, err
	}

	// 1. Check: Verify that transaction has correct status, only status "F" and "MW" is allowed according to VivaWallet
	if transactionVerificationResponse.StatusID != "F" && transactionVerificationResponse.StatusID != "MW" {
		return transactionVerificationResponse, errors.New("transaction status is not successful")
	}

	// Only check isOrderVerified status if checkDBStatus is true
	if checkDBStatus {
		// 2. Check: Verify that transaction has been verified in database
		order, err := database.Db.GetOrderByOrderCode(strconv.FormatInt(transactionVerificationResponse.OrderCode, 10))
		if err != nil {
			log.Error("VerifyTransactionID: Getting order from database failed: ", err, " for order code ", transactionVerificationResponse.OrderCode)
			return transactionVerificationResponse, err
		}
		if !order.Verified {
			log.Info("VerifyTransactionID: Order has not been verified in database but needs to be for frontend call")
			return transactionVerificationResponse, errors.New("order has not been verified in database but needs to be for frontend call")
		}
	}

	return transactionVerificationResponse, err
}

// HandlePaymentFailureResponse handles the webhook response for a failed payment
func HandlePaymentFailureResponse(paymentFailure TransactionSuccessRequest) (err error) {
	// This webhook has no purpose yet, but could be used to handle failed payments
	return
}

// HandlePaymentPriceResponse handles the webhook response for a price change for now only for Card transactions
func HandlePaymentPriceResponse(paymentPrice TransactionPriceRequest) (err error) {
	//Log the request body

	// 1. Check: Verify that webhook request belongs to VivaWallet by verifying transactionID
	_, err = VerifyTransactionID(paymentPrice.EventData.TransactionID, false)
	if err != nil {
		log.Error("HandlePaymentPriceResponse: TransactionID could not be verified: ", err)
		return err
	}

	// 2. Check: Verify that order can be found by ordercode
	var order database.Order
	// Retry getting order from database to avoid race conditions
	for i := 0; i < 5; i++ {
		order, err = database.Db.GetOrderByOrderCode(strconv.FormatInt(paymentPrice.EventData.OrderCode, 10))
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		log.Error("HandlePaymentPriceResponse: Getting order from database failed: ", err, " for order code ", paymentPrice.EventData.OrderCode)
		return err
	}

	// 3. Check: If TotalCommission is 0.0, return without creating transaction costs
	if paymentPrice.EventData.TotalCommission == 0.0 {
		return
	}

	transactionCosts := int(paymentPrice.EventData.TotalCommission * 100) // Convert to cents
	// Create order entries for transaction costs
	err = CreateTransactionCostEntries(order, transactionCosts, "VivaWallet")
	if err != nil {
		log.Error("HandlePaymentPriceResponse: Creating transaction costs failed: ", err)
		return err
	}

	return
}

// CreateTransactionCostEntries creates payments and order entries to list transaction costs
func CreateTransactionCostEntries(order database.Order, transactionCosts int, paymentProvider string) (err error) {

	if config.Config.TransactionCostsName == "" {
		return errors.New("transaction costs name is not set")
	}

	// Get ID of transaction costs item
	transactionCostsItem, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	if err != nil {
		log.Error("Getting transaction costs item failed: ", err)
		return err
	}

	// Get ID of VivaWallet account
	paymentProviderAccountID, err := database.Db.GetAccountTypeID(paymentProvider)
	if err != nil {
		log.Error("Getting account type ID failed: ", err)
		return err
	}

	// Get ID of vendor account
	vendorAccount, err := database.Db.GetAccountByVendorID(order.Vendor)
	if err != nil {
		log.Error("Getting ID of vendor account failed: ", err)
		return err
	}

	// Create order entries for transaction costs
	var entries = []database.OrderEntry{
		{
			Item:     transactionCostsItem.ID,  // ID of transaction costs item
			Quantity: transactionCosts,         // Amount of transaction costs
			Sender:   vendorAccount.ID,         // ID of vendor
			Receiver: paymentProviderAccountID, // ID of Payment Provider
		},
	}

	// Create payment with order entries
	err = database.Db.CreatePayedOrderEntries(order.ID, entries)
	if err != nil {
		log.Error("Creating payment with order entries failed: ", err)
		return err
	}

	var settings *ent.Settings
	settings, err = database.Db.GetSettings()
	if err != nil {
		log.Error("Getting settings failed: ", err)
		return err
	}

	if settings.OrgaCoversTransactionCosts {

		// Get ID of Orga account
		orgaAccountID, err := database.Db.GetAccountTypeID("Orga")
		if err != nil {
			log.Error("Getting Orga account ID failed: ", err)
			return err
		}
		// Create payment for covering transaction costs by Organization
		var entries = []database.OrderEntry{
			{
				Item:     transactionCostsItem.ID, // ID of transaction costs item
				Quantity: transactionCosts,        // Amount of transaction costs
				Sender:   orgaAccountID,           // ID of Orga
				Receiver: vendorAccount.ID,        // ID of vendor
			},
		}
		// Append transaction cost entries here
		err = database.Db.CreatePayedOrderEntries(order.ID, entries)
		if err != nil {
			log.Error("Appending transaction costs failed: ", err)
			return err
		}
	}
	return
}

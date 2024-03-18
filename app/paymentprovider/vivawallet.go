package paymentprovider

import (
	"augustin/config"
	"augustin/database"
	"augustin/utils"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	b64 "encoding/base64"
	"net/http"
	"net/url"
	"time"

	"github.com/perimeterx/marshmallow"
	"go.uber.org/zap"
)

var log = utils.GetLogger()

// AuthenticateToVivaWallet authenticates to VivaWallet and returns an access token
func AuthenticateToVivaWallet() (string, error) {
	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAccountsURL
	if apiURL == "" {
		return "", errors.New("VivaWalletAccountURL is not set")
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
		log.Error("Building request failed: ", err)
	}

	// Encode client credentials to base64

	if config.Config.VivaWalletSmartCheckoutClientID == "" || config.Config.VivaWalletSmartCheckoutClientKey == "" {
		err := errors.New("VivaWalletSmartCheckoutClientCredentials not in .env or empty")
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
		log.Error("impossible to send request: ", err)
	}

	// Close the body after the function returns
	defer res.Body.Close()

	// Log the request body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Reading body failed: ", err)
		return "", err
	}

	// Unmarshal response body to struct
	var authResponse AuthenticationResponse
	_, err = marshmallow.Unmarshal(body, &authResponse)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return "", err
	}

	return authResponse.AccessToken, nil
}

// CreatePaymentOrder creates a payment order and returns the order code
func CreatePaymentOrder(accessToken string, order database.Order, vendorLicenseID string) (int, error) {
	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAPIURL
	if apiURL == "" {
		return 0, errors.New("VivaWalletApiURL is not set")
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
			log.Error("Item could not be found", zap.Error(err))
		}
		items = append(items, item.Name)
	}

	if config.Config.VivaWalletSourceCode == "" {
		return 0, errors.New("VIVA_WALLET_SOURCE_CODE is not set")
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
		log.Error("Marshalling payment order failed: ", err)
	}

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Error("Building request failed: ", err)
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
	}

	if res.StatusCode != 200 {
		// Log the request body
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Error("Reading body failed: ", err)
			body = []byte("Reading body failed: " + err.Error())
		}
		return 0, errors.New("Request failed instead received this response status code: " + strconv.Itoa(res.StatusCode) + " " + fmt.Sprint(body))
	}

	// Close the body after the function returns
	defer res.Body.Close()
	// Log the request body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Reading body failed: ", err)
		return 0, err
	}

	// Unmarshal response body to struct
	var orderCode PaymentOrderResponse
	_, err = marshmallow.Unmarshal(body, &orderCode)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return 0, err
	}

	return int(orderCode.OrderCode), err

}

// HandlePaymentSuccessfulResponse handles the webhook response for a successful payment
func HandlePaymentSuccessfulResponse(paymentSuccessful TransactionSuccessRequest) (err error) {

	// Set everything up for the request
	var transactionVerificationResponse TransactionVerificationResponse
	transactionVerificationResponse, err = VerifyTransactionID(paymentSuccessful.EventData.TransactionID, false)
	if err != nil {
		log.Error("TransactionID could not be verified: ", err)
		return err
	}

	// 1. Check: Verify that webhook request and API response match all three fields

	if transactionVerificationResponse.OrderCode != paymentSuccessful.EventData.OrderCode {
		return errors.New("OrderCode mismatch")
	}

	if transactionVerificationResponse.Amount != paymentSuccessful.EventData.Amount {
		transactionToFloat64 := fmt.Sprintf("%f", transactionVerificationResponse.Amount)
		webhookToFloat64 := fmt.Sprintf("%f", paymentSuccessful.EventData.Amount)
		return errors.New("Amount mismatch:" + transactionToFloat64 + "  vs. " + webhookToFloat64)
	}

	if transactionVerificationResponse.StatusID != paymentSuccessful.EventData.StatusID {
		return errors.New("StatusId mismatch")
	}

	// 2. Check: Verify that order can be found by ordercode and order is not already set verified in database
	order, err := database.Db.GetOrderByOrderCode(strconv.Itoa(paymentSuccessful.EventData.OrderCode))
	if err != nil {
		log.Error("Getting order from database failed: ", err)
	}

	if order.Verified {
		return errors.New("Order already verified")
	}

	// 3. Check: Verify amount matches with the ones in the database

	// Sum up all prices of orderentries and compare with amount
	var sum float64
	for _, entry := range order.Entries {

		// Check for TransactionCostsName
		if config.Config.TransactionCostsName == "" {
			return errors.New("TransactionCostsName is not set")
		}

		// Check if entry is transaction costs, which are not included in the sum
		var item database.Item
		item, err = database.Db.GetItemByName(config.Config.TransactionCostsName)
		if err != nil {
			return err
		}
		if entry.Item == item.ID {
			continue // Skip transaction costs
		}

		sum += float64(entry.Price * entry.Quantity)
	}
	// Amount would mismatch without converting to float64
	// Note: Bad consistency by VivaWallet representing amount in cents and int vs euro and float
	sum = float64(sum) / 100

	if sum != paymentSuccessful.EventData.Amount {
		return errors.New("Amount mismatch sum is" + fmt.Sprintf("%f", sum) + "  vs. payment amount" + fmt.Sprintf("%f", paymentSuccessful.EventData.Amount) + " with transaction id " + paymentSuccessful.EventData.TransactionID)
	}

	// Since every check passed, now set verification status of order and create payments
	log.Info("Order has been verified and payments are being created")
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, paymentSuccessful.EventData.TransactionTypeID)
	if err != nil {
		log.Error("Verifying order and creating payments failed: ", err)
		return err
	}

	// Create transaction costs for Paypal
	//err = CreatePaypalTransactionCosts(paymentSuccessful, order)

	return
}

// CreatePaypalTransactionCosts creates transaction costs for Paypal payments
func CreatePaypalTransactionCosts(paymentSuccessful TransactionSuccessRequest, order database.Order) (err error) {
	// Check if VivaWalletTransactionTypeIDPaypal is set
	if config.Config.VivaWalletTransactionTypeIDPaypal == 0 {
		return errors.New("Env variable VivaWalletTransactionTypeIDPaypal is not set")
	}

	// Check if order has been payed via Paypal i.e. TransactionTypeId == 48
	// Check TransactionTypeId here: https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter
	if paymentSuccessful.EventData.TransactionTypeID == config.Config.VivaWalletTransactionTypeIDPaypal {

		// Check if PaypalPercentageCosts and PaypalFixCosts are set
		if config.Config.PaypalPercentageCosts == 0 {
			return errors.New("Env variable PaypalPercentageCosts is not set")
		}

		if config.Config.PaypalFixCosts == 0 {
			return errors.New("Env variable PaypalFixCosts is not set")
		}

		// Convert percentage to multiply it with total sum i.e. 0.05 for 5% transaction costs
		convertedPercentageCosts := (config.Config.PaypalPercentageCosts) / 100

		// Calculate transaction costs i.e. 0.034 * 100ct + 35 = 38.4ct
		paypalAmount := convertedPercentageCosts*float64(order.GetTotal()) + config.Config.PaypalFixCosts

		// Given after research that aypal rounds down on 3.4 ct to 3 ct we use math.Round
		paypalAmount = math.Round(paypalAmount)

		// Create order entries for transaction costs
		// WARNING: int() always rounds down in case you stop using math.Round
		err = CreateTransactionCostEntries(order, int(paypalAmount), "Paypal")
		if err != nil {
			return err
		}
	}

	return

}

// VerifyTransactionID verifies that the transactionID belongs to VivaWallet and returns the transaction details
func VerifyTransactionID(transactionID string, checkDBStatus bool) (transactionVerificationResponse TransactionVerificationResponse, err error) {

	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAPIURL
	if apiURL == "" {
		return transactionVerificationResponse, errors.New("VivaWalletApiURL is not set")
	}
	// Use transactionId from webhook to get transaction details
	resource := "/checkout/v2/transactions/" + transactionID
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String()

	// Create a new get request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		log.Error("Building request failed: ", err)
	}

	// Get access token
	accessToken, err := AuthenticateToVivaWallet()
	if err != nil {
		log.Error("Authentication failed: ", err)
	}

	// Create Header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create a new client with a 10 second timeout
	client := http.Client{Timeout: 10 * time.Second}
	// Send the request
	res, err := client.Do(req)
	if err != nil {
		log.Error("Sending request failed: ", err)
	}

	if res.StatusCode != 200 {
		return transactionVerificationResponse, errors.New("Request failed instead received this response status code: " + strconv.Itoa(res.StatusCode))
	}

	// Close the body after the function returns
	defer res.Body.Close()
	// Log the request body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Reading body failed: ", err)
		return transactionVerificationResponse, err
	}

	// Unmarshal response body to struct
	_, err = marshmallow.Unmarshal(body, &transactionVerificationResponse)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return transactionVerificationResponse, err
	}

	// 1. Check: Verify that transaction has correct status, only status "F" and "MW" is allowed according to VivaWallet
	if transactionVerificationResponse.StatusID != "F" && transactionVerificationResponse.StatusID != "MW" {
		return transactionVerificationResponse, errors.New("Transaction status is either pending or has failed. No successfull transaction")
	}

	// Only check isOrderVerified status if checkDBStatus is true
	if checkDBStatus {
		// 2. Check: Verify that transaction has been verified in database
		order, err := database.Db.GetOrderByOrderCode(strconv.Itoa(transactionVerificationResponse.OrderCode))
		if err != nil {
			log.Error("Getting order from database failed: ", err)
			return transactionVerificationResponse, err
		}
		if !order.Verified {
			log.Info("Order has not been verified in database but needs to be for frontend call")
			return transactionVerificationResponse, errors.New("Order has not been verified in database but needs to be for frontend call")
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
		log.Error("TransactionID could not be verified: ", err)
		return err
	}

	// 2. Check: Verify that order can be found by ordercode
	order, err := database.Db.GetOrderByOrderCode(strconv.Itoa(paymentPrice.EventData.OrderCode))
	if err != nil {
		log.Error("Getting order from database failed: ", err)
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
		log.Error("Creating transaction costs failed: ", err)
		return err
	}

	return
}

// CreateTransactionCostEntries creates payments and order entries to list transaction costs
func CreateTransactionCostEntries(order database.Order, transactionCosts int, paymentProvider string) (err error) {

	if config.Config.TransactionCostsName == "" {
		return errors.New("TransactionCostsName is not set")
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

	var settings database.Settings
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

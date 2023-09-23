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
	"strconv"
	"strings"

	b64 "encoding/base64"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

var log = utils.GetLogger()

func AuthenticateToVivaWallet() (string, error) {
	// Create a new request URL using http
	apiURL := config.Config.VivaWalletAccountsURL
	if apiURL == "" {
		return "", errors.New("VivaWalletAccountURL is not set")
	}
	resource := "/connect/token"
	jsonPost := []byte(`grant_type=client_credentials`)
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String()

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Error("Building request failed: ", err)
	}

	// Encode client credentials to base64

	if config.Config.VivaWalletSmartCheckoutClientID == "" || config.Config.VivaWalletSmartCheckoutClientKey == "" {
		err := errors.New("VivaWalletSmartCheckoutClientCredentials not in .env or empty")
		log.Error(err)
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
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return "", err
	}

	return authResponse.AccessToken, nil
}

func CreatePaymentOrder(accessToken string, order database.Order) (int, error) {
	// Create a new request URL using http
	apiURL := config.Config.VivaWalletApiURL
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
		Email:       "verein@augustin.or.at",
		Fullname:    "Augustin Straßenzeitung",
		CountryCode: "AT",
		RequestLang: "de-AT",
	}

	// Create string slice listing every item name in order
	items := []string{}

	// Iterate through the order entries and retrieve item names
	for _, entry := range order.Entries {
		item, err := database.Db.GetItem(entry.Item) // Replace this with your logic
		if err != nil {
			log.Error("Item could not be found", zap.Error(err))
		}
		items = append(items, item.Name)
	}

	// Create a new sample payment order
	paymentOrderRequest := PaymentOrderRequest{
		Amount:              order.GetTotal(),
		CustomerTrns:        strings.Join(items, ", "),
		Customer:            customer,
		PaymentTimeout:      300,
		Preauth:             false,
		AllowRecurring:      false,
		MaxInstallments:     0,
		PaymentNotification: true,
		TipAmount:           0,
		DisableExactAmount:  false,
		DisableCash:         true,
		DisableWallet:       true,
		SourceCode:          utils.GetEnv("VIVA_WALLET_SOURCE_CODE", ""),
		MerchantTrns:        "Die Augustin Familie bedankt sich für Ihre Überweisung!",
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
		return 0, errors.New("Request failed instead received this response status code: " + strconv.Itoa(res.StatusCode))
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
	err = json.Unmarshal(body, &orderCode)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return 0, err
	}

	return int(orderCode.OrderCode), nil

}

func HandlePaymentSuccessfulResponse(paymentSuccessful TransactionDetailRequest) (err error) {

	// Set everything up for the request
	var transactionVerificationResponse TransactionVerificationResponse
	transactionVerificationResponse, err = VerifyTransactionID(paymentSuccessful.EventData.TransactionId)

	// 1. Check: Verify that webhook request and API response match all three fields

	if transactionVerificationResponse.OrderCode != paymentSuccessful.EventData.OrderCode {
		return errors.New("OrderCode mismatch")
	}

	if transactionVerificationResponse.Amount != paymentSuccessful.EventData.Amount {
		log.Info("Amount mismatch", zap.Float64(" transactionVerificationResponse.Amount ", transactionVerificationResponse.Amount), zap.Float64(" paymentSuccessful.EventData.Amount ", paymentSuccessful.EventData.Amount))
		transactionToFloat64 := fmt.Sprintf("%f", transactionVerificationResponse.Amount)
		webhookToFloat64 := fmt.Sprintf("%f", paymentSuccessful.EventData.Amount)
		return errors.New("Amount mismatch:" + transactionToFloat64 + "  vs. " + webhookToFloat64)
	}

	if transactionVerificationResponse.StatusId != paymentSuccessful.EventData.StatusId {
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
		sum += float64(entry.Price)
	}
	// Amount would mismatch without converting to float64
	// Note: Bad consistency by VivaWallet representing amount in cents and int vs euro and float
	sum = float64(sum) / 100

	if sum != paymentSuccessful.EventData.Amount {
		return errors.New("Amount mismatch")
	}

	// Since every check passed, now set verification status of order and create payments
	err = database.Db.VerifyOrderAndCreatePayments(order.ID)
	if err != nil {
		return err
	}

	return
}

func VerifyTransactionID(transactionID string) (transactionVerificationResponse TransactionVerificationResponse, err error) {

	// Create a new request URL using http
	apiURL := config.Config.VivaWalletApiURL
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
	err = json.Unmarshal(body, &transactionVerificationResponse)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return transactionVerificationResponse, err
	}

	// 1. Check: Verify that transaction has correct status, only status "F" and "MW" is allowed according to VivaWallet
	if transactionVerificationResponse.StatusId != "F" && transactionVerificationResponse.StatusId != "MW" {
		return transactionVerificationResponse, errors.New("Transaction status is either pending or has failed. No successfull transaction.")
	}

	return transactionVerificationResponse, nil
}

func HandlePaymentFailureResponse(paymentFailure TransactionDetailRequest) (err error) {
	log.Info("paymentFailure", paymentFailure)
	return
}

func HandlePaymentPriceResponse(paymentPrice TransactionPriceRequest) (err error) {

	// 1. Check: Verify that webhook request belongs to VivaWallet by verifying transactionID
	_, err = VerifyTransactionID(paymentPrice.EventData.TransactionId)
	if err != nil {
		return err
	}

	// 2. Check: Verify that order can be found by ordercode
	order, err := database.Db.GetOrderByOrderCode(strconv.Itoa(paymentPrice.EventData.OrderCode))
	if err != nil {
		return err
	}

	// WARNING: This logic builds upon the assumption that there is only Paypal as a payment provider, which leads to a 0.0 TotalCommission
	// TODO: Add Paypal API call to get transaction fees
	// TODO: Test if VivaWallet still sends a 0.0 TotalCommission by an amount of 10€
	// TOTHINKABOUT: Should we save which payment provider has been used for our transaction in the database i.e. Paypal or VivaWallet?
	// 3. Check: Check if TotalCommission is 0.0, which means that transaction costs are on Paypals side
	if paymentPrice.EventData.TotalCommission == 0.0 {
		// TODO: Add additional entries in order (e.g. transaction fees) for hardcoded Paypal transaction fees
		paypal_amount := config.Config.PaypalTransactionFee * order.GetTotal()
		// Create order entries for transaction costs
		err = CreateTransactionFeeEntries(order.ID, int(paypal_amount))

	} else {
		transactionFee := int(paymentPrice.EventData.TotalCommission * 100) // Convert to cents
		// Create order entries for transaction costs
		err = CreateTransactionFeeEntries(order.ID, transactionFee)

	}

	// TODO: Figure out via transaction type what type (e.g. paypal, card, etc.) of payment this is
	// https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter
	log.Info("paymentPrice", paymentPrice)
	return
}

// Create payments and order entries to list transaction fees
func CreateTransactionFeeEntries(orderID int, transactionFee int) (err error) {
	var entries = []database.OrderEntry{
		{
			Item:     TransactionFeeID, // ID of transaction fee item
			Quantity: transactionFee,   // Amount of transaction fee
			Sender:   order.Vendor,     // ID of vendor
			Receiver: VivaWalletID,     // ID of VivaWallet
		},
	}
	// append transaction cost entries here
	err = database.Db.CreatePayedOrderEntries(orderID, entries)
	if err != nil {
		return err
	}

	if config.Config.OrgaCoversTransactionCosts {
		// 4. Step: Create payment for covering transaction costs by Organization
		var entries = []database.OrderEntry{
			{
				Item:     1,              // ID of transaction fee item
				Quantity: transactionFee, // Amount of transaction fee
				Sender:   OrgaID,         // ID of vendor
				Receiver: order.Vendor,   // ID of VivaWallet
			},
		}
		// append transaction cost entries here
		err = database.Db.CreatePayedOrderEntries(orderID, entries)
		if err != nil {
			return err
		}
	}
}

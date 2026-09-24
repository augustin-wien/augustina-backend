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

// ErrTransactionNotSuccessful means VivaWallet answered normally and
// positively confirmed the transaction did not succeed - unlike every other
// error VerifyTransactionID can return (network failure, auth failure,
// unexpected HTTP status), this one is safe to treat as "definitely not
// paid" rather than "we couldn't tell".
var ErrTransactionNotSuccessful = errors.New("transaction status is not successful")

// ErrOrderNotVerifiedYet means VivaWallet confirms the payment but the success webhook has
// not verified the order yet. The frontend polls /orders/verify/ while it waits, so on its
// own this is expected and not worth an alert - the reconcile job alerts if it persists.
var ErrOrderNotVerifiedYet = errors.New("order has not been verified in database but needs to be for frontend call")

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
		// The body is only {"orderCode": ...}, so it is safe to log in full
		log.Errorw("CreatePaymentOrder: unmarshalling VivaWallet response failed", "error", err, "body", string(body))
		return "", err
	}

	if orderCode.OrderCode == 0 {
		log.Errorw("VivaWallet returned empty OrderCode", "body", string(body))
		return "", errors.New("VivaWallet returned empty OrderCode")
	}

	return orderCode.OrderCode.String(), err

}

// HandlePaymentSuccessfulResponse handles the webhook response for a successful payment
// isSimulatedTransaction reports whether a webhook may skip verification against the VivaWallet
// API because it was produced by the local development simulation.
//
// The development check is what makes this safe, and it must not be dropped. The webhook endpoint
// is unauthenticated by necessity, and anyone can create an order and read back its order code,
// so without it an attacker could post a "dev-simulation-" transaction id and have an unpaid
// order marked as paid: the simulated branch fills the verification response from the request
// itself, which makes every subsequent comparison compare the payload against itself.
func isSimulatedTransaction(transactionID string) bool {
	return config.Config.Development && strings.HasPrefix(transactionID, "dev-simulation-")
}

// logWebhookNotVerified logs why a success webhook left its order unverified. It is the one
// error log per failed delivery, carrying enough to find the order and the cause from the
// alert alone: reason is a short stable slug to search and group by.
func logWebhookNotVerified(reason string, orderID int, data EventData, err error, extra ...any) {
	fields := []any{
		"reason", reason,
		"order_id", orderID,
		"order_code", data.OrderCode.String(),
		"transaction_id", data.TransactionID,
		"status_id", data.StatusID,
		"amount", data.Amount,
		"error", err,
	}
	log.Errorw("HandlePaymentSuccessfulResponse: order not verified", append(fields, extra...)...)
}

func HandlePaymentSuccessfulResponse(paymentSuccessful TransactionSuccessRequest) (err error) {
	data := paymentSuccessful.EventData

	// Get the order before verifying with VivaWallet, so the real transaction ID
	// is stored even if verification fails. Otherwise the order keeps its
	// "manual-" placeholder ID and can never be verified against VivaWallet later.
	var order database.Order
	// Retry getting order from database to avoid race conditions
	for i := 0; i < 5; i++ {
		order, err = database.Db.GetOrderByOrderCode(data.OrderCode.String())
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		logWebhookNotVerified("order_not_found", 0, data, err)
		return err
	}

	// VivaWallet retries webhook deliveries, so a duplicate delivery for an
	// already verified order is a no-op, not an error
	if order.Verified {
		log.Infow("HandlePaymentSuccessfulResponse: order already verified, skipping", "order_id", order.ID, "transaction_id", data.TransactionID)
		return nil
	}

	err = database.Db.SetOrderTransactionID(order.ID, data.TransactionID)
	if err != nil {
		logWebhookNotVerified("set_transaction_id_failed", order.ID, data, err)
		return err
	}

	// Set everything up for the request
	var transactionVerificationResponse TransactionVerificationResponse

	// Check if this is a development simulation webhook
	isSimulation := isSimulatedTransaction(data.TransactionID)

	// Skip VivaWallet verification for simulated webhooks in development mode
	if !isSimulation {
		// Retry verification to handle eventual consistency
		for i := 0; i < 5; i++ {
			transactionVerificationResponse, err = VerifyTransactionID(data.TransactionID, false)
			if err != nil {
				log.Warnw("HandlePaymentSuccessfulResponse: verifying transaction failed, retrying", "attempt", i+1, "order_id", order.ID, "transaction_id", data.TransactionID, "error", err)
			} else {
				// Check if OrderCode matches
				if transactionVerificationResponse.OrderCode == data.OrderCode {
					break // Match found, proceed
				}
				log.Warnw("HandlePaymentSuccessfulResponse: order code mismatch, retrying", "attempt", i+1, "order_id", order.ID, "viva_order_code", transactionVerificationResponse.OrderCode.String(), "webhook_order_code", data.OrderCode.String())
			}
			time.Sleep(500 * time.Millisecond)
		}

		if err != nil {
			logWebhookNotVerified("viva_verification_failed", order.ID, data, err)
			return err
		}
	} else {
		// For simulated webhooks, use the webhook data directly
		log.Info("HandlePaymentSuccessfulResponse: Skipping VivaWallet verification for simulated webhook")
		transactionVerificationResponse.OrderCode = data.OrderCode
		transactionVerificationResponse.Amount = data.Amount
		transactionVerificationResponse.StatusID = data.StatusID
		transactionVerificationResponse.TransactionTypeID = data.TransactionTypeID
	}

	// 1. Check: Verify that webhook request and API response match all three fields

	if transactionVerificationResponse.OrderCode != data.OrderCode {
		err = errors.New("HandlePaymentSuccessfulResponse: order code mismatch")
		logWebhookNotVerified("order_code_mismatch", order.ID, data, err, "viva_order_code", transactionVerificationResponse.OrderCode.String())
		return err
	}

	if transactionVerificationResponse.Amount != data.Amount {
		transactionToFloat64 := fmt.Sprintf("%f", transactionVerificationResponse.Amount)
		webhookToFloat64 := fmt.Sprintf("%f", data.Amount)
		err = errors.New("HandlePaymentSuccessfulResponse: amount mismatch: " + transactionToFloat64 + " vs " + webhookToFloat64 + " with transaction id " + data.TransactionID)
		logWebhookNotVerified("viva_amount_mismatch", order.ID, data, err, "viva_amount", transactionVerificationResponse.Amount)
		return err
	}

	if transactionVerificationResponse.StatusID != data.StatusID {
		err = errors.New("HandlePaymentSuccessfulResponse: status id mismatch")
		logWebhookNotVerified("status_id_mismatch", order.ID, data, err, "viva_status_id", transactionVerificationResponse.StatusID)
		return err
	}

	// 2. Check: Verify amount matches with the ones in the database

	// Check for TransactionCostsName
	if config.Config.TransactionCostsName == "" {
		err = errors.New("transaction costs name is not set")
		logWebhookNotVerified("config_transaction_costs_name_missing", order.ID, data, err)
		return err
	}

	// Transaction costs are not included in the sum
	transactionCostItem, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	if err != nil {
		logWebhookNotVerified("transaction_costs_item_not_found", order.ID, data, err)
		return err
	}

	// Sum up all prices of orderentries and compare with amount. entries records how each
	// entry was counted, so a sum mismatch can be explained from the log line alone.
	var sum float64
	entries := make([]string, 0, len(order.Entries))
	for _, entry := range order.Entries {
		if entry.Item == transactionCostItem.ID {
			entries = append(entries, fmt.Sprintf("item=%d price=%d qty=%d skipped=transaction_costs", entry.Item, entry.Price, entry.Quantity))
			continue
		}
		item, err := database.Db.GetItem(entry.Item) // Get item by ID
		if err != nil {
			log.Errorw("HandlePaymentSuccessfulResponse: item could not be found", "order_id", order.ID, "item_id", entry.Item, "error", err)
		}

		if item.IsLicenseItem {
			entries = append(entries, fmt.Sprintf("item=%d price=%d qty=%d skipped=license_item", entry.Item, entry.Price, entry.Quantity))
			continue // Skip license items
		}

		entries = append(entries, fmt.Sprintf("item=%d price=%d qty=%d", entry.Item, entry.Price, entry.Quantity))
		sum += float64(entry.Price * entry.Quantity)
	}
	// Amount would mismatch without converting to float64
	// Note: Bad consistency by VivaWallet representing amount in cents and int vs euro and float
	sum = float64(sum) / 100

	if sum != data.Amount {
		err = errors.New("amount mismatch: " + fmt.Sprintf("%f", sum) + " vs " + fmt.Sprintf("%f", data.Amount) + " with transaction id " + data.TransactionID)
		logWebhookNotVerified("order_sum_mismatch", order.ID, data, err, "order_sum", sum, "entries", entries)
		return err
	}

	// Since every check passed, now set verification status of order and create payments
	err = database.Db.VerifyOrderAndCreatePayments(order.ID, data.TransactionTypeID)
	if err != nil {
		logWebhookNotVerified("verify_order_failed", order.ID, data, err)
		return err
	}
	log.Infow("HandlePaymentSuccessfulResponse: order verified", "order_id", order.ID, "order_code", data.OrderCode.String(), "transaction_id", data.TransactionID)
	// odoo
	if config.Config.OdooWebhookURL != "" {
		log.Info("Odoo Webhook set, sending webhook for order", order.ID)
		go func(id int, timestamp time.Time, items []database.OrderEntry, vendorID int, totalSum int) {
			vendor, err := database.Db.GetVendor(vendorID)
			if err != nil {
				log.Error("Odoo webhook: Getting vendor failed: ", err)
				return
			}
			err = integrations.SendPaymentToOdoo(id, timestamp, items, vendor, totalSum)
			if err != nil {
				log.Errorw("Sending payment to Odoo failed", "order_id", id, "error", err)
				if dbErr := database.Db.SetOdooSyncFailure(id, err); dbErr != nil {
					log.Errorw("Failed to record Odoo sync failure", "order_id", id, "error", dbErr)
				}
				return
			}
			if dbErr := database.Db.SetOdooSyncSuccess(id); dbErr != nil {
				log.Errorw("Failed to record Odoo sync success", "order_id", id, "error", dbErr)
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
		log.Errorw("VerifyTransactionID: sending request failed", "transaction_id", transactionID, "error", err)
		return transactionVerificationResponse, err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != 200 {
		body, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			log.Errorw("VerifyTransactionID: reading error body failed", "transaction_id", transactionID, "status", res.StatusCode, "error", readErr)
			return transactionVerificationResponse, readErr
		}
		return transactionVerificationResponse, errors.New("request failed: status " + strconv.Itoa(res.StatusCode) + " " + string(body))
	}

	// Read the response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorw("VerifyTransactionID: reading body failed", "transaction_id", transactionID, "error", err)
		return transactionVerificationResponse, err
	}

	// Unmarshal response body to struct. The body holds the customer's name and email, so
	// it is not logged: the unmarshal error already names the field and type that broke.
	err = json.Unmarshal(body, &transactionVerificationResponse)
	if err != nil {
		log.Errorw("VerifyTransactionID: unmarshalling VivaWallet response failed - payments cannot be verified until this is fixed",
			"transaction_id", transactionID, "error", err, "body_bytes", len(body))
		return transactionVerificationResponse, err
	}

	// 1. Check: Verify that transaction has correct status, only status "F" and "MW" is allowed according to VivaWallet
	if transactionVerificationResponse.StatusID != "F" && transactionVerificationResponse.StatusID != "MW" {
		log.Infow("VerifyTransactionID: transaction not successful at VivaWallet",
			"transaction_id", transactionID, "order_code", transactionVerificationResponse.OrderCode.String(), "status_id", transactionVerificationResponse.StatusID)
		return transactionVerificationResponse, ErrTransactionNotSuccessful
	}

	// Only check isOrderVerified status if checkDBStatus is true
	if checkDBStatus {
		// 2. Check: Verify that transaction has been verified in database
		order, err := database.Db.GetOrderByOrderCode(transactionVerificationResponse.OrderCode.String())
		if err != nil {
			log.Errorw("VerifyTransactionID: getting order from database failed", "transaction_id", transactionID, "order_code", transactionVerificationResponse.OrderCode.String(), "error", err)
			return transactionVerificationResponse, err
		}
		if !order.Verified {
			return transactionVerificationResponse, ErrOrderNotVerifiedYet
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
		order, err = database.Db.GetOrderByOrderCode(paymentPrice.EventData.OrderCode.String())
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

package paymentprovider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"

	b64 "encoding/base64"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

var log = utils.GetLogger()

// merchantTrns is sent with every payment order this backend creates and comes back in the
// transaction details, so it tells our transactions apart from others on the same VivaWallet
// account, such as payments at a card terminal.
const merchantTrns = "Ein gutes Leben für alle!"

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

// ErrInvalidTransactionID means the transaction id can not be a VivaWallet transaction id. It goes
// into the path of an authenticated VivaWallet API request, so it must not be able to change it.
var ErrInvalidTransactionID = errors.New("invalid transaction id")

// transactionIDPattern matches VivaWallet transaction ids, which are UUIDs
var transactionIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// placeholderTransactionIDPrefix marks the transaction id CreateOrder stores until the real one is known
const placeholderTransactionIDPrefix = "manual-"

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
		item, err := database.Db.GetItemIncludingArchived(entry.Item)
		if err != nil {
			log.Error("CreatePaymentOrder: Item could not be found", zap.Error(err))
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
		MerchantTrns:        merchantTrns,
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

// VerifyTransactionID verifies that the transactionID belongs to VivaWallet and returns the transaction details
func VerifyTransactionID(transactionID string, checkDBStatus bool) (transactionVerificationResponse TransactionVerificationResponse, err error) {
	if !transactionIDPattern.MatchString(transactionID) {
		return transactionVerificationResponse, ErrInvalidTransactionID
	}

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

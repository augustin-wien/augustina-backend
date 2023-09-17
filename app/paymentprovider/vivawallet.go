package paymentprovider

import (
	"augustin/config"
	"augustin/database"
	"augustin/utils"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"net/http"
	"net/url"
	"time"
)

var log = utils.GetLogger()

type PaymentOrderRequest struct {
	Amount              int      `json:"amount"`
	CustomerTrns        string   `json:"customerTrns"`
	Customer            Customer `json:"customer"`
	PaymentTimeout      int      `json:"paymentTimeout"`
	Preauth             bool     `json:"preauth"`
	AllowRecurring      bool     `json:"allowRecurring"`
	MaxInstallments     int      `json:"maxInstallments"`
	PaymentNotification bool     `json:"paymentNotification"`
	TipAmount           int      `json:"tipAmount"`
	DisableExactAmount  bool     `json:"disableExactAmount"`
	DisableCash         bool     `json:"disableCash"`
	DisableWallet       bool     `json:"disableWallet"`
	SourceCode          string   `json:"sourceCode"`
	MerchantTrns        string   `json:"merchantTrns"`
	Tags                []string `json:"tags"`
	// TODO: Additional fields that Aaron found in the API docs
	// PaymentMethodFees   []struct {
	// 	PaymentMethodId int `json:"paymentMethodId"`
	// 	Fee             int `json:"fee"`
	// } `json:"paymentMethodFees"`
	// CardTokens []string `json:"cardTokens"`
}

type Customer struct {
	Email       string `json:"email"`
	Fullname    string `json:"fullName"`
	Phone       string `json:"phone"`
	CountryCode string `json:"countryCode"`
	RequestLang string `json:"requestLang"`
}

type AuthenticationResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type PaymentOrderResponse struct {
	OrderCode int `json:"orderCode"`
}

type TransactionVerificationResponse struct {
	Email               string  `json:"email"`
	Amount              float64 `json:"amount"`
	OrderCode           int     `json:"orderCode"`
	StatusId            string  `json:"statusId"`
	FullName            string  `json:"fullName"`
	InsDate             string  `json:"insDate"`
	CardNumber          string  `json:"cardNumber"`
	CurrencyCode        string  `json:"currencyCode"`
	CustomerTrns        string  `json:"customerTrns"`
	MerchantTrns        string  `json:"merchantTrns"`
	TransactionTypeId   int     `json:"transactionTypeId"`
	RecurringSupport    bool    `json:"recurringSupport"`
	TotalInstallments   int     `json:"totalInstallments"`
	CardCountryCode     string  `json:"cardCountryCode"`
	CardIssuingBank     string  `json:"cardIssuingBank"`
	CurrentInstallment  int     `json:"currentInstallment"`
	CardUniqueReference string  `json:"cardUniqueReference"`
	CardTypeId          int     `json:"cardTypeId"`
}

func AuthenticateToVivaWallet() (string, error) {
	// Create a new request URL using http
	apiURL := "https://demo-accounts.vivapayments.com"
	resource := "/connect/token"
	jsonPost := []byte(`grant_type=client_credentials`)
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Error("Building request failed: ", err)
	}

	// Create Header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+config.Config.VivaWalletClientCredentials)
	if config.Config.VivaWalletClientCredentials == "" {
		err := errors.New("VivaWalletClientCredentials not in .env or empty")
		log.Error(err)
		return "", err
	}

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

func CreatePaymentOrder(accessToken string, amount int) (int, error) {
	// Create a new request URL using http
	apiURL := "https://demo-api.vivapayments.com"
	resource := "/checkout/v2/orders"
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	// Create a new sample customer
	// TODO: Change this to a real customer
	customer := Customer{
		Email:       "verein@augustin.or.at",
		Fullname:    "Augustin Straßenzeitung",
		CountryCode: "AT",
		RequestLang: "de-AT",
	}

	// Create a new sample payment order
	// TODO: Change this to a real payment order
	paymentOrderRequest := PaymentOrderRequest{
		Amount:              amount,
		CustomerTrns:        "Augustin Straßenzeitung",
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
		MerchantTrns:        "Die Augustin Familie bedankt sich für Ihre Spende!",
		//TODO: Change tags to item name
		Tags: []string{"augustin", "spende"},
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

type PriceEventData struct {
	OrderCode       int64   `json:"OrderCode"`
	MerchantId      string  `json:"MerchantId"`
	IsvFee          float64 `json:"IsvFee"`
	TransactionId   string  `json:"TransactionId"`
	CurrencyCode    string  `json:"CurrencyCode"`
	Interchange     float64 `json:"Interchange"`
	TotalCommission float64 `json:"TotalCommission"`
}

type TransactionPriceRequest struct {
	Url           string         `json:"Url"`
	EventData     PriceEventData `json:"EventData"`
	Created       time.Time      `json:"Created"`
	CorrelationId string         `json:"CorrelationId"`
	EventTypeId   int            `json:"EventTypeId"`
	Delay         *int           `json:"Delay"`
	MessageId     string         `json:"MessageId"`
	RecipientId   string         `json:"RecipientId"`
	MessageTypeId int            `json:"MessageTypeId"`
}

type TransactionDetailRequest struct {
	Url           string    `json:"Url"`
	EventData     EventData `json:"EventData"`
	Created       time.Time `json:"Created"`
	CorrelationId string    `json:"CorrelationId"`
	EventTypeId   int       `json:"EventTypeId"`
	Delay         any       `json:"Delay"`
	MessageId     string    `json:"MessageId"`
	RecipientId   string    `json:"RecipientId"`
	MessageTypeId int       `json:"MessageTypeId"`
}

type EventData struct {
	Moto                        bool     `json:"Moto"`
	BinId                       int      `json:"BinId"`
	Ucaf                        string   `json:"Ucaf"`
	Email                       string   `json:"Email"`
	Phone                       string   `json:"Phone"`
	BankId                      string   `json:"BankId"`
	Systemic                    bool     `json:"Systemic"`
	Switching                   bool     `json:"Switching"`
	ParentId                    any      `json:"ParentId"`
	Amount                      float64  `json:"Amount"`
	ChannelId                   string   `json:"ChannelId"`
	TerminalId                  int      `json:"TerminalId"`
	MerchantId                  string   `json:"MerchantId"`
	OrderCode                   int      `json:"OrderCode"`
	ProductId                   any      `json:"ProductId"`
	StatusId                    string   `json:"StatusId"`
	FullName                    string   `json:"FullName"`
	ResellerId                  any      `json:"ResellerId"`
	DualMessage                 bool     `json:"DualMessage"`
	InsDate                     string   `json:"InsDate"`
	TotalFee                    float64  `json:"TotalFee"`
	CardToken                   string   `json:"CardToken"`
	CardNumber                  string   `json:"CardNumber"`
	TipAmount                   float64  `json:"TipAmount"`
	SourceCode                  string   `json:"SourceCode"`
	SourceName                  string   `json:"SourceName"`
	Latitude                    any      `json:"Latitude"`
	Longitude                   any      `json:"Longitude"`
	CompanyName                 any      `json:"CompanyName"`
	TransactionId               string   `json:"TransactionId"`
	CompanyTitle                any      `json:"CompanyTitle"`
	PanEntryMode                string   `json:"PanEntryMode"`
	ReferenceNumber             int      `json:"ReferenceNumber"`
	ResponseCode                string   `json:"ResponseCode"`
	CurrencyCode                string   `json:"CurrencyCode"`
	OrderCulture                string   `json:"OrderCulture"`
	MerchantTrns                string   `json:"MerchantTrns"`
	CustomerTrns                string   `json:"CustomerTrns"`
	IsManualRefund              bool     `json:"IsManualRefund"`
	TargetPersonId              any      `json:"TargetPersonId"`
	TargetWalletId              any      `json:"TargetWalletId"`
	AcquirerApproved            bool     `json:"AcquirerApproved"`
	LoyaltyTriggered            bool     `json:"LoyaltyTriggered"`
	TransactionTypeId           int      `json:"TransactionTypeId"`
	AuthorizationId             string   `json:"AuthorizationId"`
	TotalInstallments           int      `json:"TotalInstallments"`
	CardCountryCode             any      `json:"CardCountryCode"`
	CardIssuingBank             any      `json:"CardIssuingBank"`
	RedeemedAmount              float64  `json:"RedeemedAmount"`
	ClearanceDate               any      `json:"ClearanceDate"`
	CurrentInstallment          int      `json:"CurrentInstallment"`
	Tags                        []string `json:"Tags"`
	BillId                      any      `json:"BillId"`
	ConnectedAccountId          any      `json:"ConnectedAccountId"`
	ResellerSourceCode          any      `json:"ResellerSourceCode"`
	ResellerSourceName          any      `json:"ResellerSourceName"`
	MerchantCategoryCode        int      `json:"MerchantCategoryCode"`
	ResellerCompanyName         any      `json:"ResellerCompanyName"`
	CardUniqueReference         string   `json:"CardUniqueReference"`
	ResellerSourceAddress       any      `json:"ResellerSourceAddress"`
	CardExpirationDate          string   `json:"CardExpirationDate"`
	ServiceId                   any      `json:"ServiceId"`
	RetrievalReferenceNumber    string   `json:"RetrievalReferenceNumber"`
	AssignedMerchantUsers       []any    `json:"AssignedMerchantUsers"`
	AssignedResellerUsers       []any    `json:"AssignedResellerUsers"`
	CardTypeId                  int      `json:"CardTypeId"`
	ResponseEventId             any      `json:"ResponseEventId"`
	ElectronicCommerceIndicator string   `json:"ElectronicCommerceIndicator"`
	OrderServiceId              int      `json:"OrderServiceId"`
	DigitalWalletId             any      `json:"DigitalWalletId"`
}

func HandlePaymentSuccessfulResponse(paymentSuccessful TransactionDetailRequest) (err error) {

	// Set everything up for the request

	// Create a new request URL using http
	apiURL := config.Config.VivaWalletURL
	if apiURL == "" {
		return errors.New("VivaWalletURL is not set")
	}
	// Use transactionId from webhook to get transaction details
	resource := "/checkout/v2/transactions/" + paymentSuccessful.EventData.TransactionId
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-api.vivapayments.com/checkout/v2/transactions/{transactionId}"

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
		return errors.New("Request failed instead received this response status code: " + strconv.Itoa(res.StatusCode))
	}

	// Close the body after the function returns
	defer res.Body.Close()
	// Log the request body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Reading body failed: ", err)
		return err
	}

	// Unmarshal response body to struct
	var transactionVerificationResponse TransactionVerificationResponse
	err = json.Unmarshal(body, &transactionVerificationResponse)
	if err != nil {
		log.Error("Unmarshalling body failed: ", err)
		return err
	}

	// 1. Check: Verify that webhook request and API response match all three fields

	if transactionVerificationResponse.OrderCode != paymentSuccessful.EventData.OrderCode {
		return errors.New("OrderCode mismatch")
	}

	if transactionVerificationResponse.Amount != paymentSuccessful.EventData.Amount {
		return errors.New("Amount mismatch")
	}

	if transactionVerificationResponse.StatusId != paymentSuccessful.EventData.StatusId {
		return errors.New("StatusId mismatch")
	}

	// TODO: Figure out what to do if statusId is not "F"
	// https://developer.vivawallet.com/integration-reference/response-codes/#statusid-parameter
	// 2. Check: Check if this is the correct statusId
	if transactionVerificationResponse.StatusId != "F" {
		return errors.New("StatusId is not F")
	}

	// 3. Check: Verify that order can be found by ordercode and order is not already set verified in database
	order, err := database.Db.GetOrderByOrderCode(strconv.Itoa(paymentSuccessful.EventData.OrderCode))
	if err != nil {
		log.Error("Getting order from database failed: ", err)
	}

	if order.Verified {
		return errors.New("Order already verified")
	}

	// 4. Check: Verify amount matches with the ones in the database

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

func HandlePaymentFailureResponse(paymentFailure TransactionDetailRequest) (err error) {
	log.Info("paymentFailure", paymentFailure)
	return
}

func HandlePaymentPriceResponse(paymentPrice TransactionPriceRequest) (err error) {
	// Add additional entries in order (e.g. transaction fees)
	// TODO: order.Entries = append(order.Entries, MyEntry)

	// TODO: Figure out via transaction type what type (e.g. paypal, card, etc.) of payment this is
	// https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter
	log.Info("paymentPrice", paymentPrice)
	return
}

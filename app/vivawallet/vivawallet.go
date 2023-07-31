package vivawallet

import (
	"augustin/utils"
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"net/http"
	"net/url"
	"time"
)

var log = utils.GetLogger()

type PaymentOrder struct {
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

type TransactionVerification struct {
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
	apiURL := "https://demo-accounts.vivapayments.com"
	resource := "/connect/token"
	jsonPost := []byte(`grant_type=client_credentials`)
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Info("Building request failed: ", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic ZTc2cnBldnR1cmZma3RuZTduMTh2MG94eWozbTZzNTMycjFxNHk0azR4eDEzLmFwcHMudml2YXBheW1lbnRzLmNvbTpxaDA4RmtVMGRGOHZNd0g3NmpHQXVCbVdpYjlXc1A=")

	// Create a new client with a 10 second timeout
	// do not forget to set timeout; otherwise, no timeout!
	client := http.Client{Timeout: 10 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		log.Info("impossible to send request: ", err)
	}

	// closes the body after the function returns
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body) // Log the request body
	if err != nil {
		log.Info("Reading body failed: ", err)
		return "", err
	}

	var authResponse AuthenticationResponse
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		log.Info("Unmarshalling body failed: ", err)
		return "", err
	}

	return authResponse.AccessToken, nil
}

func CreatePaymentOrder(accessToken string, amount int) (int, error) {
	apiURL := "https://demo-api.vivapayments.com"
	resource := "/checkout/v2/orders"
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	customer := Customer{
		Email:       "test@example.com",
		Fullname:    "Mira Mendel",
		Phone:       "1234567890",
		CountryCode: "GR",
		RequestLang: "en-GB",
	}

	paymentOrder := PaymentOrder{
		Amount:              amount,
		CustomerTrns:        "testCustomerTrns",
		Customer:            customer,
		PaymentTimeout:      300,
		Preauth:             false,
		AllowRecurring:      false,
		MaxInstallments:     0,
		PaymentNotification: true,
		TipAmount:           100,
		DisableExactAmount:  false,
		DisableCash:         true,
		DisableWallet:       true,
		SourceCode:          "6343",
		MerchantTrns:        "testMerchantTrns",
		Tags:                []string{"testTag1", "testTag2"},
	}

	jsonPost, err := json.Marshal(paymentOrder)
	if err != nil {
		log.Info("Marshalling payment order failed: ", err)
	}

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Info("Building request failed: ", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create a new client with a 10 second timeout
	// do not forget to set timeout; otherwise, no timeout!
	client := http.Client{Timeout: 10 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		log.Info("impossible to send request: ", err)
	}
	log.Info("status Code: ", res.StatusCode)
	if res.StatusCode != 200 {
		return 0, errors.New("transaction not found")
	}

	// closes the body after the function returns
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body) // Log the request body
	if err != nil {
		log.Info("Reading body failed: ", err)
		return 0, err
	}

	var orderCode PaymentOrderResponse
	err = json.Unmarshal(body, &orderCode)
	if err != nil {
		log.Info("Unmarshalling body failed: ", err)
		return 0, err
	}

	return int(orderCode.OrderCode), nil

}

func VerifyTransactionID(accessToken string, transactionID string) (success bool, err error) {
	apiURL := "https://demo-api.vivapayments.com"
	resource := "/checkout/v2/transactions/" + transactionID
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-api.vivapayments.com/checkout/v2/transactions/{transactionId}"
	log.Info("urlStr: ", urlStr)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		log.Info("Building request failed: ", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Create a new client with a 10 second timeout
	// do not forget to set timeout; otherwise, no timeout!
	client := http.Client{Timeout: 10 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		log.Info("Sending request failed: ", err)
	}
	log.Info("status Code of Verification: ", res.StatusCode)
	if res.StatusCode != 200 {
		return false, errors.New("transaction not found")
	}

	// closes the body after the function returns
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body) // Log the request body
	if err != nil {
		log.Info("Reading body failed: ", err)
		return false, err
	}

	var transactionVerification TransactionVerification
	err = json.Unmarshal(body, &transactionVerification)
	if err != nil {
		log.Info("Unmarshalling body failed: ", err)
		return false, err
	}
	return true, nil
}

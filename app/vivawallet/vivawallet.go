package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type PaymentOrder struct {
	Amount              int      `json:"amount"`
	customerTrns        string   `json:"customerTrns"`
	customer            Customer `json:"customer"`
	paymentTimeout      int      `json:"paymentTimeout"`
	preauth             bool     `json:"preauth"`
	allowRecurring      bool     `json:"allowRecurring"`
	maxInstallments     int      `json:"maxInstallments"`
	paymentNotification bool     `json:"paymentNotification"`
	tipAmount           int      `json:"tipAmount"`
	disableExactAmount  bool     `json:"disableExactAmount"`
	disableCash         bool     `json:"disableCash"`
	disableWallet       bool     `json:"disableWallet"`
	sourceCode          string   `json:"sourceCode"`
	merchantTrns        string   `json:"merchantTrns"`
	tags                []string `json:"tags"`
}

type Customer struct {
	Email       string `json:"email"`
	Fullname    string `json:"fullName"`
	Phone       string `json:"phone"`
	countryCode string `json:"countryCode"`
	requestLang string `json:"requestLang"`
}

func main() {

	body, err := authenticateToVivaWallet()
	if err != nil {
		log.Fatalf("Authentication failed: %s", err)
	}
	log.Printf("res body: %s", string(body))

}

func authenticateToVivaWallet() ([]byte, error) {
	apiURL := "https://demo-accounts.vivapayments.com"
	resource := "/connect/token"
	jsonPost := []byte(`grant_type=client_credentials`)
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Fatalf("Building request failed: %s", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic ZTc2cnBldnR1cmZma3RuZTduMTh2MG94eWozbTZzNTMycjFxNHk0azR4eDEzLmFwcHMudml2YXBheW1lbnRzLmNvbTpxaDA4RmtVMGRGOHZNd0g3NmpHQXVCbVdpYjlXc1A=")

	// Create a new client with a 10 second timeout
	// do not forget to set timeout; otherwise, no timeout!
	client := http.Client{Timeout: 10 * time.Second}
	// send the request
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("impossible to send request: %s", err)
	}
	log.Printf("status Code: %d", res.StatusCode)

	// closes the body after the function returns
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body) // Log the request body
	if err != nil {
		log.Printf("Reading body failed: %s", err)
		return nil, err
	}
	return body, nil
}

func createPaymentOrder(accessToken string) (orderCode string) {
	apiURL := "https://demo-api.vivapayments.com"
	resource := "/checkout/v2/orders"
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = resource
	urlStr := u.String() // "https://demo-accounts.vivapayments.com/connect/token"

	customer := Customer{
		Email:       "test@example.com",
		Fullname:    "Mira Mendel",
		Phone:       "1234567890",
		countryCode: "GR",
		requestLang: "en-GB",
	}

	paymentOrder := PaymentOrder{
		Amount:              1000,
		customerTrns:        "testCustomerTrns",
		customer:            customer,
		paymentTimeout:      300,
		preauth:             false,
		allowRecurring:      false,
		maxInstallments:     0,
		paymentNotification: true,
		tipAmount:           100,
		disableExactAmount:  false,
		disableCash:         true,
		disableWallet:       true,
		sourceCode:          "6343",
		merchantTrns:        "testMerchantTrns",
		tags:                []string{"testTag1", "testTag2"},
	}

	jsonPost, err := json.Marshal(paymentOrder)

	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(jsonPost))
	if err != nil {
		log.Fatalf("Building request failed: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

}

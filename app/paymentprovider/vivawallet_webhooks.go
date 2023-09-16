package paymentprovider

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

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
	log.Info("paymentSuccessful", paymentSuccessful)
	// Create a new request URL using http
	apiURL := "https://demo-api.vivapayments.com"
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
	log.Info("transactionVerificationResponse: ", string(body))

	// Unmarshal response body to struct
	// var transactionVerificationResponse TransactionVerificationResponse
	// err = json.Unmarshal(body, &transactionVerificationResponse)
	// if err != nil {
	// 	log.Error("Unmarshalling body failed: ", err)
	// 	return err
	// }
	// log.Info("transactionVerificationResponse", transactionVerificationResponse)

	return
}

func HandlePaymentFailureResponse(paymentFailure TransactionDetailRequest) (err error) {
	log.Info("paymentFailure", paymentFailure)
	return
}

func HandlePaymentPriceResponse(paymentPrice TransactionPriceRequest) (err error) {
	log.Info("paymentPrice", paymentPrice)
	return
}

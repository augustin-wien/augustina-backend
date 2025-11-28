package paymentprovider

import "time"

// PaymentOrderRequest is the request body for creating a payment order
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
}

// Customer is the customer object for the payment order
type Customer struct {
	Email       string `json:"email"`
	Fullname    string `json:"fullName"`
	Phone       string `json:"phone"`
	CountryCode string `json:"countryCode"`
	RequestLang string `json:"requestLang"`
}

// AuthenticationResponse is the response body after authenticating with VivaWallet
type AuthenticationResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// PaymentOrderResponse is the response body for creating a payment order
type PaymentOrderResponse struct {
	OrderCode int64 `json:"orderCode"`
}

// TransactionVerificationResponse is the response body for verifying a transaction
type TransactionVerificationResponse struct {
	Email               string  `json:"email"`
	Amount              float64 `json:"amount"`
	OrderCode           int64   `json:"orderCode"`
	StatusID            string  `json:"statusId"`
	FullName            string  `json:"fullName"`
	InsDate             string  `json:"insDate"`
	CardNumber          string  `json:"cardNumber"`
	CurrencyCode        string  `json:"currencyCode"`
	CustomerTrns        string  `json:"customerTrns"`
	MerchantTrns        string  `json:"merchantTrns"`
	TransactionTypeID   int     `json:"transactionTypeId"`
	RecurringSupport    bool    `json:"recurringSupport"`
	TotalInstallments   int     `json:"totalInstallments"`
	CardCountryCode     string  `json:"cardCountryCode"`
	CardIssuingBank     string  `json:"cardIssuingBank"`
	CurrentInstallment  int     `json:"currentInstallment"`
	CardUniqueReference string  `json:"cardUniqueReference"`
	CardTypeID          int     `json:"cardTypeId"`
}

// PriceEventData is the event data for the price event
type PriceEventData struct {
	CurrencyCode    string  `json:"CurrencyCode"`
	Interchange     float64 `json:"Interchange"`
	IsvFee          float64 `json:"IsvFee"`
	MerchantID      string  `json:"MerchantId"`
	OrderCode       string  `json:"OrderCode"`
	ResellerID      *any    `json:"ResellerId"`
	TotalCommission float64 `json:"TotalCommission"`
	TransactionID   string  `json:"TransactionId"`
}

// TransactionPriceRequest is the request body for the price event
type TransactionPriceRequest struct {
	CorrelationID string         `json:"CorrelationId"`
	Created       time.Time      `json:"Created"`
	Delay         *any           `json:"Delay"`
	EventData     PriceEventData `json:"EventData"`
	EventTypeID   int            `json:"EventTypeId"`
	MessageID     string         `json:"MessageId"`
	MessageTypeID int            `json:"MessageTypeId"`
	RecipientID   string         `json:"RecipientId"`
	URL           string         `json:"Url"`
}

// TransactionSuccessRequest is the request body for the success event
type TransactionSuccessRequest struct {
	CorrelationID string    `json:"CorrelationId"`
	Created       time.Time `json:"Created"`
	Delay         *any      `json:"Delay"`
	EventData     EventData `json:"EventData"`
	EventTypeID   int       `json:"EventTypeId"`
	MessageID     string    `json:"MessageId"`
	MessageTypeID int       `json:"MessageTypeId"`
	RecipientID   string    `json:"RecipientId"`
	URL           string    `json:"Url"`
}

// EventData is the event data for the success event
type EventData struct {
	AcquirerApproved            bool
	Amount                      float64
	AssignedMerchantUsers       []interface{}
	AssignedResellerUsers       []interface{}
	AuthorizationID             string
	BankID                      string
	BillID                      *interface{}
	BinID                       int
	CardCountryCode             string
	CardExpirationDate          string
	CardIssuingBank             string
	CardNumber                  string
	CardToken                   string
	CardTypeID                  int
	CardUniqueReference         string
	ChannelID                   string
	ClearanceDate               *interface{}
	CompanyName                 string
	CompanyTitle                string
	ConnectedAccountID          *interface{}
	CurrencyCode                string
	CurrentInstallment          int
	CustomerTrns                string
	DigitalWalletID             *interface{}
	DualMessage                 bool
	ElectronicCommerceIndicator string
	Email                       string
	ExternalTransactionID       *interface{}
	FullName                    string
	InsDate                     string
	IsManualRefund              bool
	Latitude                    *interface{}
	Longitude                   *interface{}
	LoyaltyTriggered            bool
	MerchantCategoryCode        int
	MerchantID                  string
	MerchantTrns                string
	Moto                        bool
	OrderCode                   int64
	OrderCulture                string
	OrderServiceID              int
	PanEntryMode                string
	ParentID                    interface{}
	Phone                       string
	ProductID                   interface{}
	RedeemedAmount              float64
	ReferenceNumber             int
	ResellerCompanyName         interface{}
	ResellerID                  interface{}
	ResellerSourceAddress       interface{}
	ResellerSourceCode          interface{}
	ResellerSourceName          interface{}
	ResponseCode                string
	ResponseEventID             interface{}
	RetrievalReferenceNumber    string
	ServiceID                   interface{}
	SourceCode                  string
	SourceName                  string
	StatusID                    string
	Switching                   bool
	Systemic                    bool
	Tags                        []string
	TargetPersonID              interface{}
	TargetWalletID              interface{}
	TerminalID                  int
	TipAmount                   float64
	TotalFee                    float64
	TotalInstallments           int
	TransactionID               string
	TransactionTypeID           int
	Ucaf                        string
}

// VivaWalletVerificationKeyResponse is the response body for the verification key
type VivaWalletVerificationKeyResponse struct {
	Key string
}

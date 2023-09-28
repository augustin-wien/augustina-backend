package paymentprovider

import "time"

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

type PriceEventData struct {
	CurrencyCode    string  `json:"CurrencyCode"`
	Interchange     float64 `json:"Interchange"`
	IsvFee          float64 `json:"IsvFee"`
	MerchantID      string  `json:"MerchantId"`
	OrderCode       int     `json:"OrderCode"`
	ResellerID      *any    `json:"ResellerId"`
	TotalCommission float64 `json:"TotalCommission"`
	TransactionId   string  `json:"TransactionId"`
}

type TransactionPriceRequest struct {
	CorrelationId string         `json:"CorrelationId"`
	Created       time.Time      `json:"Created"`
	Delay         *any           `json:"Delay"`
	EventData     PriceEventData `json:"EventData"`
	EventTypeId   int            `json:"EventTypeId"`
	MessageId     string         `json:"MessageId"`
	MessageTypeId int            `json:"MessageTypeId"`
	RecipientId   string         `json:"RecipientId"`
	Url           string         `json:"Url"`
}

type TransactionDetailRequest struct {
	CorrelationId string    `json:"CorrelationId"`
	Created       time.Time `json:"Created"`
	Delay         *any      `json:"Delay"`
	EventData     EventData `json:"EventData"`
	EventTypeId   int       `json:"EventTypeId"`
	MessageId     string    `json:"MessageId"`
	MessageTypeId int       `json:"MessageTypeId"`
	RecipientId   string    `json:"RecipientId"`
	Url           string    `json:"Url"`
}

type EventData struct {
	AcquirerApproved            bool
	Amount                      float64
	AssignedMerchantUsers       []interface{}
	AssignedResellerUsers       []interface{}
	AuthorizationId             string
	BankId                      string
	BillId                      *interface{}
	BinId                       int
	CardCountryCode             string
	CardExpirationDate          string
	CardIssuingBank             string
	CardNumber                  string
	CardToken                   string
	CardTypeId                  int
	CardUniqueReference         string
	ChannelId                   string
	ClearanceDate               *interface{}
	CompanyName                 string
	CompanyTitle                string
	ConnectedAccountId          *interface{}
	CurrencyCode                string
	CurrentInstallment          int
	CustomerTrns                string
	DigitalWalletId             *interface{}
	DualMessage                 bool
	ElectronicCommerceIndicator string
	Email                       string
	FullName                    string
	InsDate                     string
	IsManualRefund              bool
	Latitude                    *interface{}
	Longitude                   *interface{}
	LoyaltyTriggered            bool
	MerchantCategoryCode        int
	MerchantId                  string
	MerchantTrns                string
	Moto                        bool
	OrderCode                   int
	OrderCulture                string
	OrderServiceId              int
	PanEntryMode                string
	ParentId                    interface{}
	Phone                       string
	ProductId                   interface{}
	RedeemedAmount              float64
	ReferenceNumber             int
	ResellerCompanyName         interface{}
	ResellerId                  interface{}
	ResellerSourceAddress       interface{}
	ResellerSourceCode          interface{}
	ResellerSourceName          interface{}
	ResponseCode                string
	ResponseEventId             interface{}
	RetrievalReferenceNumber    string
	ServiceId                   interface{}
	SourceCode                  string
	SourceName                  string
	StatusId                    string
	Switching                   bool
	Systemic                    bool
	Tags                        []string
	TargetPersonId              interface{}
	TargetWalletId              interface{}
	TerminalId                  int
	TipAmount                   float64
	TotalFee                    float64
	TotalInstallments           int
	TransactionId               string
	TransactionTypeId           int
	Ucaf                        string
}

type VivaWalletVerificationKeyResponse struct {
	Key string
}

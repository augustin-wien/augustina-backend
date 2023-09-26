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
	ResellerID      string  `json:"ResellerId"`
	TotalCommission float64 `json:"TotalCommission"`
	TransactionId   string  `json:"TransactionId"`
}

type TransactionPriceRequest struct {
	CorrelationId string         `json:"CorrelationId"`
	Created       time.Time      `json:"Created"`
	Delay         *int           `json:"Delay"`
	EventData     PriceEventData `json:"EventData"`
	EventTypeId   int            `json:"EventTypeId"`
	MessageId     string         `json:"MessageId"`
	MessageTypeId int            `json:"MessageTypeId"`
	RecipientId   string         `json:"RecipientId"`
	Url           string         `json:"Url"`
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

type VivaWalletVerificationKeyResponse struct {
	Key string
}

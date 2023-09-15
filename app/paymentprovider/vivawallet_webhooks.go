package paymentprovider

import "time"

type PaymentSuccessfulRequest struct {
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

func HandlePaymentSuccessfulResponse(paymentSuccessful PaymentSuccessfulRequest) (err error) {
	log.Info("paymentSuccessful", paymentSuccessful)
	return
}

func HandlyPaymentFailureResponse(paymentFailure PaymentSuccessfulRequest) (err error) {
	log.Info("paymentFailure", paymentFailure)
	return
}

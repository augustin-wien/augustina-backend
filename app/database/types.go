package database

import (
	"github.com/jackc/pgx/v5/pgtype"
)

// Attributes have to be uppercase to be exported
// Pgtype types are required if a field is nullable

type User struct {
	ID	   	   int32
	KeycloakID string
	UrlID	   string  // This will be the QR code
	LicenseID  string
	FirstName  string
	LastName   string
	IsVendor   bool
	IsAdmin    bool

}

type Account struct {
	ID     int32
	Name   string
	Person User
	// TODO: Balance
}

type Item struct {
	ID         int32
	Name       string
	Price      float32
	IsEditable bool
	Image      string
}

type PaymentType struct {
	ID   int32
	Name string
}

type PaymentBatch struct {
	ID 	     int64
	Payments []Payment
}

type Payment struct {
	ID        	 int64
	Timestamp 	 pgtype.Timestamp
	Sender    	 int32
	Receiver  	 int32
	Type      	 int32
	Amount    	 float32
	AuthorizedBy pgtype.Int4
	Item	     pgtype.Int4
	PaymentBatch pgtype.Int8

}

type Settings struct {
	ID         int32
	Color      string
	Logo       string
	Newspaper  Item
	RefundFees bool
}

type DatabaseInterface interface {
	GetHelloWorld() (string, error)
	GetPayments() ([]Payment, error)
	CreatePaymentType(pt PaymentType) (pgtype.Int4, error)
	CreateAccount(account Account) (pgtype.Int4, error)
	CreatePayments(payments []Payment) error
	UpdateSettings(settings Settings) error
	GetItems() ([]Item, error)
	GetSettings() (Settings, error)
	GetVendorSettings() (string, error)
}

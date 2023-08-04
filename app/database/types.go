package database

import (
	"gopkg.in/guregu/null.v4"
)

// Attributes have to be uppercase to be exported
// Pgtype types are required if a field is nullable

type Vendor struct {
	ID	   	   int32
	KeycloakID string
	UrlID	   string  // This is used for the QR code
	LicenseID  string
	FirstName  string
	LastName   string
    Email	   string
	LastPayout null.Time
	Account    int32
	Balance    float32 // This is joined in from the account
}

type Account struct {
	ID      int32
	Name    string
	Balance float32
}

type Item struct {
	ID         int32
	Name       string
	Description string
	Price      float32
	Image      string
}

type PaymentBatch struct {
	ID 	     int64
	Payments []Payment
}

type Payment struct {
	ID        	 int64
	Timestamp 	 null.Time
	Sender    	 int32
	Receiver  	 int32
	Amount    	 float32
	AuthorizedBy string
	Item	     null.Int
	Batch        null.Int
}

type Settings struct {
	ID         int32
	Color      string
	Logo       string
	Newspaper  Item
	RefundFees bool
}

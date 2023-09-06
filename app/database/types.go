package database

import (
	"gopkg.in/guregu/null.v4"
)

// Attributes have to be uppercase to be exported

type Vendor struct {
	ID	   	   int32
	KeycloakID string
	UrlID	   string  // This is used for the QR code
	LicenseID  string
	FirstName  string
	LastName   string
    Email	   string
	LastPayout null.Time `swaggertype:"string" format:"date-time"`
	Balance    float32 // This is joined in from the account
}

type Account struct {
	ID      int32
	Name    string
	Balance float32
	Type    string
	Vendor  null.Int `swaggertype:"integer"`
}

type Item struct {
	ID         	int32
	Name       	string
	Description string
	Price      	float32
	Image      	string
}

type PaymentBatch struct {
	ID 	     int64
	Payments []Payment
}

type Payment struct {
	ID        	 int64
	Timestamp 	 null.Time `swaggertype:"string" format:"date-time"`
	Sender    	 int32
	Receiver  	 int32
	Amount    	 float32
	AuthorizedBy string
	Item	     null.Int `swaggertype:"integer"`
	Batch        null.Int `swaggertype:"integer"`
}

type Settings struct {
	ID         int32
	Color      string
	Logo       string
	MainItem   null.Int `swaggertype:"integer"`
	RefundFees bool
}

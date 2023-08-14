package database

import (
	"time"

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
	Archived   	bool
}

type OrderItem struct {
	ID         	int64
	ItemID     	int32
	Quantity   	int32
	Price      	float32  // Price at time of purchase
}

type PaymentOrder struct {
	ID         		int64
	TransactionID 	string
	Verified 		bool
	Timestamp  		time.Time
	Vendor	 		int32
	OrderItems     	[]OrderItem
}

type Payment struct {
	ID        	 	int64
	Timestamp 	 	time.Time
	Sender    	 	int32
	Receiver  	 	int32
	Amount    	 	float32
	AuthorizedBy 	string
	PaymentOrderID	null.Int `swaggertype:"integer"`
	OrderItemID  	null.Int `swaggertype:"integer"`
}

type Settings struct {
	ID         int32
	Color      string
	Logo       string
	MainItem   null.Int `swaggertype:"integer"`
	RefundFees bool
}

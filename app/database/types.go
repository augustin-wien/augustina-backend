package database

import (
	"time"

	"gopkg.in/guregu/null.v4"
)

// Attributes have to be uppercase to be exported
// All prices are in cents

type Vendor struct {
	ID	   	   int
	KeycloakID string
	UrlID	   string  // This is used for the QR code
	LicenseID  string
	FirstName  string
	LastName   string
    Email	   string
	LastPayout null.Time `swaggertype:"string" format:"date-time"`
	Balance    int // This is joined in from the account
}

type Account struct {
	ID      int
	Name    string
	Balance int
	Type    string
	User    null.String // Keycloak UUID if type = user_auth
	Vendor  null.Int `swaggertype:"integer"`
}

type Item struct {
	ID         	int
	Name       	string
	Description string
	Price      	int  // Price in cents
	Image      	string
	LicenseItem	null.Int  // License has to be bought before item
	Archived   	bool
}

type Order struct {
	ID         		int
	TransactionID 	string
	Verified 		bool
	Timestamp  		time.Time
	User			null.String  // Keycloak UUID if user is authenticated
	Vendor	 		int
	Entries     	[]OrderEntry
}

type OrderEntry struct {
	ID         	int
	Item     	int
	Quantity   	int
	Price      	int  // Price at time of purchase in cents
	Sender		int
	Receiver	int
}

type Payment struct {
	ID        	 	int
	Timestamp 	 	time.Time
	Sender    	 	int
	Receiver  	 	int
	Amount    	 	int
	AuthorizedBy 	string
	Order			null.Int `swaggertype:"integer"`
	OrderEntry  	null.Int `swaggertype:"integer"`
}

type Settings struct {
	ID         int
	Color      string
	Logo       string
	MainItem   null.Int `swaggertype:"integer"`
	RefundFees bool
}

package database

import (
	"time"

	"gopkg.in/guregu/null.v4"
)

// Attributes have to be uppercase to be exported
// All prices are in cents

// Vendor is a struct that is used for the vendor table
type Vendor struct {
	ID          int
	KeycloakID  string
	URLID       string // This is used for the QR code
	LicenseID   null.String
	FirstName   string
	LastName    string
	Email       string
	LastPayout  null.Time `swaggertype:"string" format:"date-time"`
	Balance     int       // This is joined in from the account
	IsDisabled  bool
	Longitude   float64
	Latitude    float64
	Address     string
	PLZ         string
	Location    string
	WorkingTime string
	Lang        string
}

// Account is a struct that is used for the account table
type Account struct {
	ID      int
	Name    string
	Balance int
	Type    string
	User    null.String // Keycloak UUID if type = user_auth
	Vendor  null.Int    `swaggertype:"integer"`
}

// Item is a struct that is used for the item table
type Item struct {
	ID          int
	Name        string
	Description string
	Price       int // Price in cents
	Image       string
	LicenseItem null.Int // License has to be bought before item
	Archived    bool
}

// Order is a struct that is used for the order table
type Order struct {
	ID                int
	OrderCode         null.String
	TransactionID     string
	Verified          bool
	TransactionTypeID int
	Timestamp         time.Time
	User              null.String // Keycloak UUID if user is authenticated
	Vendor            int
	Entries           []OrderEntry
}

// OrderEntry is a struct that is used for the order_entry table
type OrderEntry struct {
	ID       int
	Item     int
	Quantity int
	Price    int // Price at time of purchase in cents
	Sender   int
	Receiver int
	IsSale   bool // Whether to include this item in sales payment
}

// Payment is a struct that is used for the payment table
type Payment struct {
	ID           int
	Timestamp    time.Time
	Sender       int
	Receiver     int
	Amount       int
	AuthorizedBy string
	Order        null.Int `swaggertype:"integer"`
	OrderEntry   null.Int `swaggertype:"integer"`
	IsSale       bool
}

// Settings is a struct that is used for the settings table
type Settings struct {
	ID                         int
	Color                      string
	Logo                       string
	MainItem                   null.Int `swaggertype:"integer"`
	RefundFees                 bool
	MaxOrderAmount             int
	OrgaCoversTransactionCosts bool
}

// DBSettings is a struct that is used for the dbsettings table
type DBSettings struct {
	ID            int
	IsInitialized bool
}

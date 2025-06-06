package database

import (
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	"gopkg.in/guregu/null.v4"
)

// Attributes have to be uppercase to be exported
// All prices are in cents

// Vendor is a struct that is used for the vendor table
type Vendor struct {
	ID               int
	AccountProofUrl  null.String
	KeycloakID       string
	UrlID            string // This is used for the QR code
	LicenseID        null.String
	FirstName        string
	LastName         string
	Email            string
	LastPayout       null.Time `swaggertype:"string" format:"date-time"`
	Balance          int       // This is joined in from the account
	IsDisabled       bool
	IsDeleted        bool
	Locations        []*ent.Location
	Comments         []*ent.Comment
	Language         string
	Telephone        string
	RegistrationDate string
	VendorSince      string
	OnlineMap        bool
	HasSmartphone    bool
	HasBankAccount   bool
	Debt             string
}

// Location is a struct that is used for the location table
type Location struct {
	ID          int
	Name        string
	Address     string
	Longitude   float64
	Latitude    float64
	ZIP         string
	WorkingTime string
}

type Comment struct {
	ID         int
	Comment    string
	Warning    bool
	CreatedAt  time.Time
	ResolvedAt time.Time
}

// Account is a struct that is used for the account table
type Account struct {
	ID      int
	Name    string
	Balance int
	Type    string
	User    null.String // Keycloak UUID
	Vendor  null.Int    `swaggertype:"integer"`
}

// Item is a struct that is used for the item table
type Item struct {
	ID            int
	Archived      bool
	ItemColor     null.String // Color of the item in the webshop
	ItemTextColor null.String // Text color of the item in the webshop
	Description   string
	Name          string
	Image         string
	IsLicenseItem bool
	IsPDFItem     bool
	ItemOrder     int // Order in the webshop
	LicenseGroup  null.String
	LicenseItem   null.Int // License has to be bought before item
	PDF           null.Int
	Price         int // Price in cents
}

// Order is a struct that is used for the order table
type Order struct {
	ID                int
	OrderCode         null.String
	TransactionID     string
	Verified          bool
	TransactionTypeID int
	Timestamp         time.Time
	User              null.String `db:"userid"` // Keycloak UUID if user is authenticated
	Vendor            int
	Entries           []OrderEntry
	CustomerEmail     null.String
}

// OrderEntry is a struct that is used for the order_entry table
type OrderEntry struct {
	ID           int
	Item         int
	Quantity     int
	Price        int // Price at time of purchase in cents
	Sender       int
	Receiver     int
	SenderName   string
	ReceiverName string
	IsSale       bool // Whether to include this item in sales payment
}

// Payment is a struct that is used for the payment table
type Payment struct {
	ID           int
	Timestamp    time.Time
	Sender       int
	Receiver     int
	SenderName   null.String // JOIN from Sender Account
	ReceiverName null.String // JOIN from Receiver Account
	Amount       int
	AuthorizedBy string
	Order        null.Int `swaggertype:"integer" db:"paymentorder"`
	OrderEntry   null.Int `swaggertype:"integer"`
	IsSale       bool
	Payout       null.Int  `swaggertype:"integer"` // Connected payout payment
	IsPayoutFor  []Payment `db:"ispayoutfor"`      // Connected payout payment
	Item         null.Int  `swaggertype:"integer"`
	Quantity     int
	Price        int // Price at time of purchase in cents
}

// Settings is a struct that is used for the settings table
type Settings struct {
	ID                         int
	AGBUrl                     string
	Color                      string
	FontColor                  string
	Logo                       string
	Favicon                    string
	MainItem                   null.Int `swaggertype:"integer"`
	MaxOrderAmount             int
	MainItemName               null.String
	MainItemPrice              null.Int
	MainItemDescription        null.String
	MainItemImage              null.String
	MaintainanceModeHelpUrl    string
	NewspaperName              string
	QRCodeLogoImgUrl           string `db:"qrcodelogourl"`
	QRCodeSettings             string `db:"qrcodesettings"`
	QRCodeEnableLogo           bool
	QRCodeUrl                  string
	OrgaCoversTransactionCosts bool
	VendorEmailPostfix         string
	VendorNotFoundHelpUrl      string
	WebshopIsClosed            bool
	MapCenterLat               float64
	MapCenterLong              float64
	UseVendorLicenseIdInShop   bool `db:"usevendorlicenseidinshop"`
}

// DBSettings is a struct that is used for the dbsettings table
type DBSettings struct {
	ID            int
	IsInitialized bool
}

type PDF struct {
	ID        int
	Path      string
	Timestamp time.Time
}

type PDFDownload struct {
	ID            int
	LinkID        string
	PDF           int
	Timestamp     time.Time
	EmailSent     bool
	OrderID       null.Int
	LastDownload  time.Time
	DownloadCount int
	ItemID        null.Int
}

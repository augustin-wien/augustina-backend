package database

import (
	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"

	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// InitiateAccounts creates default settings if they don't exist
func (db *Database) InitiateAccounts() (err error) {
	for _, account := range []string{"Cash", "Orga", "UserAnon", "VivaWallet", "Paypal"} {
		_, err = Db.CreateSpecialVendorAccount(Vendor{
			LicenseID: null.StringFrom(account),
			Email:     account + "@augustina.cc",
		})
		if err != nil {
			log.Error("InitiateAccounts: ", err)
			return err
		}
	}
	return err
}

// InitiateItems creates default item for transaction costs
func (db *Database) InitiateItems() (err error) {

	newspaper := Item{
		Name:          "Zeitung",
		Description:   "Aktuelle Zeitungsausgabe",
		Price:         300,
		Archived:      false,
		IsLicenseItem: false,
		IsPDFItem:     false,
		LicenseGroup:  null.NewString("analog_edition", true),
	}

	if config.Config.DonationName == "" {
		log.Error("DonationName is not set")
		return
	}
	donation := Item{
		Name:        config.Config.DonationName,
		Description: "Spende pro Einkauf",
		Price:       1,
		Archived:    false,
	}
	if config.Config.TransactionCostsName == "" {
		log.Error("TransactionCostsName is not set")
		return
	}

	transactionCost := Item{
		Name:        config.Config.TransactionCostsName,
		Description: "Transaktionskosten der Zahlungsanbieter",
		Price:       1,
		Archived:    false,
	}

	// Create newspaper
	_, err = db.CreateItem(newspaper)
	if err != nil {
		log.Error("InitiateItem creation failed for newspaper ", zap.Error(err))
		return
	}

	// Create donation
	_, err = db.CreateItem(donation)
	if err != nil {
		log.Error("InitiateItem creation failed for donation ", zap.Error(err))
		return
	}

	// Create item transaction cost
	_, err = db.CreateItem(transactionCost)
	if err != nil {
		log.Error("InitiateItem creation failed for transaction costs ", zap.Error(err))
		return err
	}
	return
}

// UpdateInitialSettings creates test settings for the application
func (db *Database) UpdateInitialSettings() (err error) {
	settings := ent.Settings{
		Color:     "#F45793",
		FontColor: "#FFFFFF",
		Logo:      "img/logo.png",
		Edges: ent.SettingsEdges{MainItem: &ent.Item{
			ID: 1,
		}},
		MaxOrderAmount:             5000,
		OrgaCoversTransactionCosts: true,
		VendorEmailPostfix:         "@example.com",
		WebshopIsClosed:            false,
		NewspaperName:              "Zeitung",
		QRCodeUrl:                  "https://localhost:5134/",
		MaintainanceModeHelpUrl:    "https://example.com",
		AGBUrl:                     "https://example.com/AGB",
		MapCenterLat:               48.2083,
		MapCenterLong:              16.3731,
		QRCodeLogoImgUrl:           "/img/logo.png",
		VendorNotFoundHelpUrl:      "https://example.com/vendornotfound",
	}

	err = db.UpdateSettings(&settings)
	if err != nil {
		log.Error("InitiateSettings creation failed ", zap.Error(err))
	}

	return err
}

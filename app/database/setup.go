package database

import (
	"augustin/config"

	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"
)

// InitiateAccounts creates default settings if they don't exist
func (db *Database) InitiateAccounts() (err error) {
	for _, account := range []string{"Cash", "Orga", "UserAnon", "VivaWallet", "Paypal"} {
		_, err = Db.CreateAccount(Account{
			Name: account,
			Type: account,
		})
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}

// InitiateItems creates default item for transaction costs
func (db *Database) InitiateItems() (err error) {

	newspaper := Item{
		Name:        "Zeitung",
		Description: "Aktuelle Zeitungsausgabe",
		Price:       300,
		Archived:    false,
	}

	donation := Item{
		Name:        "Spende",
		Description: "Spende pro Einkauf",
		Price:       1,
		Archived:    false,
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

// CreateDevSettings creates test settings for the application
func (db *Database) UpdateInitialSettings() (err error) {
	settings := Settings{
		Color:                      "#FF0000",
		Logo:                       "/img/logo.png",
		MainItem:                   null.IntFrom(0),
		MaxOrderAmount:             5000,
		OrgaCoversTransactionCosts: true,
	}

	err = db.UpdateSettings(settings)
	if err != nil {
		log.Error("InitiateSettings creation failed ", zap.Error(err))
	}

	return err
}

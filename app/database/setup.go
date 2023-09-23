package database

import "go.uber.org/zap"

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

// InitiateItems creates default item for transaction fees
func (db *Database) InitiateItems() (err error) {
	transactionFee := Item{
		Name:        "Transaktionskosten",
		Description: "Transaktionskosten der Zahlungsanbieter",
		Price:       1,
		Archived:    false,
	}

	// Create item transaction fee
	_, err = db.CreateItem(transactionFee)
	if err != nil {
		log.Error("InitiateItem creation failed for transaction fees", zap.Error(err))
		return err
	}
	return
}

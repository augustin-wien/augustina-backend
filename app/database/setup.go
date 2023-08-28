package database

func (db *Database) InitiateAccounts() (err error) {
	// Loop through "cash", "user_anon"
	log.Info("InitiateAccounts")
	for _, account := range []string{"Cash", "Orga", "UserAnon"} {
		_, err = Db.CreateAccount(Account{
			Name:  account,
			Type:  account,
		})
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return err
}

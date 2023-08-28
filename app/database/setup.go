package database

func (db *Database) InitiateAccounts() (err error) {
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

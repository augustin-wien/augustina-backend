package database

func (db *Database) InitiateAccounts() (err error) {
	for _, account := range []string{"Cash", "Orga", "UserAnon"} {
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

// Create default settings if they don't exist
func (db *Database) InitiateSettings() (err error) {
	// _, err = db.Dbpool.Exec(context.Background(), `
	// INSERT INTO Settings (ID) VALUES (1);
	// `)
	// if err != nil {
	// 	log.Error(err)
	// 	return err
	// }
	return err
}

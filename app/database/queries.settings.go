package database

import "context"

// Settings (singleton) -------------------------------------------------------

// InitiateSettings creates default settings if they don't exist
func (db *Database) InitiateSettings() (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	INSERT INTO Settings (ID) VALUES (1)
	ON CONFLICT (ID) DO NOTHING;
	`)
	if err != nil {
		log.Error("InitiateSettings: ", err)
		return err
	}
	return err
}

// GetSettings returns the settings from the database
func (db *Database) GetSettings() (Settings, error) {
	var settings Settings
	err := db.Dbpool.QueryRow(context.Background(), `
	SELECT Settings.ID, Color, FontColor, Logo, MainItem, MaxOrderAmount, OrgaCoversTransactionCosts, Name, Price, Description, Image, WebshopIsClosed, VendorNotFoundHelpUrl, MaintainanceModeHelpUrl, VendorEmailPostfix, NewspaperName, QRCodeUrl, QRCodeLogoImgUrl, AGBUrl, MapCenterLat, MapCenterLong, UseVendorLicenseIdInShop, Favicon, QrCodeSettings, QRCodeEnableLogo  from Settings LEFT JOIN Item ON Item.ID = MainItem LIMIT 1
	`).Scan(&settings.ID, &settings.Color, &settings.FontColor,
		&settings.Logo, &settings.MainItem, &settings.MaxOrderAmount,
		&settings.OrgaCoversTransactionCosts, &settings.MainItemName,
		&settings.MainItemPrice, &settings.MainItemDescription, &settings.MainItemImage,
		&settings.WebshopIsClosed, &settings.VendorNotFoundHelpUrl,
		&settings.MaintainanceModeHelpUrl, &settings.VendorEmailPostfix,
		&settings.NewspaperName, &settings.QRCodeUrl, &settings.QRCodeLogoImgUrl,
		&settings.AGBUrl, &settings.MapCenterLat, &settings.MapCenterLong,
		&settings.UseVendorLicenseIdInShop,
		&settings.Favicon, &settings.QRCodeSettings, &settings.QRCodeEnableLogo,
	)
	if err != nil {
		log.Error("GetSettings: ", err)
	}
	return settings, err
}

// UpdateSettings updates the settings in the database
func (db *Database) UpdateSettings(settings Settings) (err error) {

	tx, err := db.Dbpool.Begin(context.Background())
	defer func() { err = DeferTx(tx, err) }()
	if err != nil {
		log.Error("UpdateSettings failed to access db pool: ", err)
		return err
	}

	_, err = tx.Exec(context.Background(), `
	UPDATE Settings
	SET Color = $1, FontColor = $2, Logo = $3, MainItem = $4, MaxOrderAmount = $5, OrgaCoversTransactionCosts = $6, WebshopIsClosed = $7, VendorNotFoundHelpUrl = $8, MaintainanceModeHelpUrl = $9, VendorEmailPostfix = $10, NewspaperName = $11, QRCodeUrl = $12, QRCodeLogoImgUrl = $13, AGBUrl = $14, MapCenterLat = $15, MapCenterLong = $16, UseVendorLicenseIdInShop = $17, Favicon = $18, QrCodeSettings = $19, QRCodeEnableLogo = $20
	WHERE ID = 1`,
		settings.Color, settings.FontColor, settings.Logo,
		settings.MainItem, settings.MaxOrderAmount,
		settings.OrgaCoversTransactionCosts, settings.WebshopIsClosed,
		settings.VendorNotFoundHelpUrl, settings.MaintainanceModeHelpUrl, settings.VendorEmailPostfix,
		settings.NewspaperName, settings.QRCodeUrl,
		settings.QRCodeLogoImgUrl, settings.AGBUrl, settings.MapCenterLat, settings.MapCenterLong,
		settings.UseVendorLicenseIdInShop,
		&settings.Favicon, &settings.QRCodeSettings, &settings.QRCodeEnableLogo,
	)
	if err != nil {
		log.Error("db UpdateSettings: ", err)
	}
	return err
}

// DBSettings -----------------------------------------------------------------

// InitiateDBSettings creates default settings if they don't exist
func (db *Database) InitiateDBSettings() (err error) {
	_, err = db.Dbpool.Exec(context.Background(), `
	INSERT INTO DBSettings (ID, isInitialized) VALUES (1, false)
	ON CONFLICT (ID) DO NOTHING;
	`)
	if err != nil {
		log.Error("InitiateDBSettings: ", err)
		return err
	}
	return err
}

// UpdateDBSettings updates the settings in the database
func (db *Database) UpdateDBSettings(dbsettings DBSettings) (err error) {
	_, err = db.Dbpool.Query(context.Background(), `
	UPDATE DBSettings
	SET isInitialized = $1
	WHERE ID = 1
	`, dbsettings.IsInitialized)
	if err != nil {
		log.Error("UpdateDBSettings: ", err)
	}
	return err
}

// GetDBSettings returns the settings from the database
func (db *Database) GetDBSettings() (DBSettings, error) {
	var dbsettings DBSettings
	err := db.Dbpool.QueryRow(context.Background(), `
	SELECT * from DBSettings LIMIT 1
	`).Scan(&dbsettings.ID, &dbsettings.IsInitialized)
	if err != nil {
		log.Error("GetDBSettings: ", err)
	}
	return dbsettings, err
}

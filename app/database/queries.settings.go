package database

import (
	"context"

	"github.com/augustin-wien/augustina-backend/ent"
)

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
func (db *Database) GetSettings() (*ent.Settings, error) {
	s, err := db.EntClient.Settings.Query().WithMainItem().Only(context.Background())
	if err != nil {
		log.Error("GetSettings: ", err)
		return nil, err
	}
	return s, err
}

// UpdateSettings updates the settings in the database
func (db *Database) UpdateSettings(settings *ent.Settings) (err error) {

	tx, err := db.Dbpool.Begin(context.Background())
	defer func() { err = DeferTx(tx, err) }()
	if err != nil {
		log.Error("UpdateSettings failed to access db pool: ", err)
		return err
	}
	log.Debug("UpdateSettings: ", settings.UseTipInsteadOfDonation)
	// Update the settings in the database
	_, err = db.EntClient.Settings.UpdateOneID(1).
		SetAGBUrl(settings.AGBUrl).
		SetColor(settings.Color).
		SetFontColor(settings.FontColor).
		SetLogo(settings.Logo).
		SetMaxOrderAmount(settings.MaxOrderAmount).
		SetOrgaCoversTransactionCosts(settings.OrgaCoversTransactionCosts).
		SetWebshopIsClosed(settings.WebshopIsClosed).
		SetVendorNotFoundHelpUrl(settings.VendorNotFoundHelpUrl).
		SetMaintainanceModeHelpUrl(settings.MaintainanceModeHelpUrl).
		SetVendorEmailPostfix(settings.VendorEmailPostfix).
		SetNewspaperName(settings.NewspaperName).
		SetQRCodeUrl(settings.QRCodeUrl).
		SetQRCodeLogoImgUrl(settings.QRCodeLogoImgUrl).
		SetMapCenterLat(settings.MapCenterLat).
		SetMapCenterLong(settings.MapCenterLong).
		SetUseVendorLicenseIdInShop(settings.UseVendorLicenseIdInShop).
		SetFavicon(settings.Favicon).
		SetQRCodeSettings(settings.QRCodeSettings).
		SetQRCodeEnableLogo(settings.QRCodeEnableLogo).
		SetUseTipInsteadOfDonation(settings.UseTipInsteadOfDonation).
		Save(context.Background())
	if err != nil {
		log.Error("UpdateSettings: ", err)
		return err
	}
	// update main item
	if settings.Edges.MainItem != nil {
		_, err = db.EntClient.Settings.UpdateOneID(1).
			SetMainItemID(settings.Edges.MainItem.ID).
			Save(context.Background())
		if err != nil {
			log.Error("UpdateSettings main item failed: ", err, settings.Edges.MainItem)
			return err
		}
	}
	return nil
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

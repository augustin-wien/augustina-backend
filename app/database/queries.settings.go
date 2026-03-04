package database

import (
	"context"

	"github.com/augustin-wien/augustina-backend/ent"
	entdbsettings "github.com/augustin-wien/augustina-backend/ent/dbsettings"
	entsettings "github.com/augustin-wien/augustina-backend/ent/settings"
)

// Settings (singleton) -------------------------------------------------------

// InitiateSettings creates default settings if they don't exist
func (db *Database) InitiateSettings() (err error) {
	// Attempt to create. If ID 1 exists, it will fail (unique constraint).
	// Since we don't have OnConflict easy access without custom dialect/features,
	// checking existence is safer migration.
	exists, err := db.EntClient.Settings.Query().Where(entsettings.ID(1)).Exist(context.Background())
	if err != nil {
		log.Error("InitiateSettings: ", err)
		return err
	}
	if !exists {
		_, err = db.EntClient.Settings.Create().SetID(1).Save(context.Background())
		if err != nil {
			log.Error("InitiateSettings: ", err)
			return err
		}
	}
	return nil
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

	tx, err := db.EntClient.Tx(context.Background())
	if err != nil {
		log.Error("UpdateSettings failed to start tx: ", err)
		return err
	}
	defer func() {
		if v, ok := err.(interface{ Rollback() error }); ok && v != nil {
			tx.Rollback()
		}
	}()

	log.Debug("UpdateSettings: ", settings.UseTipInsteadOfDonation)

	// Prepare update builder
	update := tx.Settings.UpdateOneID(1).
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
		SetShopLanding(settings.ShopLanding).
		SetDigitalItemsUrl(settings.DigitalItemsUrl)

	// Update main item if present
	if settings.Edges.MainItem != nil {
		update.SetMainItemID(settings.Edges.MainItem.ID)
	}

	err = update.Exec(context.Background())
	if err != nil {
		log.Error("UpdateSettings: ", err)
		return err
	}

	return tx.Commit()
}

// InitiateDBSettings creates default settings if they don't exist
func (db *Database) InitiateDBSettings() (err error) {
	exists, err := db.EntClient.DBSettings.Query().Where(entdbsettings.ID(1)).Exist(context.Background())
	if err != nil {
		log.Error("InitiateDBSettings: check failed ", err)
		return err
	}

	if !exists {
		_, err = db.EntClient.DBSettings.Create().SetID(1).SetIsInitialized(false).Save(context.Background())
		if err != nil {
			log.Error("InitiateDBSettings: ", err)
			return err
		}
	}
	return nil
}

// UpdateDBSettings updates the settings in the database
func (db *Database) UpdateDBSettings(dbsettings DBSettings) (err error) {
	err = db.EntClient.DBSettings.UpdateOneID(1).
		SetIsInitialized(dbsettings.IsInitialized).
		Exec(context.Background())
	if err != nil {
		log.Error("UpdateDBSettings: ", err)
	}
	return err
}

// GetDBSettings returns the settings from the database
func (db *Database) GetDBSettings() (s DBSettings, err error) {
	res, err := db.EntClient.DBSettings.Query().Where(entdbsettings.ID(1)).Only(context.Background())
	if err != nil {
		log.Error("GetDBSettings: ", err)
		return s, err
	}
	s.ID = res.ID
	s.IsInitialized = res.IsInitialized
	return s, nil
}

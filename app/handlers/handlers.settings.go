package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/mitchellh/mapstructure"
)

type KeycloakSettings struct {
	Realm string
	URL   string
}
type ExtendedSettings struct {
	Settings *ent.Settings
	Keycloak KeycloakSettings
}

// Settings -------------------------------------------------------------------

// getSettings godoc
//
//	 	@Summary 		Return settings
//		@Description	Return configuration data of the system
//		@Tags			Core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	database.Settings
//		@Router			/settings/ [get]
func getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	exSettings := ExtendedSettings{
		Settings: settings,
		Keycloak: KeycloakSettings{
			Realm: config.Config.KeycloakRealm,
			URL:   config.Config.KeycloakHostname,
		},
	}
	err = utils.WriteJSON(w, http.StatusOK, &exSettings)
	if err != nil {
		log.Error("getSettings: ", err)
	}
}

type Imagetype string

const (
	ImagetypeLogo    Imagetype = "Logo"
	ImagetypeFavicon Imagetype = "Favicon"
	ImagetypeQrCode  Imagetype = "QRCodeLogoImgUrl"
)

func updateSettingsImg(w http.ResponseWriter, r *http.Request, fileType Imagetype) (path string, err error) {

	// Get file from image field
	file, header, err := r.FormFile(string(fileType))
	if err != nil {
		// Do not return error, as not passing a file is ok
		// Could be improved by differentiating between not passed and invalid file
		err = nil
		return
	}
	defer file.Close()

	// Debugging
	name := strings.Split(header.Filename, ".")
	if len(name) < 2 {
		log.Error("updateSettingsLogo: file name too short", err)
		utils.ErrorJSON(w, errors.New("invalid filename"), http.StatusBadRequest)
		return
	}
	if name[len(name)-1] != "png" {
		log.Error("updateSettingsLogo: wrong file ending: ", name[1])
		utils.ErrorJSON(w, errors.New("file type must be png"), http.StatusBadRequest)
		return
	}

	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		log.Error("updateSettingsLogo: copying file failed", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Save file with name "logo"
	switch fType := fileType; fType {
	case "Logo":
		path = "/img/logo.png"
	case "Favicon":
		path = "/img/favicon.png"
	case "QRCodeLogoImgUrl":
		path = "/img/qrcode.png"
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Error("updateSettingsLogo: couldn't get wd", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	// Save with owner rw, group/other read
	err = os.WriteFile(dir+"/"+path, buf.Bytes(), 0644)
	if err != nil {
		log.Error("updateSettingsLogo: saving failed", err)
		utils.ErrorJSON(w, err, http.StatusBadRequest)
	}
	log.Info("updateSettingsLogo: saved file to ", dir+path)
	return
}

var updateSettingsMutex sync.Mutex

// updateSettings godoc
//
//	 	@Summary 		Update settings
//		@Description	Update configuration data of the system. Requires multipart form. Logo has to be a png and will always be saved under "img/logo.png"
//		@Tags			Core
//		@Accept			json
//		@Produce		json
//	    @Param		    data body database.Settings true "Settings Representation"
//		@Success		200
//		@Security		KeycloakAuth
//		@Router			/settings/ [put]
func updateSettings(w http.ResponseWriter, r *http.Request) {
	updateSettingsMutex.Lock()
	defer updateSettingsMutex.Unlock()
	var err error

	// Read multipart form
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Error("updateSettings: parse multipart:", err)
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}
	mForm := r.MultipartForm
	if mForm == nil {
		log.Error("updateSettings: ", errors.New("form is nil"))
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}

	// Handle normal fields
	settings, err := database.Db.GetSettings()
	if err != nil {
		log.Error("updateSettings: get settings: ", err)
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}
	fields := mForm.Value               // Values are stored in []string
	fieldsClean := make(map[string]any) // Values are stored in string
	for key, value := range fields {
		if key == "MaxOrderAmount" {
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("MaxOrderAmount is not an integer")
				utils.ErrorJSON(w, errors.New("MaxOrderAmount is not an integer"), http.StatusBadRequest)
				return
			}
		} else if key == "OrgaCoversTransactionCosts" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("OrgaCoversTransactionCosts is not a boolean")
				utils.ErrorJSON(w, errors.New("OrgaCoversTransactionCosts is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "WebshopIsClosed" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("WebShopIsClosed is not a boolean")
				utils.ErrorJSON(w, errors.New("WebShopIsClosed is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "QRCodeEnableLogo" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("QRCodeEnableLogo is not a boolean")
				utils.ErrorJSON(w, errors.New("QRCodeEnableLogo is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "MainItem" {
			value, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("MainItem is not an integer")
				utils.ErrorJSON(w, errors.New("MainItem is not an integer"), http.StatusBadRequest)
				return
			}
			fieldsClean[key] = int(value)
		} else if key == "MapCenterLat" || key == "MapCenterLong" {
			if s, err := strconv.ParseFloat(value[0], 64); err == nil {
				fieldsClean[key] = (s)
			} else {
				fieldsClean[key] = 0.1
			}
		} else if key == "UseVendorLicenseIdInShop" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("UseVendorLicenseIdInShop is not a boolean")
				utils.ErrorJSON(w, errors.New("UseVendorLicenseIdInShop is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "UseTipInsteadOfDonation" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("UseTipInsteadOfDonation is not a boolean")
				utils.ErrorJSON(w, errors.New("UseTipInsteadOfDonation is not a boolean"), http.StatusBadRequest)
				return
			}
		} else if key == "ShopLanding" {
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("ShopLanding is not a boolean")
				utils.ErrorJSON(w, errors.New("ShopLanding is not a boolean"), http.StatusBadRequest)
				return
			}
		} else {
			fieldsClean[key] = value[0]
		}
	}

	err = mapstructure.Decode(fieldsClean, &settings)
	if err != nil {
		log.Error("updateSettings: ", err)
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}
	log.Debug("updateSettings: settings are ", settings)
	// Update main item only when explicitly provided by the form.
	// If not provided, leave MainItem nil so UpdateSettings won't try
	// to change the FK (avoids foreign-key errors in tests where item 1
	// may not exist).
	if mainItemID, ok := fieldsClean["MainItem"].(int); ok {
		var mainItem ent.Item
		mainItem.ID = mainItemID
		settings.Edges.MainItem = &mainItem
	} else {
		settings.Edges.MainItem = nil
		log.Debug("updateSettings: MainItem not set; leaving unchanged")
	}
	// update the logo
	logoPath, err := updateSettingsImg(w, r, ImagetypeLogo)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if logoPath != "" {
		settings.Logo = logoPath
		log.Info("updateSettings: settings.Logo is ", settings.Logo)
	}

	// update the favicon
	faviconPath, err := updateSettingsImg(w, r, ImagetypeFavicon)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if faviconPath != "" {
		settings.Favicon = faviconPath
		log.Info("updateSettings: settings.Favicon is ", settings.Favicon)
	}

	// update the qrcode logo
	qrcodePath, err := updateSettingsImg(w, r, ImagetypeQrCode)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	if qrcodePath != "" {
		settings.QRCodeLogoImgUrl = qrcodePath
		log.Info("updateSettings: settings.QRCodeLogoImgUrl is ", settings.QRCodeLogoImgUrl)
	}

	// Save settings to database
	err = database.Db.UpdateSettings(settings)
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	err = utils.WriteJSON(w, http.StatusOK, settings)
	if err != nil {
		log.Error("updateSettings: ", err)
	}
	log.Info("updateSettings: settings updated")
}

// UpdateCss godoc
//
//	@Summary 		Update CSS
//	@Description	Gets a css as a string and saves it to the disk
//	@Tags			Settings
//	@Accept			txt
//	@Produce		txt
//	@Success		200
//	@Router			/settings/css [put]

func updateCSS(w http.ResponseWriter, r *http.Request) {

	// Read data from request
	body, err := io.ReadAll(r.Body)

	if err != nil {
		log.Error("Reading body failed for css: ", err)
		err = errors.New("failed to update css")

		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}

	// Save file with name "style.css"
	path := "/public/style.css"
	dir, err := os.Getwd()
	if err != nil {
		log.Error("updateCSS: couldn't get wd", err)
		err = errors.New("failed to update css")

		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	// Public CSS should be readable by the webserver
	err = os.WriteFile(dir+path, body, 0644)
	if err != nil {
		log.Error("updateCSS: saving failed", err)
		err = errors.New("failed to update css")
		utils.ErrorJSON(w, err, http.StatusBadRequest)
	}
	log.Info("updateCSS: success")
}

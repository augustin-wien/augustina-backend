package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/integrations"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/augustin-wien/augustina-backend/wordpress"
	"github.com/mitchellh/mapstructure"
)

type KeycloakSettings struct {
	Realm string
	URL   string
}
type ExtendedSettings struct {
	Settings *ent.Settings
	Keycloak KeycloakSettings
	// SecretsSet tells which of the secretSettings have a value, since the values are never sent
	SecretsSet map[string]bool `json:",omitempty"`
}

// secretSettings can be overwritten in the backoffice but are never sent back to it. An empty
// value on update keeps the stored one.
var secretSettings = []string{"WordPressInviteAPIKey", "NextcloudAppPassword", "OdooWebhookToken"}

func isSecretSetting(key string) bool {
	for _, k := range secretSettings {
		if k == key {
			return true
		}
	}
	return false
}

// withoutSecrets returns a copy of the settings with the secrets blanked and which of them are set
func withoutSecrets(s *ent.Settings) (*ent.Settings, map[string]bool) {
	c := *s
	set := map[string]bool{
		"WordPressInviteAPIKey": c.WordPressInviteAPIKey != "",
		"NextcloudAppPassword":  c.NextcloudAppPassword != "",
		"OdooWebhookToken":      c.OdooWebhookToken != "",
	}
	c.WordPressInviteAPIKey = ""
	c.NextcloudAppPassword = ""
	c.OdooWebhookToken = ""
	return &c, set
}

// PublicSettings is the subset of the settings the shop is allowed to see.
//
// The shop needs these before anyone logs in, so /api/settings/ is unauthenticated — which means
// every field listed here is world readable. That is why this is an explicit allowlist instead of
// a filter over ent.Settings: a setting added to the schema stays private until someone puts it
// here on purpose, and credentials (WordPressInviteAPIKey and friends) simply never appear.
// The backoffice gets the complete entity from /api/settings/admin/ instead.
type PublicSettings struct {
	ID                         int
	AGBUrl                     string
	Color                      string
	FontColor                  string
	Logo                       string
	Favicon                    string
	MaxOrderAmount             int
	OrgaCoversTransactionCosts bool
	WebshopIsClosed            bool
	VendorNotFoundHelpUrl      string
	MaintainanceModeHelpUrl    string
	VendorEmailPostfix         string
	NewspaperName              string
	QRCodeUrl                  string
	QRCodeLogoImgUrl           string
	QRCodeSettings             string
	QRCodeEnableLogo           bool
	MapCenterLat               float64
	MapCenterLong              float64
	UseVendorLicenseIdInShop   bool
	UseTipInsteadOfDonation    bool
	ShopLanding                bool
	DigitalItemsUrl            string
	AbonementUrl               string
	AbonementEnabled           bool
	POSEnabled                 bool
	OdooEnabled                bool
	PrivacyPolicyUrl           string
	MatomoUrl                  string
	MatomoSiteId               string
	DonationTrackingEnabled    bool
	AGBRequired                bool
	DonationPageEnabled        bool
	MatomoTrackCart            bool
	FeedbackEnabled            bool
	NewsletterEnabled          bool
	IDScanEnabled              bool
	NextcloudEnabled           bool
	// NextcloudConfigured reports whether the connection is filled in, without giving it away
	NextcloudConfigured bool
	Edges               ent.SettingsEdges
}

type ExtendedPublicSettings struct {
	Settings PublicSettings
	Keycloak KeycloakSettings
}

func toPublicSettings(s *ent.Settings) PublicSettings {
	return PublicSettings{
		ID:                         s.ID,
		AGBUrl:                     s.AGBUrl,
		Color:                      s.Color,
		FontColor:                  s.FontColor,
		Logo:                       s.Logo,
		Favicon:                    s.Favicon,
		MaxOrderAmount:             s.MaxOrderAmount,
		OrgaCoversTransactionCosts: s.OrgaCoversTransactionCosts,
		WebshopIsClosed:            s.WebshopIsClosed,
		VendorNotFoundHelpUrl:      s.VendorNotFoundHelpUrl,
		MaintainanceModeHelpUrl:    s.MaintainanceModeHelpUrl,
		VendorEmailPostfix:         s.VendorEmailPostfix,
		NewspaperName:              s.NewspaperName,
		QRCodeUrl:                  s.QRCodeUrl,
		QRCodeLogoImgUrl:           s.QRCodeLogoImgUrl,
		QRCodeSettings:             s.QRCodeSettings,
		QRCodeEnableLogo:           s.QRCodeEnableLogo,
		MapCenterLat:               s.MapCenterLat,
		MapCenterLong:              s.MapCenterLong,
		UseVendorLicenseIdInShop:   s.UseVendorLicenseIdInShop,
		UseTipInsteadOfDonation:    s.UseTipInsteadOfDonation,
		ShopLanding:                s.ShopLanding,
		DigitalItemsUrl:            s.DigitalItemsUrl,
		AbonementUrl:               s.AbonementUrl,
		AbonementEnabled:           s.AbonementEnabled,
		POSEnabled:                 s.POSEnabled,
		OdooEnabled:                config.Config.OdooWebhookURL != "",
		PrivacyPolicyUrl:           s.PrivacyPolicyUrl,
		MatomoUrl:                  s.MatomoUrl,
		MatomoSiteId:               s.MatomoSiteId,
		DonationTrackingEnabled:    s.DonationTrackingEnabled,
		AGBRequired:                s.AGBRequired,
		DonationPageEnabled:        s.DonationPageEnabled,
		MatomoTrackCart:            s.MatomoTrackCart,
		FeedbackEnabled:            s.FeedbackEnabled,
		NewsletterEnabled:          s.NewsletterEnabled,
		IDScanEnabled:              s.IDScanEnabled,
		NextcloudEnabled:           s.NextcloudEnabled,
		NextcloudConfigured:        integrations.NextcloudFromSettings(s).Configured(),
		Edges:                      s.Edges,
	}
}

func keycloakSettings() KeycloakSettings {
	return KeycloakSettings{
		Realm: config.Config.KeycloakRealm,
		URL:   config.Config.KeycloakHostname,
	}
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
	exSettings := ExtendedPublicSettings{
		Settings: toPublicSettings(settings),
		Keycloak: keycloakSettings(),
	}
	err = utils.WriteJSON(w, http.StatusOK, &exSettings)
	if err != nil {
		log.Error("getSettings: ", err)
	}
}

// getSettingsAdmin godoc
//
//	 	@Summary 		Return all settings
//		@Description	Return the complete settings including credentials, for the backoffice
//		@Tags			Core
//		@Accept			json
//		@Produce		json
//		@Success		200
//		@Security		KeycloakAuth
//		@Router			/settings/admin/ [get]
func getSettingsAdmin(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	safe, secretsSet := withoutSecrets(settings)
	exSettings := ExtendedSettings{
		Settings:   safe,
		Keycloak:   keycloakSettings(),
		SecretsSet: secretsSet,
	}
	err = utils.WriteJSON(w, http.StatusOK, &exSettings)
	if err != nil {
		log.Error("getSettingsAdmin: ", err)
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
		if isSecretSetting(key) && value[0] == "" {
			continue
		}
		switch key {
		case "MaxOrderAmount":
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("MaxOrderAmount is not an integer")
				utils.ErrorJSON(w, errors.New("MaxOrderAmount is not an integer"), http.StatusBadRequest)
				return
			}
		case "OrgaCoversTransactionCosts":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("OrgaCoversTransactionCosts is not a boolean")
				utils.ErrorJSON(w, errors.New("OrgaCoversTransactionCosts is not a boolean"), http.StatusBadRequest)
				return
			}
		case "WebshopIsClosed":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("WebShopIsClosed is not a boolean")
				utils.ErrorJSON(w, errors.New("WebShopIsClosed is not a boolean"), http.StatusBadRequest)
				return
			}
		case "QRCodeEnableLogo":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("QRCodeEnableLogo is not a boolean")
				utils.ErrorJSON(w, errors.New("QRCodeEnableLogo is not a boolean"), http.StatusBadRequest)
				return
			}
		case "MainItem":
			value, err := strconv.Atoi(value[0])
			if err != nil {
				log.Error("MainItem is not an integer")
				utils.ErrorJSON(w, errors.New("MainItem is not an integer"), http.StatusBadRequest)
				return
			}
			fieldsClean[key] = int(value)
		case "MapCenterLat", "MapCenterLong":
			if s, err := strconv.ParseFloat(value[0], 64); err == nil {
				fieldsClean[key] = (s)
			} else {
				fieldsClean[key] = 0.1
			}
		case "UseVendorLicenseIdInShop":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("UseVendorLicenseIdInShop is not a boolean")
				utils.ErrorJSON(w, errors.New("UseVendorLicenseIdInShop is not a boolean"), http.StatusBadRequest)
				return
			}
		case "UseTipInsteadOfDonation":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("UseTipInsteadOfDonation is not a boolean")
				utils.ErrorJSON(w, errors.New("UseTipInsteadOfDonation is not a boolean"), http.StatusBadRequest)
				return
			}
		case "ShopLanding":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("ShopLanding is not a boolean")
				utils.ErrorJSON(w, errors.New("ShopLanding is not a boolean"), http.StatusBadRequest)
				return
			}
		case "AbonementEnabled":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("AbonementEnabled is not a boolean")
				utils.ErrorJSON(w, errors.New("AbonementEnabled is not a boolean"), http.StatusBadRequest)
				return
			}
		case "POSEnabled":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("POSEnabled is not a boolean")
				utils.ErrorJSON(w, errors.New("POSEnabled is not a boolean"), http.StatusBadRequest)
				return
			}
		case "DonationTrackingEnabled":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error("DonationTrackingEnabled is not a boolean")
				utils.ErrorJSON(w, errors.New("DonationTrackingEnabled is not a boolean"), http.StatusBadRequest)
				return
			}
		case "AGBRequired", "DonationPageEnabled", "MatomoTrackCart", "FeedbackEnabled", "NewsletterEnabled", "IDScanEnabled", "NextcloudEnabled":
			fieldsClean[key], err = strconv.ParseBool(value[0])
			if err != nil {
				log.Error(key + " is not a boolean")
				utils.ErrorJSON(w, errors.New(key+" is not a boolean"), http.StatusBadRequest)
				return
			}
		case "WordPressInviteTTL":
			fieldsClean[key], err = strconv.Atoi(value[0])
			if err != nil {
				log.Error("WordPressInviteTTL is not an integer")
				utils.ErrorJSON(w, errors.New("WordPressInviteTTL is not an integer"), http.StatusBadRequest)
				return
			}
		default:
			fieldsClean[key] = value[0]
		}
	}

	err = mapstructure.Decode(fieldsClean, &settings)
	if err != nil {
		log.Error("updateSettings: ", err)
		utils.ErrorJSON(w, errors.New("invalid form"), http.StatusBadRequest)
		return
	}
	if _, ok := fieldsClean["VendorEmailPostfix"]; ok {
		if _, err := normalizeVendorEmailPostfix(settings.VendorEmailPostfix); err != nil {
			log.Warn("updateSettings: ", err)
			utils.ErrorJSON(w, err, http.StatusBadRequest)
			return
		}
	}
	if strings.ContainsAny(settings.MailSenderName, "\r\n") {
		utils.ErrorJSON(w, errors.New("mail sender name contains invalid characters"), http.StatusBadRequest)
		return
	}
	settings.FeedbackEmail = strings.TrimSpace(settings.FeedbackEmail)
	if settings.FeedbackEmail != "" {
		if _, err := mail.ParseAddress(settings.FeedbackEmail); err != nil || strings.ContainsAny(settings.FeedbackEmail, "\r\n") {
			utils.ErrorJSON(w, errors.New("feedback email is invalid"), http.StatusBadRequest)
			return
		}
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

// WordPressInviteTestRequest holds the invite settings to test, so admins can
// check them before saving. An empty value falls back to the saved setting.
type WordPressInviteTestRequest struct {
	WordPressInviteURL    string
	WordPressInviteAPIKey string
}

// WordPressInviteTestResponse tells whether WordPress handed out a login link
type WordPressInviteTestResponse struct {
	Success bool
	Message string
	Link    string
}

// wordPressInviteTestEmail is the address the test link is created for; it
// belongs to nobody, so the link cannot log anyone in
const wordPressInviteTestEmail = "verbindungstest@example.com"

// testWordPressInvite godoc
//
//	@Summary 		Test the WordPress one-time login connection
//	@Description	Asks WordPress for a short-lived login link for a dummy address and reports whether that worked
//	@Tags			Settings
//	@Accept			json
//	@Produce		json
//	@Param			data body WordPressInviteTestRequest true "Invite settings to test"
//	@Success		200 {object} WordPressInviteTestResponse
//	@Security		KeycloakAuth
//	@Router			/settings/wordpress-invite/test/ [post]
func testWordPressInvite(w http.ResponseWriter, r *http.Request) {
	var req WordPressInviteTestRequest
	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.ErrorJSON(w, err, http.StatusBadRequest)
		return
	}
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}
	if req.WordPressInviteURL == "" {
		req.WordPressInviteURL = settings.WordPressInviteURL
	}
	if req.WordPressInviteAPIKey == "" {
		req.WordPressInviteAPIKey = settings.WordPressInviteAPIKey
	}

	var res WordPressInviteTestResponse
	switch {
	case req.WordPressInviteURL == "":
		res.Message = "Keine API-URL eingetragen"
	case req.WordPressInviteAPIKey == "":
		res.Message = "Kein API-Schlüssel eingetragen"
	default:
		link, err := wordpress.CreateInvite(req.WordPressInviteURL, req.WordPressInviteAPIKey, wordpress.Invite{Email: wordPressInviteTestEmail, TTL: 60})
		if err != nil {
			log.Info("testWordPressInvite: ", err)
			res.Message = err.Error()
			var statusErr *wordpress.StatusError
			if errors.As(err, &statusErr) {
				res.Message = statusErr.Hint()
			}
		} else {
			res.Success = true
			res.Message = "WordPress hat einen Login-Link erstellt"
			res.Link = link
		}
	}
	if err = utils.WriteJSON(w, http.StatusOK, res); err != nil {
		log.Error("testWordPressInvite: ", err)
	}
}

// NextcloudTestResponse tells whether the backend can create vendor folders
type NextcloudTestResponse struct {
	Success bool
	Message string
}

// testNextcloud godoc
//
//	@Summary 		Test the Nextcloud connection
//	@Description	Logs in to Nextcloud with the saved settings and checks that vendor folders can be created in the configured folder
//	@Tags			Settings
//	@Produce		json
//	@Success		200 {object} NextcloudTestResponse
//	@Security		KeycloakAuth
//	@Router			/settings/nextcloud/test/ [post]
func testNextcloud(w http.ResponseWriter, r *http.Request) {
	// Only the saved settings are tested: a server address taken from the request would let the
	// endpoint send requests to any host.
	settings, err := database.Db.GetSettings()
	if err != nil {
		utils.ErrorJSON(w, err, http.StatusInternalServerError)
		return
	}

	var res NextcloudTestResponse
	res.Success, res.Message = integrations.NextcloudFromSettings(settings).Test()
	if !res.Success {
		log.Info("testNextcloud: ", res.Message)
	}
	if err = utils.WriteJSON(w, http.StatusOK, res); err != nil {
		log.Error("testNextcloud: ", err)
	}
}

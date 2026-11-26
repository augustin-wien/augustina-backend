package handlers

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// putSettings sends the fields (and optional files) as multipart form to the settings endpoint
func putSettings(t *testing.T, fields map[string]string, files map[string]string, status int) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		require.NoError(t, writer.WriteField(k, v))
	}
	for field, filename := range files {
		f, err := writer.CreateFormFile(field, filename)
		require.NoError(t, err)
		_, err = f.Write([]byte("png content"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, writer.FormDataContentType(), status, adminUserToken)
}

func TestUpdateSettingsValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	for _, key := range []string{
		"OrgaCoversTransactionCosts", "WebshopIsClosed", "QRCodeEnableLogo", "UseVendorLicenseIdInShop",
		"UseTipInsteadOfDonation", "ShopLanding", "AbonementEnabled", "POSEnabled", "DonationTrackingEnabled",
		"AGBRequired", "DonationPageEnabled", "MatomoTrackCart", "FeedbackEnabled", "NewsletterEnabled",
		"IDScanEnabled", "NextcloudEnabled",
	} {
		t.Run(key, func(t *testing.T) {
			putSettings(t, map[string]string{key: "maybe"}, nil, 400)
		})
	}
	for _, key := range []string{"MaxOrderAmount", "MainItem", "WordPressInviteTTL"} {
		t.Run(key, func(t *testing.T) {
			putSettings(t, map[string]string{key: "many"}, nil, 400)
		})
	}

	putSettings(t, map[string]string{"MailSenderName": "Augustin\r\nBcc: x@example.com"}, nil, 400)
	putSettings(t, map[string]string{"FeedbackEmail": "not an address"}, nil, 400)
	putSettings(t, map[string]string{"AccountingReportEmail": "a@example.com, nope"}, nil, 400)
	putSettings(t, map[string]string{"VendorEmailPostfix": "not a domain at all"}, nil, 400)
	putSettings(t, nil, map[string]string{"Logo": "logo.jpg"}, 400)
	putSettings(t, nil, map[string]string{"Favicon": "favicon"}, 400)
	utils.TestRequestStrWithAuth(t, r, "PUT", "/api/settings/", "{}", 400, adminUserToken)

	// Valid values are stored
	putSettings(t, map[string]string{
		"WebshopIsClosed":         "true",
		"POSEnabled":              "true",
		"MapCenterLat":            "48.2",
		"MapCenterLong":           "not a number",
		"FeedbackEmail":           "  feedback@example.com ",
		"AccountingReportEmail":   " a@example.com,b@example.com ",
		"MailSenderName":          "Augustin",
		"WordPressInviteTTL":      "7",
		"DonationTrackingEnabled": "true",
	}, map[string]string{"Favicon": "favicon.png", "QRCodeLogoImgUrl": "qr.png"}, 200)

	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	require.True(t, settings.WebshopIsClosed)
	require.True(t, settings.POSEnabled)
	require.True(t, settings.DonationTrackingEnabled)
	require.Equal(t, 48.2, settings.MapCenterLat)
	require.Equal(t, 0.1, settings.MapCenterLong)
	require.Equal(t, "feedback@example.com", settings.FeedbackEmail)
	require.Equal(t, "a@example.com, b@example.com", settings.AccountingReportEmail)
	require.Regexp(t, `^/img/favicon\.png\?v=[0-9a-f]{8}$`, settings.Favicon)
	require.Regexp(t, `^/img/qrcode\.png\?v=[0-9a-f]{8}$`, settings.QRCodeLogoImgUrl)
	require.Equal(t, 7, settings.WordPressInviteTTL)
}

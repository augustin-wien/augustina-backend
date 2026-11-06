package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestOpenVendorNextcloudFolder checks that the backoffice gets a link to the vendor's
// document folder once the integration is enabled and configured
func TestOpenVendorNextcloudFolder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	var created []string
	nextcloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			created = append(created, r.URL.EscapedPath())
			w.WriteHeader(http.StatusCreated)
		case "PROPFIND":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns"><d:response>` +
				`<d:propstat><d:prop><oc:fileid>42</oc:fileid></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>` +
				`</d:response></d:multistatus>`))
		}
	}))
	defer nextcloud.Close()

	_, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Cloud",
		LastName:  "Folder",
		Email:     "cloud@vendor.com",
		LicenseID: null.StringFrom("Cloud 1"),
	})
	require.NoError(t, err)

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/Cloud%201/nextcloud-folder/", nil, 403, adminUserToken)
	putSettingsFields(t, map[string]string{"NextcloudEnabled": "true"}, 200)

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/Cloud%201/nextcloud-folder/", nil, 503, adminUserToken)

	putSettingsFields(t, map[string]string{
		"NextcloudURL":         nextcloud.URL,
		"NextcloudPublicURL":   "https://cloud.example.org",
		"NextcloudUsername":    "augustin",
		"NextcloudAppPassword": "secret",
		"NextcloudVendorRoot":  "Vendors",
	}, 200)

	// The public settings say whether the connection is set up, but never what it is
	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	require.Contains(t, res.Body.String(), `"NextcloudConfigured":true`)
	require.NotContains(t, res.Body.String(), "secret")

	res = utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/Cloud%201/nextcloud-folder/", nil, 200, adminUserToken)
	var folder VendorNextcloudFolder
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &folder))
	require.Equal(t, "https://cloud.example.org/index.php/f/42", folder.URL)
	require.Equal(t, []string{"/remote.php/dav/files/augustin/Vendors/Cloud%201/"}, created)

	// The connection test uses the saved settings and ignores any address in the request
	before := len(created)
	res = utils.TestRequestWithAuth(t, r, "POST", "/api/settings/nextcloud/test/",
		map[string]string{"NextcloudURL": "http://169.254.169.254"}, 200, adminUserToken)
	var test NextcloudTestResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &test))
	require.Contains(t, test.Message, "augustin darf im Ordner „Vendors“ keine Ordner anlegen")
	require.Len(t, created, before)

	utils.TestRequestWithAuth(t, r, "POST", "/api/vendors/unknown/nextcloud-folder/", nil, 404, adminUserToken)
	utils.TestRequest(t, r, "POST", "/api/vendors/Cloud%201/nextcloud-folder/", nil, 401)
}

// TestSettingsSecretsAreWriteOnly checks that the backoffice can overwrite secrets but never
// read them back, and that saving the form with the empty secret fields keeps them
func TestSettingsSecretsAreWriteOnly(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	putSettingsFields(t, map[string]string{
		"WordPressInviteAPIKey": "wp-secret",
		"NextcloudAppPassword":  "nc-secret",
		"OdooWebhookToken":      "odoo-secret",
		"OdooDBFilter":          "augustin_db",
	}, 200)

	readAdmin := func() (ExtendedSettings, string) {
		res := utils.TestRequestWithAuth(t, r, "GET", "/api/settings/admin/", nil, 200, adminUserToken)
		var ex ExtendedSettings
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &ex))
		return ex, res.Body.String()
	}
	ex, body := readAdmin()
	for _, secret := range []string{"wp-secret", "nc-secret", "odoo-secret"} {
		require.NotContains(t, body, secret)
	}
	require.Equal(t, map[string]bool{"WordPressInviteAPIKey": true, "NextcloudAppPassword": true, "OdooWebhookToken": true}, ex.SecretsSet)
	require.Equal(t, "augustin_db", ex.Settings.OdooDBFilter)

	// The backoffice posts the blank fields back
	putSettingsFields(t, map[string]string{"WordPressInviteAPIKey": "", "NextcloudAppPassword": "", "OdooWebhookToken": ""}, 200)
	stored, err := database.Db.GetSettings()
	require.NoError(t, err)
	require.Equal(t, "wp-secret", stored.WordPressInviteAPIKey)
	require.Equal(t, "nc-secret", stored.NextcloudAppPassword)
	require.Equal(t, "odoo-secret", stored.OdooWebhookToken)

	putSettingsFields(t, map[string]string{"OdooWebhookToken": "new-token"}, 200)
	stored, err = database.Db.GetSettings()
	require.NoError(t, err)
	require.Equal(t, "new-token", stored.OdooWebhookToken)
}

// The secrets can't be read by the backoffice, so they must not reach another server either:
// changing the URL an integration sends its secret to requires entering the secret again
func TestSettingsSecretsStayWithTheirURL(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	var requests int
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer other.Close()

	putSettingsFields(t, map[string]string{
		"WordPressInviteURL":    "https://wordpress.example.org/wp-json/invite",
		"WordPressInviteAPIKey": "wp-secret",
		"NextcloudURL":          "https://cloud.example.org",
		"NextcloudAppPassword":  "nc-secret",
		"OdooWebhookURL":        "https://odoo.example.org/webhook",
		"OdooWebhookToken":      "odoo-secret",
	}, 200)

	// The connection test only uses the saved key with the saved URL
	res := utils.TestRequestWithAuth(t, r, "POST", "/api/settings/wordpress-invite/test/",
		WordPressInviteTestRequest{WordPressInviteURL: other.URL}, 200, adminUserToken)
	var test WordPressInviteTestResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &test))
	require.False(t, test.Success)
	require.Zero(t, requests, "the saved key must not be sent to another URL")

	for _, fields := range []map[string]string{
		{"WordPressInviteURL": other.URL, "WordPressInviteAPIKey": ""},
		{"NextcloudURL": other.URL, "NextcloudAppPassword": ""},
		{"OdooWebhookURL": other.URL},
	} {
		putSettingsFields(t, fields, 400)
	}
	stored, err := database.Db.GetSettings()
	require.NoError(t, err)
	require.Equal(t, "https://wordpress.example.org/wp-json/invite", stored.WordPressInviteURL)
	require.Equal(t, "https://cloud.example.org", stored.NextcloudURL)
	require.Equal(t, "https://odoo.example.org/webhook", stored.OdooWebhookURL)

	// Saving the form unchanged, entering the secret with the new URL and switching an integration
	// off all keep working
	putSettingsFields(t, map[string]string{"NextcloudURL": "https://cloud.example.org/", "NextcloudAppPassword": ""}, 200)
	putSettingsFields(t, map[string]string{"OdooWebhookURL": other.URL, "OdooWebhookToken": "new-token"}, 200)
	putSettingsFields(t, map[string]string{"WordPressInviteURL": "", "WordPressInviteAPIKey": ""}, 200)
	putSettingsFields(t, map[string]string{"OdooWebhookURL": ""}, 200)
}

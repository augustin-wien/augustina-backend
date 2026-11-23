package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/stretchr/testify/require"
)

// TestSettings tests GET and PUT operations on settings
func TestSettings(t *testing.T) {

	itemID := CreateTestItem(t, "Test main item", 314, "", "")

	// Update (multipart form!)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	writer.WriteField("MaxOrderAmount", strconv.Itoa(10))
	writer.WriteField("MainItem", itemID)
	image, _ := writer.CreateFormFile("Logo", "test.png")
	image.Write([]byte(`i am the content of a jpg file :D`))
	writer.Close()
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, writer.FormDataContentType(), 200, adminUserToken)

	// Read
	var exSettings ExtendedSettings
	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	err := json.Unmarshal(res.Body.Bytes(), &exSettings)
	utils.CheckError(t, err)
	require.Equal(t, "/img/logo.png", exSettings.Settings.Logo)
	require.Equal(t, 10, exSettings.Settings.MaxOrderAmount)

	// Check item join
	require.Equal(t, "Test main item", exSettings.Settings.Edges.MainItem.Name)
	// Price is a float in ent Item
	require.Equal(t, float64(314), exSettings.Settings.Edges.MainItem.Price)

	// Check file
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	file, err := os.ReadFile(dir + "/" + exSettings.Settings.Logo)
	utils.CheckError(t, err)
	require.Equal(t, `i am the content of a jpg file :D`, string(file))

}

// TestWordPressInviteConnectionTest checks the backoffice button that tests
// the WordPress one-time login settings
func TestWordPressInviteConnectionTest(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	err := database.Db.InitEmptyTestDb()
	require.NoError(t, err)

	wpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gone" {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer right-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":"shop_unauthorized","message":"Ungültiger oder fehlender API-Schlüssel.","data":{"status":401}}`))
			return
		}
		_, _ = w.Write([]byte(`{"magic_link":"https://wordpress.test/app/willkommen?token=abc"}`))
	}))
	defer wpServer.Close()

	test := func(body WordPressInviteTestRequest) (res WordPressInviteTestResponse) {
		resp := utils.TestRequestWithAuth(t, r, "POST", "/api/settings/wordpress-invite/test/", body, 200, adminUserToken)
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		return res
	}

	res := test(WordPressInviteTestRequest{WordPressInviteURL: wpServer.URL, WordPressInviteAPIKey: "right-key"})
	require.True(t, res.Success)
	require.Equal(t, "https://wordpress.test/app/willkommen?token=abc", res.Link)

	res = test(WordPressInviteTestRequest{WordPressInviteURL: wpServer.URL, WordPressInviteAPIKey: "wrong-key"})
	require.False(t, res.Success)
	require.Contains(t, res.Message, "API-Schlüssel")

	// Something other than the plugin answers, e.g. a removed site
	res = test(WordPressInviteTestRequest{WordPressInviteURL: wpServer.URL + "/gone", WordPressInviteAPIKey: "right-key"})
	require.False(t, res.Success)
	require.Equal(t, "Unter "+wpServer.URL+"/gone gibt es keinen Einladungs-Endpunkt (Status 410). Bitte prüfen, ob die API-URL stimmt (…/wp-json/augustin/v1/shop/create-invite) und das augustin-ki-Plugin aktiv ist.", res.Message)

	res = test(WordPressInviteTestRequest{})
	require.False(t, res.Success)
	require.Equal(t, "Keine API-URL eingetragen", res.Message)

	// Only admins may test
	utils.TestRequest(t, r, "POST", "/api/settings/wordpress-invite/test/", WordPressInviteTestRequest{}, 401)
}

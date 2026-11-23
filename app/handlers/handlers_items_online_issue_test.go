package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestGetItemPDF(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	pdfPath := filepath.Join(t.TempDir(), "ausgabe.pdf")
	require.NoError(t, os.WriteFile(pdfPath, []byte("%PDF-1.4 item"), 0o600))
	pdfID, err := database.Db.CreatePDF(database.PDF{Path: pdfPath, Timestamp: time.Now()})
	require.NoError(t, err)
	withPDF, err := database.Db.CreateItem(database.Item{Name: "Item With PDF", Description: "Test item", Price: 300, Type: "issue", IsPDFItem: true, PDF: null.IntFrom(pdfID)})
	require.NoError(t, err)
	withoutPDF, err := database.Db.CreateItem(database.Item{Name: "Item Without PDF", Description: "Test item", Price: 300, Type: "issue"})
	require.NoError(t, err)

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/items/"+strconv.Itoa(withPDF)+"/pdf/", nil, 200, adminUserToken)
	require.Equal(t, "%PDF-1.4 item", res.Body.String())
	require.Equal(t, "application/pdf", res.Header().Get("Content-Type"))
	require.Contains(t, res.Header().Get("Content-Disposition"), "ausgabe.pdf")

	utils.TestRequestWithAuth(t, r, "GET", "/api/items/"+strconv.Itoa(withoutPDF)+"/pdf/", nil, 404, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/items/999999/pdf/", nil, 404, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/items/abc/pdf/", nil, 400, adminUserToken)
}

func TestNotifyAbonementsWithPDFLicense(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	origFrontend := config.Config.FrontendURL
	origSend := mailer.Send
	t.Cleanup(func() {
		config.Config.FrontendURL = origFrontend
		mailer.Send = origSend
	})
	config.Config.FrontendURL = "https://frontend.test"

	var mu sync.Mutex
	var sent []*mailer.EmailRequest
	mailer.Send = func(m *mailer.EmailRequest) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, m)
		return true, nil
	}

	pdfID, err := database.Db.CreatePDF(database.PDF{Path: "/tmp/notify.pdf", Timestamp: time.Now()})
	require.NoError(t, err)
	licenseID, err := database.Db.CreateItem(database.Item{
		Name: "Notify PDF License", Description: "Test item", Price: 100, Type: "license_item",
		IsLicenseItem: true, IsPDFItem: true, PDF: null.IntFrom(pdfID), LicenseGroup: null.StringFrom("9702"),
	})
	require.NoError(t, err)
	issueID, err := database.Db.CreateItem(database.Item{
		Name: "Notify PDF Issue", Description: "Test item", Price: 300, Type: "online_issue",
		LicenseGroup: null.StringFrom("9702"), LicenseItem: null.IntFrom(int64(licenseID)), Image: "https://cdn.test/cover.jpg",
	})
	require.NoError(t, err)

	customer, err := database.Db.CreateCustomer(&database.Customer{Email: "pdf.abonnent@example.com"})
	require.NoError(t, err)
	_, err = database.Db.CreateAbonement(&database.Abonement{
		CustomerID: customer.ID, FromDate: time.Now().AddDate(0, -1, 0), ToDate: time.Now().AddDate(0, 1, 0), Status: "active",
	})
	require.NoError(t, err)

	groupPath := "/" + keycloak.KeycloakClient.CustomerGroup + "/" + keycloak.KeycloakClient.NewspaperGroup + "/9702"
	t.Cleanup(func() { _ = keycloak.KeycloakClient.DeleteSubGroupByPath(groupPath) })

	res := utils.TestRequestWithAuth(t, r, "POST", "/api/items/"+strconv.Itoa(issueID)+"/notify-abonements/", nil, 200, adminUserToken)
	var response NotifyAbonementsResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	require.Equal(t, 1, response.Recipients)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(sent) == 1
	}, 5*time.Second, 50*time.Millisecond)
	mu.Lock()
	body := sent[0].Body()
	mu.Unlock()
	// The PDF of the license item is mailed as a personal download link
	require.Contains(t, body, "https://frontend.test/pdf/")
	require.Contains(t, body, "https://cdn.test/cover.jpg")
}

func TestNotifyAbonementsValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	noGroup, err := database.Db.CreateItem(database.Item{Name: "Issue Without Group", Description: "Test item", Price: 300, Type: "online_issue"})
	require.NoError(t, err)
	archived, err := database.Db.CreateItem(database.Item{Name: "Archived Issue", Description: "Test item", Price: 300, Type: "online_issue", LicenseGroup: null.StringFrom("9703")})
	require.NoError(t, err)
	require.NoError(t, database.Db.DeleteItem(archived))

	utils.TestRequestWithAuth(t, r, "POST", "/api/items/abc/notify-abonements/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/items/999999/notify-abonements/", nil, 404, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/items/"+strconv.Itoa(noGroup)+"/notify-abonements/", nil, 400, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", "/api/items/"+strconv.Itoa(archived)+"/notify-abonements/", nil, 404, adminUserToken)
}

func TestCheckWordPressIssueErrors(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/issue/old"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"rest_no_route","message":"No route"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<html>down</html>`))
		}
	}))
	defer wp.Close()

	setInvite := func(url string) {
		settings, err := database.Db.GetSettings()
		require.NoError(t, err)
		settings.WordPressInviteURL, settings.WordPressInviteAPIKey = url, "key"
		require.NoError(t, database.Db.UpdateSettings(settings))
	}
	check := func(licenseGroup string) WordPressIssueCheck {
		res := utils.TestRequestWithAuth(t, r, "GET", "/api/items/wordpress-issue/?licenseGroup="+licenseGroup, nil, 200, adminUserToken)
		var out WordPressIssueCheck
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
		return out
	}

	setInvite(wp.URL + "/wp-json/augustin/v1/shop/create-invite")
	res := check("")
	require.True(t, res.Configured)
	require.False(t, res.OK)
	require.Contains(t, check("old").Message, "zu alt")
	require.False(t, check("651").OK)

	setInvite("http://127.0.0.1:1/wp-json/augustin/v1/shop/create-invite")
	require.Contains(t, check("651").Message, "nicht erreichbar")
}

func TestResolveIssueImageURL(t *testing.T) {
	orig := config.Config.FrontendURL
	t.Cleanup(func() { config.Config.FrontendURL = orig })

	config.Config.FrontendURL = "https://shop.test/"
	require.Equal(t, "", resolveIssueImageURL(nil, "  "))
	require.Equal(t, "https://cdn.test/a.jpg", resolveIssueImageURL(nil, "https://cdn.test/a.jpg"))
	require.Equal(t, "https://shop.test/img/a.jpg", resolveIssueImageURL(nil, "/img/a.jpg"))

	config.Config.FrontendURL = ""
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "backend.test"
	require.Equal(t, "http://backend.test/img/a.jpg", resolveIssueImageURL(req, "img/a.jpg"))
	require.Equal(t, "img/a.jpg", resolveIssueImageURL(nil, "img/a.jpg"))
}

package database

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func setWordPressInvite(t *testing.T, inviteURL, apiKey string) {
	settings, err := Db.GetSettings()
	require.NoError(t, err)
	settings.WordPressInviteURL = inviteURL
	settings.WordPressInviteAPIKey = apiKey
	require.NoError(t, Db.UpdateSettings(settings))
}

// enableItem publishes an item; items created with a license start disabled
func enableItem(t *testing.T, id int) {
	item, err := Db.GetItemIncludingDisabled(id)
	require.NoError(t, err)
	item.Disabled = false
	require.NoError(t, Db.UpdateItem(id, item))
}

func TestGetCurrentOnlineIssue(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	_, found, err := Db.GetCurrentOnlineIssue()
	require.NoError(t, err)
	require.False(t, found, "no online issue yet")

	createIssue := func(number string, disabled bool) int {
		id, _, err := Db.CreateOnlineIssueWithLicense(Item{
			Name: "Online Ausgabe " + number, Description: "Online issue " + number, Price: 300,
			LicenseGroup: null.StringFrom(number), Disabled: disabled,
		}, 50)
		require.NoError(t, err)
		if !disabled {
			enableItem(t, id)
		}
		return id
	}
	published := createIssue("649", false)
	current := createIssue("651", false)
	createIssue("650", false)             // draft in WordPress
	createIssue("652", false)             // demo in WordPress
	createIssue("653", true)              // prepared, not published in the backend
	createIssue("digital_edition", false) // not an issue in WordPress

	// Without WordPress the newest enabled issue wins
	setWordPressInvite(t, "", "")
	issue, found, err := Db.GetCurrentOnlineIssue()
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "digital_edition", issue.LicenseGroup.String)

	var asked []string
	wp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer key", r.Header.Get("Authorization"))
		number := strings.TrimPrefix(r.URL.Path, "/wp-json/augustin/v1/shop/issue/")
		asked = append(asked, number)
		switch number {
		case "649", "651":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"` + number + `","published":true,"is_demo":false}`))
		case "650":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"650","published":false,"is_demo":false}`))
		case "652":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"652","published":true,"is_demo":true}`))
		default:
			_, _ = w.Write([]byte(`{"exists":false,"issue_number":"` + number + `"}`))
		}
	}))
	defer wp.Close()
	setWordPressInvite(t, wp.URL+"/wp-json/augustin/v1/shop/create-invite", "key")
	defer setWordPressInvite(t, "", "")

	// With WordPress, unknown, demo and unpublished issues are skipped
	issue, found, err = Db.GetCurrentOnlineIssue()
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, current, issue.ID)
	require.Equal(t, []string{"digital_edition", "652", "650", "651"}, asked)

	// When WordPress is unreachable, the newest candidate is used
	wp.Close()
	issue, found, err = Db.GetCurrentOnlineIssue()
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "digital_edition", issue.LicenseGroup.String)
	require.NotEqual(t, published, issue.ID)
}

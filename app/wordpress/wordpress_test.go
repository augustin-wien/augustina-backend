package wordpress

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIssueNumber(t *testing.T) {
	require.Equal(t, "616", IssueNumber("616"))
	require.Equal(t, "digital_edition", IssueNumber("digital_edition"))
	require.Equal(t, "", IssueNumber(""))
	require.Equal(t, "", IssueNumber("digitale Ausgabe"))
	require.Equal(t, "", IssueNumber("../admin"))
}

func TestCreateInvite(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer key", r.Header.Get("Authorization"))
		require.Equal(t, "augustina-backend", r.Header.Get("User-Agent"))
		raw, _ := io.ReadAll(r.Body)
		body = nil
		require.NoError(t, json.Unmarshal(raw, &body))
		_, _ = w.Write([]byte(`{"magic_link":"https://wp.test/app/willkommen?token=t","expires_at":"2030-01-01T00:00:00+00:00"}`))
	}))
	defer server.Close()

	link, err := CreateInvite(server.URL, "key", Invite{Email: "person@example.com", IssueNumber: "616", FirstName: "Maria", LastName: "Muster", TTL: 2592000})
	require.NoError(t, err)
	require.Equal(t, "https://wp.test/app/willkommen?token=t", link)
	require.Equal(t, map[string]any{
		"email": "person@example.com", "issue_number": "616",
		"first_name": "Maria", "last_name": "Muster", "ttl": float64(2592000),
	}, body)

	// Unknown values are left out instead of sent empty
	_, err = CreateInvite(server.URL, "key", Invite{Email: "person@example.com", TTL: 60})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"email": "person@example.com", "ttl": float64(60)}, body)
}

func TestCreateInviteErrors(t *testing.T) {
	var status int
	var answer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	defer server.Close()
	inviteURL := server.URL + "/wp-json/augustin/v1/shop/create-invite?secret=x"
	logged := server.URL + "/wp-json/augustin/v1/shop/create-invite"

	invite := func() *StatusError {
		_, err := CreateInvite(inviteURL, "key", Invite{Email: "person@example.com", TTL: 60})
		var statusErr *StatusError
		require.ErrorAs(t, err, &statusErr)
		return statusErr
	}

	// Something other than WordPress answers, e.g. a removed site
	status, answer = http.StatusGone, ""
	err := invite()
	require.Equal(t, "wordpress.CreateInvite: POST "+logged+": 410 Gone, empty body; not an answer of the augustin-ki plugin, check the WordPress invite URL setting", err.Error())
	require.Contains(t, err.Hint(), "keinen Einladungs-Endpunkt (Status 410)")
	require.NotContains(t, err.Hint(), "secret")

	status, answer = http.StatusBadGateway, "<html>\n<h1>Bad   Gateway</h1></html>"
	err = invite()
	require.Contains(t, err.Error(), `502 Bad Gateway, body "<html> <h1>Bad Gateway</h1></html>"`)
	require.Contains(t, err.Hint(), "Status 502")

	// The plugin answers with a WP_Error
	status, answer = http.StatusUnauthorized, `{"code":"shop_unauthorized","message":"Ungültiger oder fehlender API-Schlüssel.","data":{"status":401}}`
	err = invite()
	require.Equal(t, "wordpress.CreateInvite: POST "+logged+": 401 Unauthorized: Ungültiger oder fehlender API-Schlüssel. (shop_unauthorized)", err.Error())
	require.Contains(t, err.Hint(), "API-Schlüssel abgelehnt")

	status, answer = http.StatusServiceUnavailable, `{"code":"shop_disabled","message":"Shop-API ist deaktiviert."}`
	require.Equal(t, "WordPress meldet einen Fehler (Status 503): Shop-API ist deaktiviert.", invite().Hint())
}

func TestSnippet(t *testing.T) {
	long := strings.Repeat("ä", 300)
	require.Equal(t, strings.Repeat("ä", 200)+"…", snippet([]byte(long)))
}

func TestLookupIssue(t *testing.T) {
	var path, rawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "Bearer key", r.Header.Get("Authorization"))
		require.Equal(t, "augustina-backend", r.Header.Get("User-Agent"))
		path, rawQuery = r.URL.Path, r.URL.RawQuery
		switch r.URL.Path {
		case "/wp-json/augustin/v1/shop/issue/651":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"651","id":7,"title":"Augustin Nr. 651","status":"publish","published":true,"is_demo":false}`))
		case "/wp-json/augustin/v1/shop/issue/652":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"652","id":8,"title":"Demo","status":"publish","published":true,"is_demo":true}`))
		case "/wp-json/augustin/v1/shop/issue/650":
			_, _ = w.Write([]byte(`{"exists":true,"issue_number":"650","id":9,"title":"Entwurf","status":"draft","published":false,"is_demo":false}`))
		case "/wp-json/augustin/v1/shop/issue/old_plugin":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"rest_no_route","message":"No route was found matching the URL and request method."}`))
		default:
			_, _ = w.Write([]byte(`{"exists":false,"issue_number":"x"}`))
		}
	}))
	defer server.Close()
	inviteURL := server.URL + "/wp-json/augustin/v1/shop/create-invite?secret=x"

	issue, err := LookupIssue(inviteURL, "key", "651")
	require.NoError(t, err)
	require.Equal(t, "/wp-json/augustin/v1/shop/issue/651", path)
	require.Equal(t, "secret=x", rawQuery)
	require.Equal(t, Issue{Exists: true, IssueNumber: "651", ID: 7, Title: "Augustin Nr. 651", Status: "publish", Published: true}, issue)
	require.True(t, issue.Current())

	issue, err = LookupIssue(inviteURL, "key", "652")
	require.NoError(t, err)
	require.False(t, issue.Current(), "demo issue")

	issue, err = LookupIssue(inviteURL, "key", "650")
	require.NoError(t, err)
	require.False(t, issue.Current(), "unpublished issue")

	issue, err = LookupIssue(inviteURL, "key", "digital_edition")
	require.NoError(t, err)
	require.False(t, issue.Exists)

	_, err = LookupIssue(inviteURL, "key", "old_plugin")
	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, http.StatusNotFound, statusErr.Status)
	require.Contains(t, err.Error(), "wordpress.LookupIssue: GET")

	_, err = LookupIssue(inviteURL, "key", "../admin")
	require.Error(t, err)
	_, err = LookupIssue(server.URL+"/wp-json/augustin/v1/other", "key", "651")
	require.Error(t, err)
}

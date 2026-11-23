package wordpress

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateInviteResponses(t *testing.T) {
	cases := map[string]struct {
		status  int
		body    string
		link    string
		wantErr bool
	}{
		"url field":    {200, `{"url":"https://example.com/login"}`, "https://example.com/login", false},
		"no link":      {200, `{}`, "", true},
		"invalid json": {200, `nope`, "", true},
		"html error":   {502, `<html>Bad Gateway</html>`, "", true},
		"wp error":     {403, `{"code":"forbidden","message":"Invalid key"}`, "", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			link, err := CreateInvite(server.URL+"/wp-json/augustin/v1/shop/create-invite", "key", Invite{Email: "a@example.com"})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.link, link)
		})
	}

	link, err := CreateInvite("", "key", Invite{})
	require.NoError(t, err)
	require.Empty(t, link)

	_, err = CreateInvite("http://127.0.0.1:1/create-invite", "key", Invite{})
	require.Error(t, err)
	_, err = CreateInvite("://invalid", "key", Invite{})
	require.Error(t, err)
}

func TestLookupIssueErrors(t *testing.T) {
	_, err := LookupIssue("http://example.com/create-invite", "key", "")
	require.Error(t, err)
	_, err = LookupIssue("http://example.com/other-endpoint", "key", "651")
	require.Error(t, err)
	_, err = LookupIssue("://invalid", "key", "651")
	require.Error(t, err)
	_, err = LookupIssue("http://127.0.0.1:1/create-invite", "key", "651")
	require.Error(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/shop/issue/500" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("<html>oops</html>"))
			return
		}
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()
	_, err = LookupIssue(server.URL+"/shop/create-invite", "key", "651")
	require.Error(t, err)
	_, err = LookupIssue(server.URL+"/shop/create-invite", "key", "500")
	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	require.Contains(t, statusErr.Body, "oops")

	require.Equal(t, "(invalid URL)", endpoint("://invalid"))
}

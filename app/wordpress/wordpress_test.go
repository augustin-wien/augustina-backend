package wordpress

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

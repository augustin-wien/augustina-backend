package wordpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// Invite is what WordPress needs to create a one-time login link. When the
// link is redeemed, WordPress creates the Keycloak user if it does not exist
// yet (with the given names) and adds it to /customer/newspapers/{IssueNumber}.
type Invite struct {
	Email       string `json:"email"`
	IssueNumber string `json:"issue_number,omitempty"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	TTL         int    `json:"ttl"`
}

// issueNumberPattern is what WordPress accepts as issue_number; it becomes
// part of a Keycloak group path
var issueNumberPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// IssueNumber turns an item's LicenseGroup into the invite's issue_number.
// The Keycloak group the backend assigns is /customer/newspapers/{LicenseGroup},
// so both name the same group. A value WordPress would reject is left out, so
// the invite still works.
func IssueNumber(licenseGroup string) string {
	if !issueNumberPattern.MatchString(licenseGroup) {
		return ""
	}
	return licenseGroup
}

// inviteResponse is the answer of the augustin-ki plugin's create-invite
// endpoint. Older versions answered with "url" instead of "magic_link".
type inviteResponse struct {
	MagicLink string `json:"magic_link"`
	URL       string `json:"url"`
}

// errorResponse is the body WordPress sends for a WP_Error
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateInvite calls the WordPress one-time login API and returns the invite URL.
// baseURL must include the full endpoint path, e.g.
// "http://host.docker.internal:8088/wp-json/augustin/v1/shop/create-invite".
// Returns an empty string (no error) when baseURL or apiKey is not configured.
func CreateInvite(baseURL, apiKey string, invite Invite) (string, error) {
	if baseURL == "" || apiKey == "" {
		return "", nil
	}

	body, err := json.Marshal(invite)
	if err != nil {
		return "", fmt.Errorf("wordpress.CreateInvite: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("wordpress.CreateInvite: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("wordpress.CreateInvite: send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		var wpErr errorResponse
		if json.Unmarshal(raw, &wpErr) == nil && wpErr.Message != "" {
			return "", fmt.Errorf("wordpress.CreateInvite: status %d: %s", resp.StatusCode, wpErr.Message)
		}
		return "", fmt.Errorf("wordpress.CreateInvite: unexpected status %d: %s", resp.StatusCode, raw)
	}

	var result inviteResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("wordpress.CreateInvite: decode response: %w", err)
	}
	if result.MagicLink != "" {
		return result.MagicLink, nil
	}
	if result.URL != "" {
		return result.URL, nil
	}
	return "", fmt.Errorf("wordpress.CreateInvite: response contains no login link")
}

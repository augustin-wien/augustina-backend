package wordpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		statusErr := &StatusError{URL: endpoint(baseURL), Status: resp.StatusCode}
		var wpErr errorResponse
		if json.Unmarshal(raw, &wpErr) == nil && wpErr.Message != "" {
			statusErr.Code, statusErr.Message = wpErr.Code, wpErr.Message
		} else {
			statusErr.Body = snippet(raw)
		}
		return "", statusErr
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

// StatusError is a non-2xx answer to an invite request. Code and Message are
// set when WordPress answered with a WP_Error; otherwise the answer did not
// come from the augustin-ki plugin and Body holds the start of it.
type StatusError struct {
	URL     string
	Status  int
	Code    string
	Message string
	Body    string
}

func (e *StatusError) Error() string {
	status := fmt.Sprintf("%d %s", e.Status, http.StatusText(e.Status))
	if e.Message != "" {
		return fmt.Sprintf("wordpress.CreateInvite: POST %s: %s: %s (%s)", e.URL, status, e.Message, e.Code)
	}
	body := "empty body"
	if e.Body != "" {
		body = fmt.Sprintf("body %q", e.Body)
	}
	return fmt.Sprintf("wordpress.CreateInvite: POST %s: %s, %s; not an answer of the augustin-ki plugin, check the WordPress invite URL setting", e.URL, status, body)
}

// Hint explains the error to the admin who configures the invite settings
func (e *StatusError) Hint() string {
	switch {
	case e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden:
		return fmt.Sprintf("WordPress hat den API-Schlüssel abgelehnt (Status %d). Bitte den Schlüssel mit dem im augustin-ki-Plugin vergleichen.", e.Status)
	case e.Message != "":
		return fmt.Sprintf("WordPress meldet einen Fehler (Status %d): %s", e.Status, e.Message)
	case e.Status == http.StatusNotFound || e.Status == http.StatusGone:
		return fmt.Sprintf("Unter %s gibt es keinen Einladungs-Endpunkt (Status %d). Bitte prüfen, ob die API-URL stimmt (…/wp-json/augustin/v1/shop/create-invite) und das augustin-ki-Plugin aktiv ist.", e.URL, e.Status)
	case e.Status >= 500:
		return fmt.Sprintf("Der Server unter %s ist gerade nicht erreichbar oder hat einen Fehler (Status %d). Bitte später erneut versuchen.", e.URL, e.Status)
	default:
		return fmt.Sprintf("Unter %s antwortet kein augustin-ki-Plugin (Status %d). Bitte die API-URL prüfen.", e.URL, e.Status)
	}
}

// endpoint is baseURL without query and credentials, so it can be logged
func endpoint(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "(invalid URL)"
	}
	return u.Scheme + "://" + u.Host + u.Path
}

// snippet shortens a response body for an error message
func snippet(raw []byte) string {
	s := strings.Join(strings.Fields(string(raw)), " ")
	if r := []rune(s); len(r) > 200 {
		return string(r[:200]) + "…"
	}
	return s
}

package wordpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type inviteRequest struct {
	Email string `json:"email"`
	TTL   int    `json:"ttl"`
}

type inviteResponse struct {
	URL string `json:"url"`
}

// CreateInvite calls the WordPress one-time login API and returns the invite URL.
// baseURL must include the full endpoint path, e.g.
// "http://host.docker.internal:8088/wp-json/augustin/v1/shop/create-invite".
// Returns an empty string (no error) when baseURL or apiKey is not configured.
func CreateInvite(baseURL, apiKey, email string, ttl int) (string, error) {
	if baseURL == "" || apiKey == "" {
		return "", nil
	}

	body, err := json.Marshal(inviteRequest{Email: email, TTL: ttl})
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
		return "", fmt.Errorf("wordpress.CreateInvite: unexpected status %d: %s", resp.StatusCode, raw)
	}

	var result inviteResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("wordpress.CreateInvite: decode response: %w", err)
	}
	return result.URL, nil
}

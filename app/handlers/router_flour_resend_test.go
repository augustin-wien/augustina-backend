package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

// TestFlourResendEndpointRouted verifies the resend endpoint is wired and requires auth.
func TestFlourResendEndpointRouted(t *testing.T) {
	origFlour := config.Config.FlourWebhookURL
	config.Config.FlourWebhookURL = "http://localhost:8081/flour"
	defer func() { config.Config.FlourWebhookURL = origFlour }()

	r := GetRouter()

	// Without auth headers, expect 401 due to FlourAuthMiddleware/AuthMiddleware
	req := httptest.NewRequest(http.MethodPost, "/api/flour/payments/resend/123/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Fatalf("expected auth-required status (401/403), got %d", rr.Code)
	}
}

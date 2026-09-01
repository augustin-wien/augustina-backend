package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFlourResendEndpointRouted verifies the resend endpoint is wired and requires auth.
// The endpoint lives under /api/orders/ and not under the flour prefix: splitting the
// flour integration into a flour and an odoo plugin moved it there, because it is the
// backoffice that resends a webhook, not the external software.
func TestFlourResendEndpointRouted(t *testing.T) {
	r := GetRouter()

	// Without auth headers, expect 401 from AuthMiddleware, or 403 from the user agent
	// blocking that this route, unlike the flour ones, sits behind.
	req := httptest.NewRequest(http.MethodPost, "/api/orders/resend/123/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Fatalf("expected auth-required status (401/403), got %d", rr.Code)
	}
}

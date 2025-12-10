package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

// TestRouterHealthReadyFlour ensures that the router exposes health, ready, and flour endpoints.
func TestRouterHealthReadyFlour(t *testing.T) {
	// Preserve and restore config values modified for this test.
	origFrontend := config.Config.FrontendURL
	origFlour := config.Config.FlourWebhookURL
	config.Config.FrontendURL = "http://localhost"
	config.Config.FlourWebhookURL = "http://localhost:8081/flour"
	t.Cleanup(func() {
		config.Config.FrontendURL = origFrontend
		config.Config.FlourWebhookURL = origFlour
	})

	router := GetRouter()

	// /healthz should be 200 OK.
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected /healthz 200, got %d", rr.Code)
	}

	// /readyz should return 200 when DB is available or 503 when unavailable.
	// In test environment, DB is initialized, so we expect 200.
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /readyz 200 or 503, got %d", rr.Code)
	}

	// Flour endpoint should be routed; without auth expect 401 (AuthMiddleware).
	req = httptest.NewRequest(http.MethodGet, "/api/flour/vendors/license/demo/", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected flour endpoint to require auth (401), got %d", rr.Code)
	}
}

// TestHealthAndFlourEndpointsNotBlockedByUserAgent verifies that health, ready,
// and flour endpoints do NOT require specific browser User-Agent headers.
// This is important because Flour is external software we don't control.
func TestHealthAndFlourEndpointsNotBlockedByUserAgent(t *testing.T) {
	origFrontend := config.Config.FrontendURL
	origFlour := config.Config.FlourWebhookURL
	config.Config.FrontendURL = "http://localhost"
	config.Config.FlourWebhookURL = "http://localhost:8081/flour"
	t.Cleanup(func() {
		config.Config.FrontendURL = origFrontend
		config.Config.FlourWebhookURL = origFlour
	})

	router := GetRouter()

	// Test /healthz WITHOUT any User-Agent header - should work
	// Health endpoints must not be blocked by UserAgent middleware
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("expected /healthz to work without User-Agent header, got 403")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected /healthz 200, got %d", rr.Code)
	}

	// Test /readyz WITHOUT any User-Agent header - should work
	// Ready endpoints must not be blocked by UserAgent middleware
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("expected /readyz to work without User-Agent header, got 403")
	}

	// Test /api/flour/vendors/license/test/ WITHOUT any User-Agent header
	// Flour endpoints must not be blocked by UserAgent middleware.
	// Will be blocked by AuthMiddleware (401) instead, which is expected.
	req = httptest.NewRequest(http.MethodGet, "/api/flour/vendors/license/test/", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("expected /api/flour/vendors/license/test/ to NOT be blocked by UserAgent middleware (should fail at auth layer instead), got 403")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected /api/flour/vendors/license/test/ to return 401 (auth required), got %d", rr.Code)
	}
}

package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

// TestSecurityHeaders ensures basic security headers are set by the router middleware.
func TestSecurityHeaders(t *testing.T) {
	// ensure frontend URL is set so router initialization succeeds
	config.Config.FrontendURL = "http://localhost:3000"
	// enable development mode so docs/swagger are available during tests if needed
	config.Config.Development = true

	r := GetRouter()

	req := httptest.NewRequest("GET", "/api/hello/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	headers := rec.Header()

	if got := headers.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected X-Frame-Options=DENY, got %q", got)
	}
	if got := headers.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options=nosniff, got %q", got)
	}
	if got := headers.Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("expected Referrer-Policy=no-referrer, got %q", got)
	}
	if got := headers.Get("X-XSS-Protection"); got != "1; mode=block" {
		t.Fatalf("expected X-XSS-Protection=1; mode=block, got %q", got)
	}
	if got := headers.Get("Content-Security-Policy"); got == "" {
		t.Fatalf("expected Content-Security-Policy header to be present")
	}
}

// TestCORSBehavior checks that allowed origins receive CORS headers and disallowed origins do not.
func TestCORSBehavior(t *testing.T) {
	config.Config.FrontendURL = "http://example-frontend.local"
	config.Config.Development = true

	r := GetRouter()

	// Allowed origin: configured frontend
	req := httptest.NewRequest("OPTIONS", "/api/hello/", nil)
	req.Header.Set("Origin", "http://example-frontend.local")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example-frontend.local" {
		t.Fatalf("expected Access-Control-Allow-Origin to echo origin, got %q", got)
	}

	// Allowed origin: localhost variant
	req2 := httptest.NewRequest("OPTIONS", "/api/hello/", nil)
	req2.Header.Set("Origin", "http://localhost:3000")
	req2.Header.Set("Access-Control-Request-Method", "GET")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected localhost origin to be allowed, got %q", got)
	}

	// Disallowed origin should not receive CORS allow header
	req3 := httptest.NewRequest("OPTIONS", "/api/hello/", nil)
	req3.Header.Set("Origin", "http://evil.example")
	req3.Header.Set("Access-Control-Request-Method", "GET")
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if got := rec3.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected disallowed origin to receive no Access-Control-Allow-Origin but got %q", got)
	}
}

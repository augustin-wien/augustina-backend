package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/middlewares"
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

	// Lookalike localhost origin (attacker-controlled domain) must be rejected.
	req4 := httptest.NewRequest("OPTIONS", "/api/hello/", nil)
	req4.Header.Set("Origin", "http://localhost.evil.com")
	req4.Header.Set("Access-Control-Request-Method", "GET")
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, req4)
	if got := rec4.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected lookalike localhost origin to be rejected but got %q", got)
	}
}

// TestIsAllowedOrigin exercises the origin allowlist directly, including the lookalike
// domains that a naive prefix check would have accepted.
func TestIsAllowedOrigin(t *testing.T) {
	const frontend = "https://shop.example.org"

	allowed := []string{
		frontend,
		"http://localhost",
		"http://localhost:3000",
		"https://localhost:5173",
		"http://127.0.0.1:8080",
		"http://[::1]:3000",
	}
	for _, origin := range allowed {
		if !isAllowedOrigin(frontend, origin, true) {
			t.Errorf("expected origin %q to be allowed", origin)
		}
	}

	rejected := []string{
		"",
		"http://localhost.evil.com",
		"http://localhostx",
		"http://notlocalhost.com",
		"https://evil.com",
		"http://localhost@evil.com",
		"ftp://localhost",
		"https://shop.example.org.evil.com",
	}
	for _, origin := range rejected {
		if isAllowedOrigin(frontend, origin, true) {
			t.Errorf("expected origin %q to be rejected", origin)
		}
	}
}

// TestIsAllowedOriginWithoutLocalhost pins the production behaviour: outside development the
// loopback origins are no longer trusted. Combined with AllowCredentials, trusting them there
// would let any page served from the visitor's own machine talk to the production API.
func TestIsAllowedOriginWithoutLocalhost(t *testing.T) {
	const frontend = "https://shop.example.org"

	if !isAllowedOrigin(frontend, frontend, false) {
		t.Errorf("expected the configured frontend to stay allowed")
	}

	rejected := []string{
		"http://localhost",
		"http://localhost:3000",
		"https://localhost:5173",
		"http://127.0.0.1:8080",
		"http://[::1]:3000",
	}
	for _, origin := range rejected {
		if isAllowedOrigin(frontend, origin, false) {
			t.Errorf("expected origin %q to be rejected outside development", origin)
		}
	}
}

// TestCORSRejectsLocalhostOutsideDevelopment drives the same thing through the real router.
func TestCORSRejectsLocalhostOutsideDevelopment(t *testing.T) {
	config.Config.FrontendURL = "http://example-frontend.local"

	// Restore whatever the other tests in this package rely on.
	original := config.Config.Development
	config.Config.Development = false
	defer func() { config.Config.Development = original }()

	r := GetRouter()

	req := httptest.NewRequest("OPTIONS", "/api/hello/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected localhost origin to be rejected outside development but got %q", got)
	}

	// The configured frontend still works.
	req2 := httptest.NewRequest("OPTIONS", "/api/hello/", nil)
	req2.Header.Set("Origin", "http://example-frontend.local")
	req2.Header.Set("Access-Control-Request-Method", "GET")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "http://example-frontend.local" {
		t.Fatalf("expected the configured frontend to stay allowed, got %q", got)
	}
}

// TestLegitimateAdminRouteNotBlocked guards against the scanner blocklist swallowing our own
// routes: "/api/settings/admin/" contains the suspicious substring "/admin/", which used to
// answer 403, block the caller's IP for 24h, and - because the block ran before the CORS
// middleware - surface in the browser as a missing Access-Control-Allow-Origin header.
func TestLegitimateAdminRouteNotBlocked(t *testing.T) {
	config.Config.FrontendURL = "http://example-frontend.local"
	config.Config.Development = true

	r := GetRouter()

	const ip = "203.0.113.7"

	// Preflight for the admin settings route must succeed and carry the CORS headers.
	pre := httptest.NewRequest("OPTIONS", "/api/settings/admin/", nil)
	pre.Header.Set("Origin", "http://example-frontend.local")
	pre.Header.Set("Access-Control-Request-Method", "GET")
	pre.Header.Set("X-Real-Ip", ip)
	preRec := httptest.NewRecorder()
	r.ServeHTTP(preRec, pre)

	if preRec.Code == http.StatusForbidden {
		t.Fatalf("preflight for /api/settings/admin/ was blocked with 403")
	}
	if got := preRec.Header().Get("Access-Control-Allow-Origin"); got != "http://example-frontend.local" {
		t.Fatalf("expected preflight to carry Access-Control-Allow-Origin, got %q", got)
	}

	// The request itself must reach the auth middleware (401 without a token) rather than
	// being rejected as a scanner probe.
	req := httptest.NewRequest("GET", "/api/settings/admin/", nil)
	req.Header.Set("Origin", "http://example-frontend.local")
	req.Header.Set("X-Real-Ip", ip)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 from the auth middleware, got %d", rec.Code)
	}
	if middlewares.GlobalBlocker.IsBlocked(ip) {
		t.Fatalf("legitimate request to /api/settings/admin/ got the caller's IP blocked")
	}

	// A real scanner probe on a path we do not serve is still blocked.
	probe := httptest.NewRequest("GET", "/wp-login.php", nil)
	probe.Header.Set("X-Real-Ip", "203.0.113.8")
	probeRec := httptest.NewRecorder()
	r.ServeHTTP(probeRec, probe)

	if probeRec.Code != http.StatusForbidden {
		t.Fatalf("expected scanner probe to be blocked, got %d", probeRec.Code)
	}
}

package utils

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

func TestGetEnvFallback(t *testing.T) {
	key := "TEST_UTILS_ENV"
	os.Unsetenv(key)
	val := GetEnv(key, "fallback")
	if val != "fallback" {
		t.Fatalf("expected fallback, got %q", val)
	}
	os.Setenv(key, "value")
	val = GetEnv(key, "fallback")
	if val != "value" {
		t.Fatalf("expected value, got %q", val)
	}
}

func TestRandomString(t *testing.T) {
	s := RandomString(10)
	if len(s) != 10 {
		t.Fatalf("expected length 10, got %d", len(s))
	}
}

func TestReadUserIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-Ip", "1.2.3.4")
	ip := ReadUserIP(req)
	if ip != "1.2.3.4" {
		t.Fatalf("unexpected ip, got %q", ip)
	}
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	ip = ReadUserIP(req)
	if ip != "5.6.7.8" {
		t.Fatalf("unexpected ip, got %q", ip)
	}
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "9.10.11.12:1234"
	ip = ReadUserIP(req)
	if ip != "9.10.11.12:1234" {
		t.Fatalf("unexpected ip, got %q", ip)
	}
}

// TestReadUserIPStrictIgnoresUntrustedProxy ensures that once TRUSTED_PROXIES is configured,
// forwarding headers are ignored for requests that do not come from a listed proxy - so an
// attacker cannot spoof their IP to poison the blocklist or evade a block.
func TestReadUserIPStrictIgnoresUntrustedProxy(t *testing.T) {
	original := config.Config.TrustedProxies
	config.Config.TrustedProxies = []string{"10.0.0.1"}
	defer func() { config.Config.TrustedProxies = original }()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	req.Header.Set("X-Real-Ip", "203.0.113.9")

	if ip := ReadUserIP(req); ip != "1.2.3.4" {
		t.Fatalf("expected untrusted peer IP 1.2.3.4, got %q", ip)
	}
}

// TestReadUserIPStrictTrustsListedProxy ensures that when the request does come from a
// trusted proxy, the real client IP is taken from the forwarding header.
func TestReadUserIPStrictTrustsListedProxy(t *testing.T) {
	original := config.Config.TrustedProxies
	config.Config.TrustedProxies = []string{"10.0.0.1"}
	defer func() { config.Config.TrustedProxies = original }()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:5555"
	// Right-most non-proxy entry is the closest real client the chain saw.
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")

	if ip := ReadUserIP(req); ip != "203.0.113.9" {
		t.Fatalf("expected client IP 203.0.113.9 from trusted proxy, got %q", ip)
	}
}

package utils

import (
	"net/http/httptest"
	"os"
	"testing"
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

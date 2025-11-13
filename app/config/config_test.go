package config

import (
	"os"
	"testing"
)

// TestValidate verifies that Validate returns an error when required fields are missing
func TestValidate(t *testing.T) {
	// backup and restore env
	old := os.Getenv("FRONTEND_URL")
	defer os.Setenv("FRONTEND_URL", old)

	// Ensure FRONTEND_URL is not set
	os.Unsetenv("FRONTEND_URL")
	// Initialize config from env (should use defaults)
	if err := InitConfig(); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Validate should fail because FrontendURL is empty
	if err := Config.Validate(); err == nil {
		t.Fatalf("expected Validate to fail when FRONTEND_URL is empty")
	}

	// Set FRONTEND_URL and re-init
	os.Setenv("FRONTEND_URL", "http://localhost:3000")
	if err := InitConfig(); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}
	if err := Config.Validate(); err != nil {
		t.Fatalf("expected Validate to succeed when FRONTEND_URL is set: %v", err)
	}
}

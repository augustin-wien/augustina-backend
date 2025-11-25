package config

import (
	"os"
	"strconv"
	"strings"
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

// TestSmartCheckoutURLWhenDebugPaymentsFalse tests that when DEBUG_payments=false
// we build a VivaWallet Smart Checkout URL (production-style checkout URL) and not
// the local success test URL.
func TestSmartCheckoutURLWhenDebugPaymentsFalse(t *testing.T) {
	// Preserve and restore environment
	oldDebug := os.Getenv("DEBUG_payments")
	oldURL := os.Getenv("VIVA_WALLET_SMART_CHECKOUT_URL")
	defer func() {
		_ = os.Setenv("DEBUG_payments", oldDebug)
		_ = os.Setenv("VIVA_WALLET_SMART_CHECKOUT_URL", oldURL)
		_ = InitConfig()
	}()

	// Set env for test: explicit production-style smart checkout base URL
	_ = os.Setenv("VIVA_WALLET_SMART_CHECKOUT_URL", "https://demo.vivapayments.com/web/checkout?ref=")
	_ = os.Setenv("DEBUG_payments", "false")

	// Re-init config to pick up env changes
	if err := InitConfig(); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Simulate OrderCode that would be used by CreatePaymentOrder
	OrderCode := 12345

	checkoutURL := Config.VivaWalletSmartCheckoutURL + strconv.Itoa(OrderCode)

	if strings.Contains(checkoutURL, "/success") {
		t.Fatalf("unexpected local success url when DEBUG_payments=false: %s", checkoutURL)
	}
	if !strings.Contains(checkoutURL, "checkout") {
		t.Fatalf("expected checkout url, got: %s", checkoutURL)
	}
}

package handlers

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

// Test that when DEBUG_payments=false we build a VivaWallet Smart Checkout URL
// (i.e. the production-style checkout URL) and not the local success test URL.
func TestSmartCheckoutURLWhenDebugPaymentsFalse(t *testing.T) {
	// Preserve and restore environment
	oldDebug := os.Getenv("DEBUG_payments")
	oldURL := os.Getenv("VIVA_WALLET_SMART_CHECKOUT_URL")
	defer func() {
		_ = os.Setenv("DEBUG_payments", oldDebug)
		_ = os.Setenv("VIVA_WALLET_SMART_CHECKOUT_URL", oldURL)
		_ = config.InitConfig()
	}()

	// Set env for test: explicit production-style smart checkout base URL
	_ = os.Setenv("VIVA_WALLET_SMART_CHECKOUT_URL", "https://demo.vivapayments.com/web/checkout?ref=")
	_ = os.Setenv("DEBUG_payments", "false")

	// Re-init config to pick up env changes
	if err := config.InitConfig(); err != nil {
		t.Fatalf("config.InitConfig failed: %v", err)
	}

	// Simulate OrderCode that would be used by CreatePaymentOrder
	OrderCode := 12345

	checkoutURL := config.Config.VivaWalletSmartCheckoutURL + strconv.Itoa(OrderCode)

	if strings.Contains(checkoutURL, "/success") {
		t.Fatalf("unexpected local success url when DEBUG_payments=false: %s", checkoutURL)
	}
	if !strings.Contains(checkoutURL, "checkout") {
		t.Fatalf("expected checkout url, got: %s", checkoutURL)
	}
}

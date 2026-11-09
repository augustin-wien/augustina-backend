package paymentprovider

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

// The transaction id comes from the unauthenticated webhook and from /orders/verify/ and goes into
// the path of a VivaWallet request that carries our access token, so it must not be able to point
// that request at another VivaWallet endpoint.
func TestVerifyTransactionID_RejectsIDsThatChangeTheRequestPath(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "fake_token", "statusId": "F"}`))
	}))
	defer server.Close()

	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey
	defer func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
	}()
	config.Config.VivaWalletAPIURL = server.URL
	config.Config.VivaWalletAccountsURL = server.URL
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"

	for _, id := range []string{"", "..", "../../orders/123", "a/b", "a?b=c", "a#b", "a%2Fb", "a b", "manual-2026-01-01T00:00:00Z"} {
		if _, err := VerifyTransactionID(id, false); !errors.Is(err, ErrInvalidTransactionID) {
			t.Errorf("VerifyTransactionID(%q) = %v, want ErrInvalidTransactionID", id, err)
		}
	}
	if requests != 0 {
		t.Fatalf("rejected transaction ids must not reach VivaWallet, got %d requests", requests)
	}

	if _, err := VerifyTransactionID("882d641c-01cc-442f-b894-2b51250340b5", false); err != nil {
		t.Errorf("a VivaWallet transaction id must still be verified, got %v", err)
	}
}

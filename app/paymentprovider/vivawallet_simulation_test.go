package paymentprovider

import (
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
)

// The VivaWallet webhook endpoint is unauthenticated, and the simulated branch trusts the request
// body instead of verifying it against VivaWallet. That shortcut may only ever exist in
// development — in production it would turn any unpaid order into a paid one.
func TestIsSimulatedTransaction(t *testing.T) {
	original := config.Config.Development
	defer func() { config.Config.Development = original }()

	config.Config.Development = false
	for _, id := range []string{
		"dev-simulation-1",
		"dev-simulation-",
		"dev-simulation-anything-goes",
	} {
		if isSimulatedTransaction(id) {
			t.Errorf("transaction %q must not skip verification outside development", id)
		}
	}

	config.Config.Development = true
	if !isSimulatedTransaction("dev-simulation-1") {
		t.Error("expected the simulation to be recognised in development")
	}
	for _, id := range []string{"", "12345", "manual-1", "xdev-simulation-1"} {
		if isSimulatedTransaction(id) {
			t.Errorf("transaction %q is not a simulation", id)
		}
	}
}

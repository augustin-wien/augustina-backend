package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Tests run from jobs/, but config/migrations expect the app root, same as database's TestMain.
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	config.InitConfig()
	if err := database.Db.InitEmptyTestDb(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// createUnverifiedOrder inserts an order directly via ent, bypassing database.Db.CreateOrder -
// which always forces Timestamp to time.Now() and fabricates a non-empty TransactionID when
// given none, both of which this test needs to control precisely.
func createUnverifiedOrder(t *testing.T, transactionID string, age time.Duration) int {
	t.Helper()

	// InitEmptyTestDb seeds a handful of system vendors (Cash, Orga, ...) with
	// non-deterministic IDs - paymentorder.vendor_id has a real FK constraint, so any one of
	// them will do.
	v, err := database.Db.EntClient.Vendor.Query().First(context.Background())
	require.NoError(t, err)

	// paymentorder.timestamp is TIMESTAMP WITHOUT TIME ZONE, and pq reads it back
	// assuming UTC - CreateOrder always normalizes with .UTC() before writing for
	// exactly this reason (see database/queries.orders.go), so this must too, or a
	// non-UTC local machine round-trips it back offset by its own UTC difference.
	o, err := database.Db.EntClient.Order.Create().
		SetVendorID(v.ID).
		SetVerified(false).
		SetTransactionTypeID(0).
		SetTransactionID(transactionID).
		SetTimestamp(time.Now().UTC().Add(-age)).
		Save(context.Background())
	require.NoError(t, err)

	return o.ID
}

func isInvalidated(t *testing.T, orderID int) bool {
	t.Helper()

	o, err := database.Db.EntClient.Order.Get(context.Background(), orderID)
	require.NoError(t, err)

	return o.InvalidatedAt != nil
}

// mockVivaWallet points VivaWalletAccountsURL/VivaWalletAPIURL at a test server answering
// /connect/token and /checkout/v2/transactions/{id}, and restores the previous config after.
func mockVivaWallet(t *testing.T, statusID string, transactionErr bool) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token": "fake_token", "expires_in": 3600, "token_type": "Bearer", "scope": "api"}`))
			return
		}
		if transactionErr {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"statusId": "` + statusID + `"}`))
	}))
	t.Cleanup(server.Close)

	originalAPIURL := config.Config.VivaWalletAPIURL
	originalAccountsURL := config.Config.VivaWalletAccountsURL
	originalClientID := config.Config.VivaWalletSmartCheckoutClientID
	originalClientKey := config.Config.VivaWalletSmartCheckoutClientKey

	t.Cleanup(func() {
		config.Config.VivaWalletAPIURL = originalAPIURL
		config.Config.VivaWalletAccountsURL = originalAccountsURL
		config.Config.VivaWalletSmartCheckoutClientID = originalClientID
		config.Config.VivaWalletSmartCheckoutClientKey = originalClientKey
	})

	config.Config.VivaWalletAPIURL = server.URL
	config.Config.VivaWalletAccountsURL = server.URL
	config.Config.VivaWalletSmartCheckoutClientID = "dummy"
	config.Config.VivaWalletSmartCheckoutClientKey = "dummy"
}

func TestReconcile_InvalidatesOldOrderWithNoTransactionID(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())

	id := createUnverifiedOrder(t, "", 30*time.Minute)

	reconcileUnverifiedOrders()

	require.True(t, isInvalidated(t, id), "an order that never got a transaction id should be invalidated once past invalidateTimeout")
}

func TestReconcile_LeavesRecentOrderWithNoTransactionIDAlone(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())

	id := createUnverifiedOrder(t, "", 5*time.Minute)

	reconcileUnverifiedOrders()

	require.False(t, isInvalidated(t, id), "an order still inside invalidateTimeout must not be invalidated yet")
}

func TestReconcile_NeverInvalidatesAnOrderVivaWalletConfirmsWasPaid(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	mockVivaWallet(t, "F", false)

	id := createUnverifiedOrder(t, "paid-tx-id", 30*time.Minute)

	reconcileUnverifiedOrders()

	require.False(t, isInvalidated(t, id), "an order VivaWallet confirms was paid must never be invalidated, however old it gets")

	orders, err := database.Db.GetUnverifiedOrders()
	require.NoError(t, err)
	require.Contains(t, orderIDs(orders), id, "it must stay visible for a human to resolve")
}

func TestReconcile_InvalidatesOldOrderVivaWalletConfirmsFailed(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	mockVivaWallet(t, "E", false)

	id := createUnverifiedOrder(t, "failed-tx-id", 30*time.Minute)

	reconcileUnverifiedOrders()

	require.True(t, isInvalidated(t, id), "VivaWallet positively confirming the transaction failed is safe to invalidate on")
}

func TestReconcile_DoesNotInvalidateOnTransientVivaWalletError(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	mockVivaWallet(t, "", true)

	id := createUnverifiedOrder(t, "flaky-tx-id", 30*time.Minute)

	reconcileUnverifiedOrders()

	require.False(t, isInvalidated(t, id), "a VivaWallet outage must never be treated as confirmation the payment failed")
}

func TestReconcile_InvalidatedOrdersDropOutOfGetUnverifiedOrders(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())

	id := createUnverifiedOrder(t, "", 30*time.Minute)
	reconcileUnverifiedOrders()
	require.True(t, isInvalidated(t, id))

	orders, err := database.Db.GetUnverifiedOrders()
	require.NoError(t, err)
	require.NotContains(t, orderIDs(orders), id)
}

func orderIDs(orders []database.Order) []int {
	ids := make([]int, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}

	return ids
}

func TestReconcile_InvalidatesOldOrderWithPlaceholderTransactionID(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	// A VivaWallet error must not matter: a placeholder id is never looked up there
	mockVivaWallet(t, "", true)

	id := createUnverifiedOrder(t, "manual-2026-09-23T14:03:48.139945299Z", 30*time.Minute)

	reconcileUnverifiedOrders()

	require.True(t, isInvalidated(t, id), "an order still carrying CreateOrder's placeholder id never reached VivaWallet and is abandoned")
}

func TestReconcile_LeavesRecentOrderWithPlaceholderTransactionIDAlone(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())

	id := createUnverifiedOrder(t, "manual-2026-09-23T14:03:48.139945299Z", 5*time.Minute)

	reconcileUnverifiedOrders()

	require.False(t, isInvalidated(t, id), "an order still inside invalidateTimeout must not be invalidated yet")
}

func TestShouldAlert_DeduplicatesWithinRealertInterval(t *testing.T) {
	t.Cleanup(func() { forgetResolvedAlerts(nil) })
	now := time.Now()

	require.True(t, shouldAlert(1, now), "first sighting alerts")
	require.False(t, shouldAlert(1, now.Add(15*time.Minute)), "next reconcile run must not re-alert")
	require.True(t, shouldAlert(2, now.Add(15*time.Minute)), "another order alerts independently")
	require.True(t, shouldAlert(1, now.Add(realertInterval)), "still unresolved after realertInterval alerts again")
}

func TestForgetResolvedAlerts_ReAlertsIfOrderComesBack(t *testing.T) {
	t.Cleanup(func() { forgetResolvedAlerts(nil) })
	now := time.Now()

	require.True(t, shouldAlert(1, now))
	forgetResolvedAlerts([]database.Order{{ID: 2}})
	require.True(t, shouldAlert(1, now.Add(time.Minute)), "state for an order no longer unverified is dropped")
}

package handlers

import (
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// The donation's quantity is its amount in cents, a negative one would lower the
// price of the other items, down to an order VivaWallet rejects for its zero amount
func TestCreateOrderRejectsNegativeDonation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testnegativedonation"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)

	itemID, _ := strconv.Atoi(CreateTestItem(t, "Negative donation item", 20, "", ""))
	for _, quantity := range []int{-20, -5} {
		res := utils.TestRequest(t, r, "POST", "/api/orders/", createOrderRequest{
			Entries:         []createOrderRequestEntry{{Item: itemID, Quantity: 1}, {Item: 2, Quantity: quantity}},
			VendorLicenseID: vendorLicenseID,
		}, 400)
		require.Equal(t, `{"error":{"message":"nice try! Quantity has to be greater than 0"}}`, res.Body.String())
	}

	orders, err := database.Db.GetOrders()
	require.NoError(t, err)
	require.Empty(t, orders)
}

func TestCreateOrderTrimsCustomerEmail(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testtrimemail"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)

	itemID, _ := strconv.Atoi(CreateTestItem(t, "Trim email item", 20, "", ""))
	order := createTestOrder(t, createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom("  trim_email@example.com \n"),
	})
	require.Equal(t, "trim_email@example.com", order.CustomerEmail.String)
}

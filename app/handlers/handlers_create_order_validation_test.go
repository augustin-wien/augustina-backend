package handlers

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestCreatePaymentOrderValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	startSweepViva(t)
	config.Config.Development = false

	v, _ := sweepVendor(t)
	item := sweepItem(t, "normal_item")
	license, err := database.Db.CreateItem(database.Item{Name: "Order License", Description: "Test item", Price: 100, Type: "license_item", IsLicenseItem: true})
	require.NoError(t, err)
	digital, err := database.Db.CreateItem(database.Item{Name: "Order Digital", Description: "Test item", Price: 300, Type: "online_issue", LicenseGroup: null.StringFrom("order_digital"), LicenseItem: null.IntFrom(int64(license))})
	require.NoError(t, err)
	expensive, err := database.Db.CreateItem(database.Item{Name: "Order Expensive", Description: "Test item", Price: 10_000_000, Type: "normal_item"})
	require.NoError(t, err)
	blocked, _ := sweepVendor(t)
	blocked.IsBlocked = true
	require.NoError(t, database.Db.UpdateVendor(blocked.ID, blocked))

	order := func(entries, license, email string) string {
		return fmt.Sprintf(`{"Entries":%s,"VendorLicenseID":%q,"CustomerEmail":%s}`, entries, license, email)
	}
	one := fmt.Sprintf(`[{"Item":%d,"Quantity":1}]`, item)

	for name, tc := range map[string]struct {
		body string
		want int
	}{
		"invalid json":              {"{not json", 400},
		"negative quantity":         {order(fmt.Sprintf(`[{"Item":%d,"Quantity":-1}]`, item), v.LicenseID.String, "null"), 400},
		"unknown item":              {order(`[{"Item":999999,"Quantity":1}]`, v.LicenseID.String, "null"), 400},
		"digital without email":     {order(fmt.Sprintf(`[{"Item":%d,"Quantity":1}]`, digital), v.LicenseID.String, `""`), 400},
		"duplicate items":           {order(fmt.Sprintf(`[{"Item":%d,"Quantity":1},{"Item":%d,"Quantity":2}]`, item, item), v.LicenseID.String, "null"), 400},
		"unknown vendor":            {order(one, "no-such-vendor", "null"), 400},
		"blocked vendor":            {order(one, blocked.LicenseID.String, "null"), 403},
		"order amount too high":     {order(fmt.Sprintf(`[{"Item":%d,"Quantity":1}]`, expensive), v.LicenseID.String, "null"), 400},
		"digital with email":        {order(fmt.Sprintf(`[{"Item":%d,"Quantity":1}]`, digital), v.LicenseID.String, `"buyer@example.com"`), 200},
		"normal item without email": {order(one, v.LicenseID.String, "null"), 200},
	} {
		t.Run(name, func(t *testing.T) {
			utils.TestRequestStr(t, r, "POST", "/api/orders/", tc.body, tc.want)
		})
	}

	// Without a VivaWallet login no order is created
	config.Config.VivaWalletAccountsURL = "http://127.0.0.1:1"
	utils.TestRequestStr(t, r, "POST", "/api/orders/", order(one, v.LicenseID.String, "null"), 400)

	// With DEBUG_payments the order gets a local code and the shop's own success page
	config.Config.DEBUG_payments = true
	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	settings.Color = "#AB12CD"
	require.NoError(t, database.Db.UpdateSettings(settings))
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", order(one, v.LicenseID.String, "null"), 200)
	var resp createOrderResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	require.Contains(t, resp.SmartCheckoutURL, "/success?")
	require.Contains(t, resp.SmartCheckoutURL, "&color=ab12cd")

	config.Config.VivaWalletSmartCheckoutURL = ""
	utils.TestRequestStr(t, r, "POST", "/api/orders/", order(one, v.LicenseID.String, "null"), 400)
}

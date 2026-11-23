package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// TestForgedWebhooksDoNotMarkOrdersPaid posts success webhooks an attacker could
// send for an order whose code they know
func TestForgedWebhooksDoNotMarkOrdersPaid(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())
	viva := startSweepViva(t)
	config.Config.Development = false

	v, a := sweepVendor(t)
	item := sweepItem(t, "normal_item")
	post := func(code int64, tx string, amount float64) {
		body, err := json.Marshal(webhookFor(code, tx, amount))
		require.NoError(t, err)
		call("POST", "/api/webhooks/vivawallet/success/", "", map[string]string{"Content-Type": "application/json"})
		req := httptest.NewRequest("POST", "/api/webhooks/vivawallet/success/", bytes.NewReader(body))
		utils.SubmitRequest(req, r)
	}
	verified := func(orderID int) bool {
		o, err := database.Db.GetOrderByID(orderID)
		require.NoError(t, err)
		return o.Verified
	}

	// A transaction VivaWallet does not know
	orderID, code := sweepUnverifiedOrder(t, v, a, item)
	post(code, "0b6d4b6c-0000-4000-8000-000000000001", 3)
	require.False(t, verified(orderID))

	// The development simulation outside development mode
	orderID, code = sweepUnverifiedOrder(t, v, a, item)
	post(code, fmt.Sprintf("dev-simulation-%d", code), 3)
	require.False(t, verified(orderID))

	// A real payment of a smaller amount reused for a more expensive order
	orderID, code = sweepUnverifiedOrder(t, v, a, item)
	viva.register("cheap-payment", code, 0.5)
	post(code, "cheap-payment", 3)
	require.False(t, verified(orderID))

	// A transaction id that tries to reach another VivaWallet endpoint
	orderID, code = sweepUnverifiedOrder(t, v, a, item)
	post(code, "../../orders/1", 3)
	require.False(t, verified(orderID))
}

// TestOrderPricesComeFromTheServer sends an order with a price of its own
func TestOrderPricesComeFromTheServer(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())
	startSweepViva(t)
	config.Config.Development = false

	v, _ := sweepVendor(t)
	item := sweepItem(t, "normal_item")
	body := fmt.Sprintf(`{"Entries":[{"Item":%d,"Quantity":2,"Price":1}],"VendorLicenseID":%q,"Price":1,"Amount":1}`, item, v.LicenseID.String)
	res := utils.TestRequestStr(t, r, "POST", "/api/orders/", body, 200)
	var resp createOrderResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	u, err := url.Parse(resp.SmartCheckoutURL)
	require.NoError(t, err)
	order, err := database.Db.GetOrderByOrderCode(u.Query().Get("s"))
	require.NoError(t, err)
	for _, e := range order.Entries {
		if e.Item == item {
			require.Equal(t, 300, e.Price, "the item's price from the database")
		}
	}
	require.Equal(t, 600, order.GetTotal()-transactionCostsOf(order))

	// The system items cannot be bought on their own
	donation, err := database.Db.GetItemByName(config.Config.DonationName)
	require.NoError(t, err)
	costs, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	require.NoError(t, err)
	utils.TestRequestStr(t, r, "POST", "/api/orders/", fmt.Sprintf(`{"Entries":[{"Item":%d,"Quantity":5}],"VendorLicenseID":%q}`, costs.ID, v.LicenseID.String), 400)
	utils.TestRequestStr(t, r, "POST", "/api/orders/", fmt.Sprintf(`{"Entries":[{"Item":%d,"Quantity":5}],"VendorLicenseID":%q}`, donation.ID, v.LicenseID.String), 400)
	utils.TestRequestStr(t, r, "POST", "/api/orders/", fmt.Sprintf(`{"Entries":[{"Item":%d,"Quantity":1},{"Item":%d,"Quantity":-100}],"VendorLicenseID":%q}`, item, donation.ID, v.LicenseID.String), 400)

	// Disabled and archived items are not for sale
	disabled, err := database.Db.CreateItem(database.Item{Name: "Threat disabled", Description: "Test item", Price: 1, Type: "normal_item", Disabled: true})
	require.NoError(t, err)
	archived := sweepItem(t, "normal_item")
	require.NoError(t, database.Db.DeleteItem(archived))
	for _, id := range []int{disabled, archived} {
		utils.TestRequestStr(t, r, "POST", "/api/orders/", fmt.Sprintf(`{"Entries":[{"Item":%d,"Quantity":1}],"VendorLicenseID":%q}`, id, v.LicenseID.String), 400)
	}
}

func transactionCostsOf(o database.Order) int {
	sum := 0
	costs, err := database.Db.GetItemByName(config.Config.TransactionCostsName)
	if err != nil {
		return 0
	}
	for _, e := range o.Entries {
		if e.Item == costs.ID {
			sum += e.Price * e.Quantity
		}
	}
	return sum
}

// TestInjectionAttemptsAreHarmless sends SQL and template fragments in paths and parameters
func TestInjectionAttemptsAreHarmless(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())
	sweepVendor(t)
	sweepVendor(t)

	payloads := []string{"' OR '1'='1", "1; DROP TABLE vendor;--", "%' OR 1=1 --", "{{.Config}}", "${jndi:ldap://x}"}
	for _, p := range payloads {
		q := url.QueryEscape(p)
		path := url.PathEscape(p)
		// Public lookups find nothing and do not fail
		res := call("GET", "/api/vendors/check/"+path+"/", "", nil)
		require.Contains(t, []int{http.StatusBadRequest, http.StatusForbidden}, res.Code, p)
		// Admin filters treat the value literally
		// Either the security filter rejects the request or no vendor matches
		res = call("GET", "/api/payments/forpayout/?vendor="+q, "Bearer "+adminUserToken.AccessToken, nil)
		require.Contains(t, []int{http.StatusBadRequest, http.StatusForbidden}, res.Code, p)
		res = call("GET", "/api/mail-tasks/?search="+q, "Bearer "+adminUserToken.AccessToken, nil)
		require.Contains(t, []int{http.StatusOK, http.StatusForbidden}, res.Code, p)
		if res.Code == http.StatusOK {
			var list database.MailTaskList
			require.NoError(t, json.Unmarshal(res.Body.Bytes(), &list))
			require.Empty(t, list.Tasks, p)
		}
	}
	// All vendors are still there
	vendors, err := database.Db.ListVendors()
	require.NoError(t, err)
	require.Len(t, vendors, 2)
}

// TestPathTraversalIsBlocked tries to read files outside the served folders
func TestPathTraversalIsBlocked(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	env, err := os.ReadFile(".env")
	require.NoError(t, err)
	secret := strings.SplitN(strings.TrimSpace(string(env)), "\n", 2)[0]

	for _, path := range []string{
		"/img/..%2f.env", "/img/%2e%2e/.env", "/img/thumbs/..%2f..%2f.env", "/img/thumbs/%2e%2e%2f%2e%2e%2fgo.mod",
		"/public/..%2f.env", "/public/%2e%2e/main.go", "/api/pdf/..%2f..%2f.env/", "/img/../.env", "/img/",
	} {
		req := httptest.NewRequest("GET", "/", nil)
		req.URL.RawPath = path
		req.URL.Path, _ = url.PathUnescape(path)
		res := utils.SubmitRequest(req, r)
		require.NotEqual(t, http.StatusOK, res.Code, path)
		require.NotContains(t, res.Body.String(), secret, path)
	}
}

// TestUploadFileNamesCannotLeaveTheImageFolder uploads an item image with a traversing name
func TestUploadFileNamesCannotLeaveTheImageFolder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	for k, v := range map[string]string{"Name": "Threat upload", "Description": "Test item", "Price": "100", "Type": "normal_item"} {
		require.NoError(t, w.WriteField(k, v))
	}
	f, err := w.CreateFormFile("Image", "../../threat-escape.png")
	require.NoError(t, err)
	_, _ = f.Write([]byte("png"))
	require.NoError(t, w.Close())
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, w.FormDataContentType(), 200, adminUserToken)
	var id int
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &id))
	t.Cleanup(func() { removeItemFiles(t, id) })

	item, err := database.Db.GetItemIncludingDisabled(id)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(item.Image, "img/threat-escape"), item.Image)
	require.NotContains(t, item.Image, "..")
	_, err = os.Stat("../threat-escape.png")
	require.True(t, os.IsNotExist(err))
}

// TestAbuseIsLimited checks the limits for request floods, oversized bodies and scanners
func TestAbuseIsLimited(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	flood := func(remote string, n int) int {
		code := 0
		for i := 0; i < n; i++ {
			req := httptest.NewRequest("GET", "/api/hello/", nil)
			req.RemoteAddr = remote
			code = utils.SubmitRequest(req, r).Code
		}
		return code
	}
	require.Equal(t, http.StatusTooManyRequests, flood("198.51.100.200:1234", 501), "more than 500 requests a minute")
	require.Equal(t, http.StatusOK, flood("198.51.100.201:1234", 1), "other clients are not affected")

	big := `{"Entries":[],"VendorLicenseID":"` + strings.Repeat("x", 2<<20) + `"}`
	require.Equal(t, http.StatusBadRequest, call("POST", "/api/orders/", "", nil).Code)
	req := httptest.NewRequest("POST", "/api/orders/", strings.NewReader(big))
	require.Equal(t, http.StatusBadRequest, utils.SubmitRequest(req, r).Code, "body over one megabyte")

	// A scanner probing for secrets is blocked, also for its following requests
	scanner := "198.51.100.202:1234"
	req = httptest.NewRequest("GET", "/.env", nil)
	req.RemoteAddr = scanner
	require.NotEqual(t, http.StatusOK, utils.SubmitRequest(req, r).Code)
	req = httptest.NewRequest("GET", "/api/hello/", nil)
	req.RemoteAddr = scanner
	require.Equal(t, http.StatusForbidden, utils.SubmitRequest(req, r).Code)

	// So is a script in the query string
	req = httptest.NewRequest("GET", "/api/items/?q=%3Cscript%3Ealert(1)%3C/script%3E", nil)
	req.RemoteAddr = "198.51.100.203:1234"
	require.Equal(t, http.StatusForbidden, utils.SubmitRequest(req, r).Code)
}


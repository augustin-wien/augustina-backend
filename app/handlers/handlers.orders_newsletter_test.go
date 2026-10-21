package handlers

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// createTestOrder posts an order and returns it as stored in the database
func createTestOrder(t *testing.T, req createOrderRequest) database.Order {
	res := utils.TestRequest(t, r, "POST", "/api/orders/", req, 200)
	var response createOrderResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	u, err := url.Parse(response.SmartCheckoutURL)
	require.NoError(t, err)
	order, err := database.Db.GetOrderByOrderCode(u.Query().Get("s"))
	require.NoError(t, err)
	return order
}

func TestOrderNewsletterOptIn(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testnewsletter"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)

	customerEmail := "newsletter_optin@example.com"
	_ = keycloak.KeycloakClient.DeleteUser(customerEmail)
	defer keycloak.KeycloakClient.DeleteUser(customerEmail)

	itemWithLicense, _ := CreateTestItemWithLicense(t)
	itemWithLicenseID, _ := strconv.Atoi(itemWithLicense)
	normalItemID, _ := strconv.Atoi(CreateTestItem(t, "Newsletter normal item", 20, "", ""))

	// The opt-in is only kept for digital items, which require a customer email
	normalOrder := createTestOrder(t, createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: normalItemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
		Newsletter:      true,
	})
	require.False(t, normalOrder.Newsletter)

	withoutOptIn := createTestOrder(t, createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemWithLicenseID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
	})
	require.False(t, withoutOptIn.Newsletter)

	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(withoutOptIn.ID, 48))
	customer, err := database.Db.GetCustomerByEmail(customerEmail)
	require.NoError(t, err)
	require.False(t, customer.Newsletter)

	withOptIn := createTestOrder(t, createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemWithLicenseID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
		Newsletter:      true,
	})
	require.True(t, withOptIn.Newsletter)

	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(withOptIn.ID, 48))
	customer, err = database.Db.GetCustomerByEmail(customerEmail)
	require.NoError(t, err)
	require.True(t, customer.Newsletter)
	require.NotNil(t, customer.NewsletterAt)

	// The backoffice customer list shows the subscription
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/customers/", nil, 200, adminUserToken)
	var customers []database.Customer
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &customers))
	require.Len(t, customers, 1)
	require.True(t, customers[0].Newsletter)
}

// A PDF purchase creates no customer record of its own, the opt-in still has to show up
func TestOrderNewsletterOptInPDF(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testnewsletterpdf"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)

	origBuild := database.BuildEmailRequestFromTemplate
	origSend := mailer.Send
	defer func() {
		database.BuildEmailRequestFromTemplate = origBuild
		mailer.Send = origSend
	}()
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		return mailer.NewRequest(to, name, "body")
	}
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) { return true, nil }

	pdfID, err := database.Db.CreatePDF(database.PDF{Path: "testfile.pdf", Timestamp: time.Now()})
	require.NoError(t, err)
	licenseID, err := database.Db.CreateItem(database.Item{Name: "newsletter-license-pdf", Description: "license pdf", Price: 1, IsLicenseItem: true, IsPDFItem: true, PDF: null.IntFrom(pdfID), Type: "license_item"})
	require.NoError(t, err)
	itemID, err := database.Db.CreateItem(database.Item{Name: "newsletter-item-pdf", Description: "main item", Price: 200, LicenseItem: null.IntFrom(int64(licenseID)), Type: "issue"})
	require.NoError(t, err)

	customerEmail := "newsletter_pdf@example.com"
	order := createTestOrder(t, createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom(customerEmail),
		Newsletter:      true,
	})
	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(order.ID, 48))

	customer, err := database.Db.GetCustomerByEmail(customerEmail)
	require.NoError(t, err)
	require.True(t, customer.Newsletter)
	require.Empty(t, customer.KeycloakID)
}

func TestListOrderStatistics(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testorderstatistics"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)
	itemID, _ := strconv.Atoi(CreateTestItem(t, "Statistics item", 20, "", ""))

	var orders []database.Order
	for range 3 {
		orders = append(orders, createTestOrder(t, createOrderRequest{
			Entries:         []createOrderRequestEntry{{Item: itemID, Quantity: 1}},
			VendorLicenseID: vendorLicenseID,
		}))
	}
	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orders[0].ID, 48))
	require.NoError(t, database.Db.InvalidateOrder(orders[1].ID))

	res := utils.TestRequestWithAuth(t, r, "GET", "/api/orders/statistics/", nil, 200, adminUserToken)
	var stats database.OrderStatistics
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &stats))
	require.Equal(t, database.OrderStatistics{Started: 3, Completed: 1, Abandoned: 1, Open: 1}, stats)

	from := url.QueryEscape(time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	res = utils.TestRequestWithAuth(t, r, "GET", "/api/orders/statistics/?from="+from, nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &stats))
	require.Equal(t, database.OrderStatistics{}, stats)

	utils.TestRequest(t, r, "GET", "/api/orders/statistics/", nil, 401)
}

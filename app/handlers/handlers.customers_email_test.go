package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// TestCorrectCustomerEmail fixes a mistyped address of a customer without a
// Keycloak login: the orders follow the new address, the password reset mail
// creates the login, and the order mail can be resent to the new address
func TestCorrectCustomerEmail(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	require.NoError(t, database.Db.CreateOrUpdateMailTemplate("digitalLicenceItemTemplate.html", "Digital", `<a href="{{.URL}}">read</a>`))
	require.NoError(t, database.Db.BackfillOnlinePaperUrl("https://paper.test"))
	origSMTPSender := config.Config.SMTPSenderAddress
	config.Config.SMTPSenderAddress = "noreply@example.com"
	defer func() { config.Config.SMTPSenderAddress = origSMTPSender }()

	origSend := mailer.Send
	defer func() { mailer.Send = origSend }()
	var sentTo [][]string
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		sentTo = append(sentTo, r.To())
		return true, nil
	}

	suffix := time.Now().UnixNano()
	wrongEmail := fmt.Sprintf("wrong_%d@exmaple.com", suffix)
	rightEmail := fmt.Sprintf("right_%d@example.com", suffix)
	_ = keycloak.KeycloakClient.DeleteUser(rightEmail)
	defer func() { _ = keycloak.KeycloakClient.DeleteUser(rightEmail) }()

	vendorLicenseID := "testcorrectcustomeremail"
	createTestVendor(t, vendorLicenseID)
	vendor, err := database.Db.GetVendorByLicenseID(vendorLicenseID)
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendor.ID)
	require.NoError(t, err)
	buyerAccountID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	licenseID, err := database.Db.CreateItem(database.Item{Name: "correct-license", Description: "correct-license", Price: 1, IsLicenseItem: true, Type: "license_item"})
	require.NoError(t, err)
	digitalID, err := database.Db.CreateItem(database.Item{Name: "correct-digital", Description: "correct-digital", Price: 300, LicenseItem: null.IntFrom(int64(licenseID)), Type: "online_issue"})
	require.NoError(t, err)
	orderID, err := database.Db.CreateOrder(database.Order{
		Vendor: vendor.ID, Verified: true, CustomerEmail: null.StringFrom(wrongEmail),
		Entries: []database.OrderEntry{{Item: digitalID, Quantity: 1, Price: 300, Sender: buyerAccountID, Receiver: vendorAccount.ID, IsSale: true}},
	})
	require.NoError(t, err)
	unverifiedID, err := database.Db.CreateOrder(database.Order{
		Vendor: vendor.ID, CustomerEmail: null.StringFrom(wrongEmail),
		Entries: []database.OrderEntry{{Item: digitalID, Quantity: 1, Price: 300, Sender: buyerAccountID, Receiver: vendorAccount.ID, IsSale: true}},
	})
	require.NoError(t, err)

	customer, err := database.Db.CreateCustomer(&database.Customer{Email: wrongEmail, LicenseGroups: []string{}})
	require.NoError(t, err)
	customerURL := "/api/customers/" + strconv.Itoa(customer.ID) + "/"

	listCustomer := func() database.Customer {
		res := utils.TestRequestWithAuth(t, r, "GET", "/api/customers/", nil, 200, adminUserToken)
		var customers []database.Customer
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &customers))
		require.Len(t, customers, 1)
		return customers[0]
	}
	listed := listCustomer()
	require.Equal(t, 1, listed.PurchaseCount, "only the verified order counts")
	require.NotNil(t, listed.LastPurchaseAt)

	utils.TestRequestStrWithAuth(t, r, "PUT", customerURL, `{"email": "`+rightEmail+`", "licensegroups": []}`, 200, adminUserToken)

	for _, id := range []int{orderID, unverifiedID} {
		order, err := database.Db.GetOrderByID(id)
		require.NoError(t, err)
		require.Equal(t, rightEmail, order.CustomerEmail.String)
	}
	require.Equal(t, 1, listCustomer().PurchaseCount)

	utils.TestRequestWithAuth(t, r, "POST", customerURL+"password-reset-email/", nil, 204, adminUserToken)
	updated, err := database.Db.GetCustomerByID(customer.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updated.KeycloakID)
	requireKeycloakEmail(t, updated.KeycloakID, rightEmail)

	// Sending again reuses the login
	utils.TestRequestWithAuth(t, r, "POST", customerURL+"verify-email/", nil, 204, adminUserToken)
	again, err := database.Db.GetCustomerByID(customer.ID)
	require.NoError(t, err)
	require.Equal(t, updated.KeycloakID, again.KeycloakID)

	utils.TestRequestWithAuth(t, r, "POST", "/api/orders/"+strconv.Itoa(orderID)+"/resend-mail/", nil, 200, adminUserToken)
	require.Equal(t, [][]string{{rightEmail}}, sentTo)
}

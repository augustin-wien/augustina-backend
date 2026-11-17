package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

type handlerFakeVoucherAccounts struct{}

func (handlerFakeVoucherAccounts) GetOrCreateUser(email string) (string, bool, error) {
	return "kc-" + email, false, nil
}
func (handlerFakeVoucherAccounts) AssignGroup(string, string) error { return nil }
func (handlerFakeVoucherAccounts) AssignDigitalLicenseGroup(string, string) error {
	return nil
}

func stubHandlerVoucherDeps(t *testing.T) {
	origBuild, origSend, origAccounts := database.BuildEmailRequestFromTemplate, database.SendVoucherMail, database.VoucherAccounts
	t.Cleanup(func() {
		database.BuildEmailRequestFromTemplate, database.SendVoucherMail, database.VoucherAccounts = origBuild, origSend, origAccounts
	})
	database.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		return mailer.NewRequest(to, name, "<p>"+name+"</p>")
	}
	database.SendVoucherMail = func(*mailer.EmailRequest) {}
	database.VoucherAccounts = handlerFakeVoucherAccounts{}
}

func TestCreateOrderGiftTexts(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testgifttexts"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)
	itemID, _ := strconv.Atoi(CreateTestItem(t, "Gift text item", 20, "", ""))

	res := utils.TestRequest(t, r, "POST", "/api/orders/", createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		GiftMessage:     strings.Repeat("x", database.MaxVoucherMessageLength+1),
	}, 400)
	require.Contains(t, res.Body.String(), "gift message")

	order := createTestOrder(t, createOrderRequest{
		Entries:           []createOrderRequestEntry{{Item: itemID, Quantity: 1}},
		VendorLicenseID:   vendorLicenseID,
		GiftRecipientName: "  Grete ",
		GiftMessage:       " Alles Gute! ",
	})
	require.Equal(t, "Grete", order.GiftRecipientName)
	require.Equal(t, "Alles Gute!", order.GiftMessage)
}

func TestVoucherCheckAndRedeem(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	stubHandlerVoucherDeps(t)
	vendorLicenseID := "testvoucherredeem"
	vendorID, err := strconv.Atoi(createTestVendor(t, vendorLicenseID))
	require.NoError(t, err)

	itemID, _, err := database.Db.CreateAbonementVoucherItemWithLicense(database.Item{
		Name:              "Abo Gutschein",
		Description:       "Online-Abo zum Verschenken",
		Price:             3000,
		LicenseGroup:      null.StringFrom("digital_abo"),
		AboDurationMonths: 6,
	}, 1000)
	require.NoError(t, err)
	_, err = database.Db.EntClient.Item.UpdateOneID(itemID).SetDisabled(false).Save(context.Background())
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	orderID, err := database.Db.CreateOrder(database.Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom("buyer@example.com"),
		Entries:       []database.OrderEntry{{Item: itemID, Quantity: 1, Sender: vendorAccount.ID, Receiver: vendorAccount.ID, IsSale: true}},
	})
	require.NoError(t, err)
	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := database.Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)
	require.Len(t, vouchers, 1)
	code := vouchers[0].Code

	utils.TestRequest(t, r, "POST", "/api/vouchers/check/", voucherCodeRequest{Code: "AAAA-BBBB-CCCC"}, 404)

	res := utils.TestRequest(t, r, "POST", "/api/vouchers/check/", voucherCodeRequest{Code: strings.ToLower(code)}, 200)
	var check VoucherCheckResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &check))
	require.Equal(t, database.VoucherStatusActive, check.Status)
	require.Equal(t, 6, check.DurationMonths)

	utils.TestRequest(t, r, "POST", "/api/vouchers/redeem/", redeemVoucherRequest{Code: code, Email: "not-an-email"}, 400)

	res = utils.TestRequest(t, r, "POST", "/api/vouchers/redeem/", redeemVoucherRequest{Code: code, Email: "reader@example.com"}, 200)
	var redeemed RedeemVoucherResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &redeemed))
	require.Equal(t, "reader@example.com", redeemed.Email)
	require.Equal(t, 6, redeemed.DurationMonths)
	require.NotContains(t, res.Body.String(), "http", "login links are only sent by mail")

	utils.TestRequest(t, r, "POST", "/api/vouchers/redeem/", redeemVoucherRequest{Code: code, Email: "reader@example.com"}, 409)
}

func TestVoucherAdminRoutesRequireAuth(t *testing.T) {
	utils.TestRequest(t, r, "GET", "/api/vouchers/", nil, 401)
	utils.TestRequest(t, r, "POST", "/api/vouchers/1/disable/", nil, 401)
}

func setVoucherModule(t *testing.T, abonements, vouchers bool) {
	_, err := database.Db.EntClient.Settings.Update().
		SetAbonementEnabled(abonements).
		SetVoucherEnabled(vouchers).
		Save(context.Background())
	require.NoError(t, err)
}

func TestVoucherModuleControlsSale(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	vendorLicenseID := "testvouchermodule"
	createTestVendor(t, vendorLicenseID)
	setMaxOrderAmount(t, 100000)

	itemID, _, err := database.Db.CreateAbonementVoucherItemWithLicense(database.Item{
		Name:         "Modul Gutschein",
		Description:  "Online-Abo zum Verschenken",
		Price:        3000,
		LicenseGroup: null.StringFrom("digital_abo"),
	}, 1000)
	require.NoError(t, err)
	_, err = database.Db.EntClient.Item.UpdateOneID(itemID).SetDisabled(false).Save(context.Background())
	require.NoError(t, err)

	order := createOrderRequest{
		Entries:         []createOrderRequestEntry{{Item: itemID, Quantity: 1}},
		VendorLicenseID: vendorLicenseID,
		CustomerEmail:   null.StringFrom("buyer@example.com"),
	}
	shopHasVoucher := func() bool {
		items, err := database.Db.ListItemsShop()
		require.NoError(t, err)
		for _, it := range items {
			if it.ID == itemID {
				return true
			}
		}
		return false
	}

	for _, c := range []struct{ abonements, vouchers bool }{{false, false}, {false, true}, {true, false}} {
		setVoucherModule(t, c.abonements, c.vouchers)
		require.False(t, shopHasVoucher(), "abonements=%v vouchers=%v", c.abonements, c.vouchers)
		res := utils.TestRequest(t, r, "POST", "/api/orders/", order, 400)
		require.Contains(t, res.Body.String(), "abonement vouchers are not available")
	}

	setVoucherModule(t, true, true)
	require.True(t, shopHasVoucher())
	utils.TestRequest(t, r, "POST", "/api/orders/", order, 200)
}

// The limit must apply per client behind the proxy, not to all customers together
func TestVoucherRateLimitIsPerClient(t *testing.T) {
	check := func(ip string) int {
		req, err := http.NewRequest("POST", "/api/vouchers/check/", strings.NewReader(`{"Code":"AAAA-BBBB-CCCC"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-Ip", ip)
		return utils.SubmitRequest(req, r).Code
	}
	for i := 0; i < 10; i++ {
		require.Equal(t, http.StatusNotFound, check("203.0.113.10"))
	}
	require.Equal(t, http.StatusTooManyRequests, check("203.0.113.10"))
	require.Equal(t, http.StatusNotFound, check("203.0.113.11"), "another client is not affected")
}

// createHandlerTestVoucher sells one voucher and returns it
func createHandlerTestVoucher(t *testing.T, vendorLicenseID string) database.AbonementVoucher {
	vendorID, err := strconv.Atoi(createTestVendor(t, vendorLicenseID))
	require.NoError(t, err)
	itemID, _, err := database.Db.CreateAbonementVoucherItemWithLicense(database.Item{
		Name:         "Admin Gutschein",
		Description:  "Online-Abo zum Verschenken",
		Price:        3000,
		LicenseGroup: null.StringFrom("digital_abo"),
	}, 1000)
	require.NoError(t, err)
	_, err = database.Db.EntClient.Item.UpdateOneID(itemID).SetDisabled(false).Save(context.Background())
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	orderID, err := database.Db.CreateOrder(database.Order{
		Vendor:        vendorID,
		CustomerEmail: null.StringFrom("buyer@example.com"),
		Entries:       []database.OrderEntry{{Item: itemID, Quantity: 1, Sender: vendorAccount.ID, Receiver: vendorAccount.ID, IsSale: true}},
	})
	require.NoError(t, err)
	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := database.Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)
	require.Len(t, vouchers, 1)
	return vouchers[0]
}

func TestVoucherAdminEndpoints(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	stubHandlerVoucherDeps(t)
	var sentTo []string
	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	mailer.Send = func(m *mailer.EmailRequest) (bool, error) {
		sentTo = append(sentTo, m.To()...)
		require.Len(t, m.Attachments(), 1)
		return true, nil
	}

	v := createHandlerTestVoucher(t, "testvoucheradmin")
	base := "/api/vouchers/" + strconv.Itoa(v.ID)

	// List
	res := utils.TestRequestWithAuth(t, r, "GET", "/api/vouchers/", nil, 200, adminUserToken)
	var list []database.AbonementVoucher
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &list))
	require.Len(t, list, 1)
	require.Equal(t, v.Code, list[0].Code)
	require.Equal(t, "Admin Gutschein", list[0].ItemName)

	// PDF
	res = utils.TestRequestWithAuth(t, r, "GET", base+"/pdf/", nil, 200, adminUserToken)
	require.Equal(t, "application/pdf", res.Header().Get("Content-Type"))
	require.True(t, strings.HasPrefix(res.Body.String(), "%PDF"))
	utils.TestRequestWithAuth(t, r, "GET", "/api/vouchers/999999/pdf/", nil, 404, adminUserToken)
	utils.TestRequestWithAuth(t, r, "GET", "/api/vouchers/abc/pdf/", nil, 400, adminUserToken)

	// Resend to the buyer and to another address
	utils.TestRequestWithAuth(t, r, "POST", base+"/resend/", nil, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", base+"/resend/", resendVoucherRequest{Email: "gift@example.com"}, 200, adminUserToken)
	utils.TestRequestWithAuth(t, r, "POST", base+"/resend/", resendVoucherRequest{Email: "nope"}, 400, adminUserToken)
	require.Equal(t, []string{"buyer@example.com", "gift@example.com"}, sentTo)

	// Disable blocks redemption, enable allows it again
	res = utils.TestRequestWithAuth(t, r, "POST", base+"/disable/", nil, 200, adminUserToken)
	require.Contains(t, res.Body.String(), database.VoucherStatusDisabled)
	utils.TestRequest(t, r, "POST", "/api/vouchers/redeem/", redeemVoucherRequest{Code: v.Code, Email: "reader@example.com"}, 409)
	utils.TestRequestWithAuth(t, r, "POST", base+"/enable/", nil, 200, adminUserToken)
	utils.TestRequest(t, r, "POST", "/api/vouchers/redeem/", redeemVoucherRequest{Code: v.Code, Email: "reader@example.com"}, 200)

	// A redeemed voucher can't be disabled
	utils.TestRequestWithAuth(t, r, "POST", base+"/disable/", nil, 409, adminUserToken)
}

func voucherItemForm(t *testing.T, fields map[string]string) (*bytes.Buffer, string) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		require.NoError(t, writer.WriteField(k, v))
	}
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

func TestCreateAndUpdateVoucherItem(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })

	valid := map[string]string{
		"Name":              "API Gutschein",
		"Description":       "Online-Abo zum Verschenken",
		"Price":             "3000",
		"Type":              database.AbonementVoucherItemType,
		"LicenseGroup":      "digital_abo",
		"license_cost":      "1000",
		"AboDurationMonths": "6",
	}

	invalid := map[string]string{}
	for k, v := range valid {
		invalid[k] = v
	}
	invalid["AboDurationMonths"] = "121"
	body, ct := voucherItemForm(t, invalid)
	utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, ct, 400, adminUserToken)

	invalid["AboDurationMonths"] = "6"
	invalid["LicenseGroup"] = ""
	body, ct = voucherItemForm(t, invalid)
	utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, ct, 400, adminUserToken)

	body, ct = voucherItemForm(t, valid)
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, ct, 200, adminUserToken)
	itemID, err := strconv.Atoi(strings.TrimSpace(res.Body.String()))
	require.NoError(t, err)

	item, err := database.Db.GetItemIncludingDisabled(itemID)
	require.NoError(t, err)
	require.Equal(t, database.AbonementVoucherItemType, item.Type)
	require.Equal(t, 6, item.AboDurationMonths)
	require.True(t, item.LicenseItem.Valid, "a voucher item gets a license item like an abonement")

	update := map[string]string{}
	for k, v := range valid {
		update[k] = v
	}
	delete(update, "license_cost")
	update["AboDurationMonths"] = "24"
	update["LicenseItem"] = strconv.Itoa(int(item.LicenseItem.Int64))
	body, ct = voucherItemForm(t, update)
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/items/"+strconv.Itoa(itemID)+"/", body, ct, 200, adminUserToken)
	item, err = database.Db.GetItemIncludingDisabled(itemID)
	require.NoError(t, err)
	require.Equal(t, 24, item.AboDurationMonths)
}

func TestResendOrderMailForVoucherOrder(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())
	t.Cleanup(func() { _ = database.Db.InitEmptyTestDb() })
	stubHandlerVoucherDeps(t)
	var sent []*mailer.EmailRequest
	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	mailer.Send = func(m *mailer.EmailRequest) (bool, error) {
		sent = append(sent, m)
		return true, nil
	}

	v := createHandlerTestVoucher(t, "testvoucherresendorder")
	utils.TestRequestWithAuth(t, r, "POST", "/api/orders/"+strconv.Itoa(int(v.OrderID.Int64))+"/resend-mail/", nil, 200, adminUserToken)

	require.Len(t, sent, 1, "only the voucher mail, no online paper mail")
	require.Equal(t, []string{"buyer@example.com"}, sent[0].To())
	require.Len(t, sent[0].Attachments(), 1)
}

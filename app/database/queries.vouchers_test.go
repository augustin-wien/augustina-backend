package database

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

type fakeVoucherAccounts struct {
	mu            sync.Mutex
	users         map[string]string
	licenseGroups map[string][]string
}

func newFakeVoucherAccounts() *fakeVoucherAccounts {
	return &fakeVoucherAccounts{users: map[string]string{}, licenseGroups: map[string][]string{}}
}

func (f *fakeVoucherAccounts) GetOrCreateUser(email string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.users[email]; ok {
		return id, false, nil
	}
	id := "kc-" + email
	f.users[email] = id
	return id, true, nil
}

func (f *fakeVoucherAccounts) AssignGroup(string, string) error { return nil }

func (f *fakeVoucherAccounts) AssignDigitalLicenseGroup(userID, group string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.licenseGroups[userID] = append(f.licenseGroups[userID], group)
	return nil
}

type sentMail struct {
	template string
	to       []string
	data     interface{}
}

// stubVoucherMails records built mails instead of rendering DB templates and sending them
func stubVoucherMails(t *testing.T) (*[]sentMail, *[]*mailer.EmailRequest) {
	var mu sync.Mutex
	built := []sentMail{}
	sent := []*mailer.EmailRequest{}
	origBuild, origSend, origAccounts := BuildEmailRequestFromTemplate, SendVoucherMail, VoucherAccounts
	t.Cleanup(func() {
		BuildEmailRequestFromTemplate, SendVoucherMail, VoucherAccounts = origBuild, origSend, origAccounts
	})
	BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		mu.Lock()
		defer mu.Unlock()
		built = append(built, sentMail{template: name, to: to, data: data})
		return mailer.NewRequest(to, name, "<p>"+name+"</p>")
	}
	SendVoucherMail = func(m *mailer.EmailRequest) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, m)
	}
	return &built, &sent
}

func createVoucherTestItem(t *testing.T, duration int) Item {
	item := Item{
		Name:              "Online-Abo Gutschein",
		Description:       "Ein Online-Abo zum Verschenken",
		Price:             4000,
		LicenseGroup:      null.StringFrom("digital_abo"),
		AboDurationMonths: duration,
	}
	id, _, err := Db.CreateAbonementVoucherItemWithLicense(item, 1000)
	utils.CheckError(t, err)
	created, err := Db.voucherItem(id)
	utils.CheckError(t, err)
	return created
}

func createVoucherTestOrder(t *testing.T, item Item, quantity int) int {
	vendorID, err := Db.CreateVendor(Vendor{
		FirstName: "Test",
		LastName:  "Vendor",
		Email:     "voucher-vendor@example.com",
		LicenseID: null.StringFrom("tv-voucher"),
	})
	utils.CheckError(t, err)
	vendorAccount, err := Db.GetAccountByVendorID(vendorID)
	utils.CheckError(t, err)

	// Items are created disabled; orders need them enabled
	_, err = Db.EntClient.Item.UpdateOneID(item.ID).SetDisabled(false).Save(context.Background())
	utils.CheckError(t, err)

	orderID, err := Db.CreateOrder(Order{
		Vendor:            vendorID,
		CustomerEmail:     null.StringFrom("buyer@example.com"),
		GiftRecipientName: "Grete",
		GiftMessage:       "Alles Gute!",
		Entries: []OrderEntry{{
			Item:     item.ID,
			Quantity: quantity,
			Sender:   vendorAccount.ID,
			Receiver: vendorAccount.ID,
			IsSale:   true,
		}},
	})
	utils.CheckError(t, err)
	return orderID
}

func TestGenerateAndNormalizeVoucherCode(t *testing.T) {
	code, err := GenerateVoucherCode()
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^[A-HJKMNP-Z2-9]{4}-[A-HJKMNP-Z2-9]{4}-[A-HJKMNP-Z2-9]{4}$`), code)

	require.Equal(t, "ABCD-EFGH-JKMN", NormalizeVoucherCode(" abcd efgh-jkmn "))
	require.Equal(t, "ABCD-EFGH-JKMN", NormalizeVoucherCode("ABCDEFGHJKMN"))
	require.Equal(t, "ABC", NormalizeVoucherCode("a-b-c"))
}

func TestVerifyOrderCreatesVouchersAndMailsPDF(t *testing.T) {
	Db.InitEmptyTestDb()
	built, sent := stubVoucherMails(t)
	accounts := newFakeVoucherAccounts()
	VoucherAccounts = accounts

	item := createVoucherTestItem(t, 6)
	orderID := createVoucherTestOrder(t, item, 2)

	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))

	vouchers, err := Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)
	require.Len(t, vouchers, 2)
	require.NotEqual(t, vouchers[0].Code, vouchers[1].Code)
	for _, v := range vouchers {
		require.Equal(t, VoucherStatusActive, v.Status)
		require.Equal(t, 6, v.DurationMonths)
		require.Equal(t, "Grete", v.RecipientName)
		require.Equal(t, "Alles Gute!", v.Message)
		require.Equal(t, "buyer@example.com", v.BuyerEmail)
		require.Equal(t, item.Name, v.ItemName)
	}

	// The buyer gets the voucher mail with one PDF, but no account, abonement or online paper mail
	require.Len(t, *sent, 1)
	attachments := (*sent)[0].Attachments()
	require.Len(t, attachments, 1)
	require.Equal(t, "application/pdf", attachments[0].ContentType)
	require.Equal(t, "%PDF", string(attachments[0].Data[:4]))
	for _, m := range *built {
		require.Equal(t, "abonementVoucher", m.template)
	}
	_, err = Db.GetCustomerByEmail("buyer@example.com")
	require.Error(t, err, "buying a voucher must not create a customer for the buyer")

	// Verifying again creates no further vouchers
	_, err = Db.EntClient.Order.UpdateOneID(orderID).SetVerified(false).Save(context.Background())
	require.NoError(t, err)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err = Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)
	require.Len(t, vouchers, 2)
}

func TestRedeemVoucher(t *testing.T) {
	Db.InitEmptyTestDb()
	built, _ := stubVoucherMails(t)
	accounts := newFakeVoucherAccounts()
	VoucherAccounts = accounts

	item := createVoucherTestItem(t, 6)
	orderID := createVoucherTestOrder(t, item, 1)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)
	code := vouchers[0].Code

	// Typed in lower case without dashes
	typed := NormalizeVoucherCode(code)
	typed = regexp.MustCompile("-").ReplaceAllString(typed, " ")

	result, err := Db.RedeemVoucher(RedeemVoucherRequest{Code: typed, Email: " Grete@Example.com ", FirstName: "Grete", LastName: "Muster"})
	require.NoError(t, err)
	require.Equal(t, 6, result.DurationMonths)
	require.True(t, result.NewUser)
	require.WithinDuration(t, time.Now(), result.FromDate, 5*time.Second)
	require.WithinDuration(t, time.Now().AddDate(0, 6, 0), result.ToDate, 5*time.Second)

	customer, err := Db.GetCustomerByEmail("grete@example.com")
	require.NoError(t, err)
	require.Equal(t, "Grete", customer.FirstName)
	require.Contains(t, customer.LicenseGroups, "digital_abo")
	require.Contains(t, accounts.licenseGroups["kc-grete@example.com"], "digital_abo")

	abos, err := Db.ListAbonementsByCustomer(customer.ID)
	require.NoError(t, err)
	require.Len(t, abos, 1)
	require.Equal(t, item.ID, abos[0].ItemID)

	v, err := Db.GetVoucherByCode(code)
	require.NoError(t, err)
	require.Equal(t, VoucherStatusRedeemed, v.Status)
	require.Equal(t, "grete@example.com", v.RedeemedEmail.String)
	require.Equal(t, int64(abos[0].ID), v.AbonementID.Int64)

	templates := map[string]bool{}
	for _, m := range *built {
		templates[m.template] = true
	}
	require.True(t, templates["abonementConfirmation"])
	require.True(t, templates["welcome"])

	// A second redemption fails and creates no abonement
	_, err = Db.RedeemVoucher(RedeemVoucherRequest{Code: code, Email: "other@example.com"})
	require.ErrorIs(t, err, ErrVoucherRedeemed)
	_, err = Db.GetCustomerByEmail("other@example.com")
	require.Error(t, err, "a redeemed code must not create an account")

	_, err = Db.RedeemVoucher(RedeemVoucherRequest{Code: "AAAA-BBBB-CCCC", Email: "x@example.com"})
	require.ErrorIs(t, err, ErrVoucherNotFound)
}

func TestRedeemVoucherExtendsActiveAbonement(t *testing.T) {
	Db.InitEmptyTestDb()
	stubVoucherMails(t)
	accounts := newFakeVoucherAccounts()
	accounts.users["reader@example.com"] = "kc-reader@example.com"
	VoucherAccounts = accounts

	item := createVoucherTestItem(t, 12)
	orderID := createVoucherTestOrder(t, item, 1)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)

	customer, err := Db.GetOrCreateCustomerByEmail("reader@example.com", "kc-reader@example.com")
	require.NoError(t, err)
	end := time.Now().AddDate(0, 3, 0).UTC().Truncate(time.Second)
	_, err = Db.CreateAbonement(&Abonement{CustomerID: customer.ID, ItemID: item.ID, FromDate: time.Now().AddDate(0, -9, 0), ToDate: end, Status: "active"})
	require.NoError(t, err)

	result, err := Db.RedeemVoucher(RedeemVoucherRequest{Code: vouchers[0].Code, Email: "reader@example.com"})
	require.NoError(t, err)
	require.False(t, result.NewUser)
	require.True(t, result.FromDate.Equal(end), "new abonement starts when the running one ends")
	require.True(t, result.ToDate.Equal(end.AddDate(1, 0, 0)))
}

func TestRedeemVoucherConcurrently(t *testing.T) {
	Db.InitEmptyTestDb()
	stubVoucherMails(t)
	VoucherAccounts = newFakeVoucherAccounts()

	item := createVoucherTestItem(t, 12)
	orderID := createVoucherTestOrder(t, item, 1)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)

	const attempts = 5
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := Db.RedeemVoucher(RedeemVoucherRequest{Code: vouchers[0].Code, Email: "same@example.com"})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	succeeded := 0
	for err := range errs {
		if err == nil {
			succeeded++
		}
	}
	require.Equal(t, 1, succeeded)

	customer, err := Db.GetCustomerByEmail("same@example.com")
	require.NoError(t, err)
	abos, err := Db.ListAbonementsByCustomer(customer.ID)
	require.NoError(t, err)
	require.Len(t, abos, 1)
}

func TestDisableVoucher(t *testing.T) {
	Db.InitEmptyTestDb()
	stubVoucherMails(t)
	VoucherAccounts = newFakeVoucherAccounts()

	item := createVoucherTestItem(t, 12)
	orderID := createVoucherTestOrder(t, item, 1)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)
	v := vouchers[0]

	disabled, err := Db.SetVoucherDisabled(v.ID, true)
	require.NoError(t, err)
	require.Equal(t, VoucherStatusDisabled, disabled.Status)

	_, err = Db.RedeemVoucher(RedeemVoucherRequest{Code: v.Code, Email: "a@example.com"})
	require.ErrorIs(t, err, ErrVoucherDisabled)

	enabled, err := Db.SetVoucherDisabled(v.ID, false)
	require.NoError(t, err)
	require.Equal(t, VoucherStatusActive, enabled.Status)

	_, err = Db.RedeemVoucher(RedeemVoucherRequest{Code: v.Code, Email: "a@example.com"})
	require.NoError(t, err)

	_, err = Db.SetVoucherDisabled(v.ID, true)
	require.ErrorIs(t, err, ErrVoucherRedeemed)
}

func TestResendOrderMailsSendsVoucherPDF(t *testing.T) {
	Db.InitEmptyTestDb()
	built, _ := stubVoucherMails(t)
	VoucherAccounts = newFakeVoucherAccounts()

	item := createVoucherTestItem(t, 12)
	orderID := createVoucherTestOrder(t, item, 1)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	*built = nil

	// ResendOrderMails sends synchronously over SMTP; only check the mail it builds
	mail, err := Db.orderVoucherMail(orderID, item.ID, "buyer@example.com")
	require.NoError(t, err)
	require.NotNil(t, mail)
	require.Len(t, mail.Attachments(), 1)
	require.Len(t, *built, 1)
	require.Equal(t, "abonementVoucher", (*built)[0].template)
}

func TestRedeemVoucherEscapesNameInMail(t *testing.T) {
	Db.InitEmptyTestDb()
	built, _ := stubVoucherMails(t)
	VoucherAccounts = newFakeVoucherAccounts()

	item := createVoucherTestItem(t, 12)
	orderID := createVoucherTestOrder(t, item, 1)
	require.NoError(t, Db.VerifyOrderAndCreatePayments(orderID, 1))
	vouchers, err := Db.GetVouchersByOrder(orderID)
	require.NoError(t, err)

	_, err = Db.RedeemVoucher(RedeemVoucherRequest{
		Code:      vouchers[0].Code,
		Email:     "inject@example.com",
		FirstName: `<a href="https://evil.example">Klick</a>`,
		LastName:  "Muster",
	})
	require.NoError(t, err)

	found := false
	for _, m := range *built {
		if m.template != "abonementConfirmation" {
			continue
		}
		found = true
		name := m.data.(map[string]interface{})["CustomerName"].(string)
		require.NotContains(t, name, "<a")
		require.Contains(t, name, "&lt;a href=")
	}
	require.True(t, found)
}

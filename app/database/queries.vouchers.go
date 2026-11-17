package database

import (
	"context"
	"crypto/rand"
	"errors"
	"html"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/ent"
	entabonement "github.com/augustin-wien/augustina-backend/ent/abonement"
	entvoucher "github.com/augustin-wien/augustina-backend/ent/abonementvoucher"
	entitem "github.com/augustin-wien/augustina-backend/ent/item"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/voucherpdf"
	"gopkg.in/guregu/null.v4"
)

// Voucher statuses
const (
	VoucherStatusActive   = "active"
	VoucherStatusRedeemed = "redeemed"
	VoucherStatusDisabled = "disabled"
)

// AbonementVoucherItemType is the item type of a gift voucher for an online abonement
const AbonementVoucherItemType = "abonement_voucher"

// Limits for the texts printed on a voucher PDF
const (
	MaxVoucherRecipientNameLength = 80
	MaxVoucherMessageLength       = 300
)

var (
	// ErrVoucherNotFound is returned when no voucher has the given code
	ErrVoucherNotFound = errors.New("voucher not found")
	// ErrVoucherRedeemed is returned when the voucher was already redeemed
	ErrVoucherRedeemed = errors.New("voucher already redeemed")
	// ErrVoucherDisabled is returned when the voucher was disabled by an admin
	ErrVoucherDisabled = errors.New("voucher disabled")
)

// voucherCodeAlphabet leaves out characters that are easily confused when typed off paper (0/O, 1/I/L)
const voucherCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const (
	voucherCodeGroups    = 3
	voucherCodeGroupSize = 4
)

// AbonementVoucher is a gift voucher that can be redeemed once for an online abonement
type AbonementVoucher struct {
	ID                 int
	Code               string
	ItemID             int
	ItemName           string
	OrderID            null.Int `swaggertype:"integer"`
	BuyerEmail         string
	RecipientName      string
	Message            string
	DurationMonths     int
	Status             string
	CreatedAt          time.Time
	RedeemedAt         null.Time `swaggertype:"string"`
	RedeemedEmail      null.String `swaggertype:"string"`
	RedeemedCustomerID null.Int    `swaggertype:"integer"`
	AbonementID        null.Int    `swaggertype:"integer"`
}

// VoucherAccountProvider creates the Keycloak account of the person redeeming a voucher
// and grants it access. Extracted as an interface so it can be replaced in tests.
type VoucherAccountProvider interface {
	GetOrCreateUser(email string) (userID string, newUser bool, err error)
	AssignGroup(userID string, groupName string) error
	AssignDigitalLicenseGroup(userID string, licenseGroup string) error
}

// VoucherAccounts is the account provider used when redeeming vouchers
var VoucherAccounts VoucherAccountProvider = &keycloak.KeycloakClient

// SendVoucherMail sends voucher related mails; replaced in tests
var SendVoucherMail = func(m *mailer.EmailRequest) {
	go func() {
		success, err := mailer.SendWithRetry(m)
		if err != nil || !success {
			log.Error("SendVoucherMail: failed to send mail: ", err)
		}
	}()
}

// VouchersOnSale reports whether the shop sells abonement vouchers. Vouchers already sold
// stay redeemable when the module is switched off, they were sold without an expiry date.
func VouchersOnSale(s *ent.Settings) bool {
	return s != nil && s.AbonementEnabled && s.VoucherEnabled
}

// GenerateVoucherCode returns a random code like "AB3D-EFGH-JK7M"
func GenerateVoucherCode() (string, error) {
	max := big.NewInt(int64(len(voucherCodeAlphabet)))
	var b strings.Builder
	for g := 0; g < voucherCodeGroups; g++ {
		if g > 0 {
			b.WriteByte('-')
		}
		for i := 0; i < voucherCodeGroupSize; i++ {
			n, err := rand.Int(rand.Reader, max)
			if err != nil {
				return "", err
			}
			b.WriteByte(voucherCodeAlphabet[n.Int64()])
		}
	}
	return b.String(), nil
}

// NormalizeVoucherCode turns user input into the stored form: upper case, grouped with dashes.
// Spaces, dashes and other separators in the input are ignored.
func NormalizeVoucherCode(input string) string {
	var raw strings.Builder
	for _, r := range strings.ToUpper(input) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			raw.WriteRune(r)
		}
	}
	s := raw.String()
	if len(s) != voucherCodeGroups*voucherCodeGroupSize {
		return s
	}
	parts := make([]string, 0, voucherCodeGroups)
	for i := 0; i < len(s); i += voucherCodeGroupSize {
		parts = append(parts, s[i:i+voucherCodeGroupSize])
	}
	return strings.Join(parts, "-")
}

// VoucherRedeemURL is the frontend page where vouchers are redeemed
func VoucherRedeemURL() string {
	return strings.TrimRight(config.Config.FrontendURL, "/") + "/einloesen"
}

// VoucherRedeemURLWithCode is the redeem page with the code filled in; the QR code links there
func VoucherRedeemURLWithCode(code string) string {
	return VoucherRedeemURL() + "?code=" + url.QueryEscape(code)
}

func convertEntVoucher(e *ent.AbonementVoucher) AbonementVoucher {
	v := AbonementVoucher{
		ID:             e.ID,
		Code:           e.Code,
		ItemID:         e.ItemID,
		BuyerEmail:     e.BuyerEmail,
		RecipientName:  e.RecipientName,
		Message:        e.Message,
		DurationMonths: e.DurationMonths,
		Status:         e.Status,
		CreatedAt:      e.CreatedAt,
	}
	if e.OrderID != nil {
		v.OrderID = null.IntFrom(int64(*e.OrderID))
	}
	if e.RedeemedAt != nil {
		v.RedeemedAt = null.TimeFrom(*e.RedeemedAt)
	}
	if e.RedeemedEmail != nil {
		v.RedeemedEmail = null.StringFrom(*e.RedeemedEmail)
	}
	if e.RedeemedCustomerID != nil {
		v.RedeemedCustomerID = null.IntFrom(int64(*e.RedeemedCustomerID))
	}
	if e.AbonementID != nil {
		v.AbonementID = null.IntFrom(int64(*e.AbonementID))
	}
	return v
}

// voucherItem returns the voucher's item, also when it has been disabled or archived since
func (db *Database) voucherItem(itemID int) (Item, error) {
	e, err := db.EntClient.Item.Query().Where(entitem.IDEQ(itemID)).WithLicenseItem().WithPDF().Only(context.Background())
	if err != nil {
		return Item{}, err
	}
	return convertEntItem(e), nil
}

func (db *Database) withItemNames(vouchers []AbonementVoucher) []AbonementVoucher {
	names := map[int]string{}
	for i := range vouchers {
		name, ok := names[vouchers[i].ItemID]
		if !ok {
			if item, err := db.voucherItem(vouchers[i].ItemID); err == nil {
				name = item.Name
			}
			names[vouchers[i].ItemID] = name
		}
		vouchers[i].ItemName = name
	}
	return vouchers
}

// createVouchersForOrderEntryTx creates the vouchers bought with an order entry. Calling it
// again for the same order and item creates only the ones still missing.
func createVouchersForOrderEntryTx(tx *ent.Tx, o Order, item Item, quantity int) ([]AbonementVoucher, error) {
	ctx := context.Background()
	existing, err := tx.AbonementVoucher.Query().
		Where(entvoucher.OrderIDEQ(o.ID), entvoucher.ItemIDEQ(item.ID)).
		Order(ent.Asc(entvoucher.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	duration := item.AboDurationMonths
	if duration <= 0 {
		duration = 12
	}
	for len(existing) < quantity {
		code, err := GenerateVoucherCode()
		if err != nil {
			return nil, err
		}
		taken, err := tx.AbonementVoucher.Query().Where(entvoucher.CodeEQ(code)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if taken {
			continue
		}
		created, err := tx.AbonementVoucher.Create().
			SetCode(code).
			SetItemID(item.ID).
			SetOrderID(o.ID).
			SetBuyerEmail(o.CustomerEmail.String).
			SetRecipientName(o.GiftRecipientName).
			SetMessage(o.GiftMessage).
			SetDurationMonths(duration).
			SetStatus(VoucherStatusActive).
			SetCreatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		existing = append(existing, created)
	}
	vouchers := make([]AbonementVoucher, len(existing))
	for i, e := range existing {
		vouchers[i] = convertEntVoucher(e)
		vouchers[i].ItemName = item.Name
	}
	return vouchers, nil
}

// GetVoucherByCode returns the voucher with the code; the code may be typed in any form NormalizeVoucherCode accepts
func (db *Database) GetVoucherByCode(code string) (AbonementVoucher, error) {
	e, err := db.EntClient.AbonementVoucher.Query().
		Where(entvoucher.CodeEQ(NormalizeVoucherCode(code))).
		Only(context.Background())
	if ent.IsNotFound(err) {
		return AbonementVoucher{}, ErrVoucherNotFound
	}
	if err != nil {
		return AbonementVoucher{}, err
	}
	return db.withItemNames([]AbonementVoucher{convertEntVoucher(e)})[0], nil
}

// GetVoucherByID returns the voucher with the ID
func (db *Database) GetVoucherByID(id int) (AbonementVoucher, error) {
	e, err := db.EntClient.AbonementVoucher.Get(context.Background(), id)
	if ent.IsNotFound(err) {
		return AbonementVoucher{}, ErrVoucherNotFound
	}
	if err != nil {
		return AbonementVoucher{}, err
	}
	return db.withItemNames([]AbonementVoucher{convertEntVoucher(e)})[0], nil
}

// GetVouchersByOrder returns the vouchers bought with an order
func (db *Database) GetVouchersByOrder(orderID int) ([]AbonementVoucher, error) {
	es, err := db.EntClient.AbonementVoucher.Query().
		Where(entvoucher.OrderIDEQ(orderID)).
		Order(ent.Asc(entvoucher.FieldID)).
		All(context.Background())
	if err != nil {
		return nil, err
	}
	vouchers := make([]AbonementVoucher, len(es))
	for i, e := range es {
		vouchers[i] = convertEntVoucher(e)
	}
	return db.withItemNames(vouchers), nil
}

// ListVouchers returns all vouchers, newest first
func (db *Database) ListVouchers() ([]AbonementVoucher, error) {
	es, err := db.EntClient.AbonementVoucher.Query().
		Order(ent.Desc(entvoucher.FieldID)).
		All(context.Background())
	if err != nil {
		return nil, err
	}
	vouchers := make([]AbonementVoucher, len(es))
	for i, e := range es {
		vouchers[i] = convertEntVoucher(e)
	}
	return db.withItemNames(vouchers), nil
}

// SetVoucherDisabled disables an active voucher or enables a disabled one. Redeemed vouchers can't be changed.
func (db *Database) SetVoucherDisabled(id int, disabled bool) (AbonementVoucher, error) {
	from, to := VoucherStatusActive, VoucherStatusDisabled
	if !disabled {
		from, to = VoucherStatusDisabled, VoucherStatusActive
	}
	n, err := db.EntClient.AbonementVoucher.Update().
		Where(entvoucher.IDEQ(id), entvoucher.StatusEQ(from)).
		SetStatus(to).
		Save(context.Background())
	if err != nil {
		return AbonementVoucher{}, err
	}
	v, err := db.GetVoucherByID(id)
	if err != nil {
		return v, err
	}
	if n == 0 && v.Status != to {
		if v.Status == VoucherStatusRedeemed {
			return v, ErrVoucherRedeemed
		}
		return v, errors.New("voucher status can't be changed")
	}
	return v, nil
}

// VoucherPDF renders the printable PDF of the vouchers
func (db *Database) VoucherPDF(vouchers []AbonementVoucher) ([]byte, error) {
	newspaperName := ""
	if settings, err := db.GetSettings(); err == nil && settings != nil {
		newspaperName = settings.NewspaperName
	}
	pages := make([]voucherpdf.Voucher, len(vouchers))
	for i, v := range vouchers {
		pages[i] = voucherpdf.Voucher{
			Code:           v.Code,
			ItemName:       v.ItemName,
			DurationMonths: v.DurationMonths,
			RecipientName:  v.RecipientName,
			Message:        v.Message,
			RedeemURL:      VoucherRedeemURL(),
			QRCodeURL:      VoucherRedeemURLWithCode(v.Code),
			NewspaperName:  newspaperName,
			LogoPath:       "img/logo.png",
		}
	}
	return voucherpdf.Generate(pages)
}

// BuildVoucherMail builds the mail that delivers the voucher PDF to the buyer
func (db *Database) BuildVoucherMail(email string, vouchers []AbonementVoucher) (*mailer.EmailRequest, error) {
	if len(vouchers) == 0 {
		return nil, errors.New("no vouchers")
	}
	pdf, err := db.VoucherPDF(vouchers)
	if err != nil {
		return nil, err
	}
	codes := make([]string, len(vouchers))
	for i, v := range vouchers {
		codes[i] = v.Code
	}
	data := map[string]interface{}{
		"ItemName":       vouchers[0].ItemName,
		"DurationMonths": vouchers[0].DurationMonths,
		"Count":          len(vouchers),
		"Codes":          codes,
		"RedeemURL":      VoucherRedeemURL(),
		"RecipientName":  html.EscapeString(vouchers[0].RecipientName),
	}
	m, err := BuildEmailRequestFromTemplate("abonementVoucher", []string{email}, data)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("voucher mail template returned no mail")
	}
	if err := m.AddAttachment("Gutschein.pdf", "application/pdf", pdf); err != nil {
		return nil, err
	}
	return m, nil
}

// ResendVoucherMail sends the voucher PDF to the buyer again, or to another address
func (db *Database) ResendVoucherMail(id int, email string) error {
	v, err := db.GetVoucherByID(id)
	if err != nil {
		return err
	}
	if email == "" {
		email = v.BuyerEmail
	}
	if email == "" {
		return errors.New("voucher has no buyer email")
	}
	m, err := db.BuildVoucherMail(email, []AbonementVoucher{v})
	if err != nil {
		return err
	}
	return sendMail(m)
}

// RedeemVoucherRequest is what the person redeeming a voucher enters
type RedeemVoucherRequest struct {
	Code      string
	Email     string
	FirstName string
	LastName  string
}

// RedeemVoucherResult describes the abonement created by redeeming a voucher
type RedeemVoucherResult struct {
	ItemName       string
	DurationMonths int
	FromDate       time.Time
	ToDate         time.Time
	NewUser        bool
}

// RedeemVoucher creates an abonement for the email from an active voucher and marks the voucher redeemed.
// If the customer already has an active abonement, the new one starts when the latest one ends.
func (db *Database) RedeemVoucher(req RedeemVoucherRequest) (result RedeemVoucherResult, err error) {
	ctx := context.Background()
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return result, errors.New("email is required")
	}

	v, err := db.GetVoucherByCode(req.Code)
	if err != nil {
		return result, err
	}
	switch v.Status {
	case VoucherStatusRedeemed:
		return result, ErrVoucherRedeemed
	case VoucherStatusDisabled:
		return result, ErrVoucherDisabled
	}
	item, err := db.voucherItem(v.ItemID)
	if err != nil {
		return result, err
	}

	keycloakID, newUser, err := VoucherAccounts.GetOrCreateUser(email)
	if err != nil {
		log.Error("RedeemVoucher: failed to get or create keycloak user: ", err)
		return result, errors.New("failed to create account")
	}
	if err := VoucherAccounts.AssignGroup(keycloakID, "customer"); err != nil {
		log.Error("RedeemVoucher: failed to assign customer group: ", err)
	}
	customer, err := db.GetOrCreateCustomerByEmail(email, keycloakID)
	if err != nil {
		return result, err
	}
	if customer.FirstName == "" && customer.LastName == "" && (req.FirstName != "" || req.LastName != "") {
		if _, err := db.EntClient.Customer.UpdateOneID(customer.ID).
			SetFirstname(strings.TrimSpace(req.FirstName)).
			SetLastname(strings.TrimSpace(req.LastName)).
			SetUpdatedAt(time.Now()).
			Save(ctx); err != nil {
			log.Error("RedeemVoucher: failed to set customer name: ", err)
		} else {
			customer.FirstName = strings.TrimSpace(req.FirstName)
			customer.LastName = strings.TrimSpace(req.LastName)
		}
	}

	tx, err := db.EntClient.Tx(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	from := now
	latest, err := tx.Abonement.Query().
		Where(
			entabonement.CustomerIDEQ(customer.ID),
			entabonement.StatusEQ("active"),
			entabonement.ToDateGT(now),
		).
		Order(ent.Desc(entabonement.FieldToDate)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return result, err
	}
	if latest != nil {
		from = latest.ToDate
	}
	to := from.AddDate(0, v.DurationMonths, 0)

	abo, err := tx.Abonement.Create().
		SetCustomerID(customer.ID).
		SetItemID(item.ID).
		SetFromDate(from).
		SetToDate(to).
		SetStatus("active").
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return result, err
	}

	// Only one request can move the voucher from active to redeemed
	n, err := tx.AbonementVoucher.Update().
		Where(entvoucher.IDEQ(v.ID), entvoucher.StatusEQ(VoucherStatusActive)).
		SetStatus(VoucherStatusRedeemed).
		SetRedeemedAt(now).
		SetRedeemedEmail(email).
		SetRedeemedCustomerID(customer.ID).
		SetAbonementID(abo.ID).
		Save(ctx)
	if err != nil {
		return result, err
	}
	if n == 0 {
		current, getErr := db.GetVoucherByID(v.ID)
		if getErr == nil && current.Status == VoucherStatusDisabled {
			return result, ErrVoucherDisabled
		}
		return result, ErrVoucherRedeemed
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}

	result = RedeemVoucherResult{
		ItemName:       item.Name,
		DurationMonths: v.DurationMonths,
		FromDate:       from,
		ToDate:         to,
		NewUser:        newUser,
	}

	db.grantVoucherAccess(customer.ID, keycloakID, item)
	db.sendVoucherRedeemedMails(email, customer, item, result)
	return result, nil
}

// grantVoucherAccess gives the customer the license groups of the voucher item and of the current online issue
func (db *Database) grantVoucherAccess(customerID int, keycloakID string, item Item) {
	groups := []string{}
	if item.LicenseGroup.Valid && item.LicenseGroup.String != "" {
		groups = append(groups, item.LicenseGroup.String)
	}
	if issue, found, err := db.GetCurrentOnlineIssue(); err != nil {
		log.Error("grantVoucherAccess: failed to look up current online_issue: ", err)
	} else if found && issue.LicenseGroup.String != "" && issue.LicenseGroup.String != item.LicenseGroup.String {
		groups = append(groups, issue.LicenseGroup.String)
	}
	for _, group := range groups {
		if _, err := db.AddLicenseGroupToCustomer(customerID, group); err != nil {
			log.Error("grantVoucherAccess: failed to add license group to customer: ", err)
		}
		if keycloakID != "" {
			if err := VoucherAccounts.AssignDigitalLicenseGroup(keycloakID, group); err != nil {
				log.Error("grantVoucherAccess: failed to assign license group: ", err)
			}
		}
	}
}

// sendVoucherRedeemedMails sends the same mails as buying an abonement does
func (db *Database) sendVoucherRedeemedMails(email string, customer *Customer, item Item, result RedeemVoucherResult) {
	inviteURL := db.CreateWordPressInvite(email, item)

	aboData := map[string]interface{}{
		// The name comes from the public redeem form and the mail body is HTML
		"CustomerName": html.EscapeString(strings.TrimSpace(customer.FirstName + " " + customer.LastName)),
		"ItemName":     item.Name,
		"FromDate":     result.FromDate.Format("2006-01-02"),
		"ToDate":       result.ToDate.Format("2006-01-02"),
		"Status":       "active",
		"InviteURL":    inviteURL,
	}
	if m, err := BuildEmailRequestFromTemplate("abonementConfirmation", []string{email}, aboData); err != nil {
		log.Error("sendVoucherRedeemedMails: failed to build abonement confirmation mail: ", err)
	} else if m != nil {
		SendVoucherMail(m)
	}

	if m, err := db.buildDigitalLicenceMail(email, inviteURL); err != nil {
		log.Error("sendVoucherRedeemedMails: failed to build digital licence mail: ", err)
	} else if m != nil {
		SendVoucherMail(m)
	}

	if result.NewUser {
		welcomeData := struct {
			URL   string
			EMAIL string
		}{URL: config.Config.OnlinePaperUrl, EMAIL: email}
		if m, err := BuildEmailRequestFromTemplate("welcome", []string{email}, welcomeData); err != nil {
			log.Error("sendVoucherRedeemedMails: failed to build welcome mail: ", err)
		} else if m != nil {
			SendVoucherMail(m)
		}
	}
}

package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/stretchr/testify/require"
)

func TestAnonymizeOldSales(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	ctx := context.Background()
	c := database.Db.EntClient

	vendor, err := c.Vendor.Query().First(ctx)
	require.NoError(t, err)
	item, err := c.Item.Query().First(ctx)
	require.NoError(t, err)

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -100)
	recent := now.AddDate(0, 0, -10)

	createOrder := func(ts time.Time) *ent.Order {
		o, err := c.Order.Create().
			SetVendorID(vendor.ID).SetVerified(true).SetTransactionTypeID(0).
			SetTransactionID("tx-" + ts.Format(time.RFC3339Nano)).SetTimestamp(ts).
			SetCustomerEmail("buyer@example.com").SetUserID("00000000-0000-4000-8000-000000000001").
			SetGiftRecipientName("Recipient").SetGiftMessage("Message").
			Save(ctx)
		require.NoError(t, err)
		return o
	}
	oldOrder := createOrder(old)
	recentOrder := createOrder(recent)

	createVoucher := func(code, status string) *ent.AbonementVoucher {
		v, err := c.AbonementVoucher.Create().
			SetCode(code).SetItemID(item.ID).SetOrderID(oldOrder.ID).SetDurationMonths(12).
			SetBuyerEmail("buyer@example.com").SetRecipientName("Recipient").SetMessage("Message").
			SetStatus(status).SetCreatedAt(old).SetRedeemedEmail("redeemer@example.com").
			Save(ctx)
		require.NoError(t, err)
		return v
	}
	redeemedVoucher := createVoucher("ANON-REDEEMED", database.VoucherStatusRedeemed)
	activeVoucher := createVoucher("ANON-ACTIVE", database.VoucherStatusActive)

	oldMail, err := c.MailTask.Create().
		SetKind(database.MailKindWelcome).SetRecipients("buyer@example.com").SetSubject("s").
		SetBody("personal link").SetStatus(database.MailTaskSent).SetOrderID(oldOrder.ID).
		SetCreatedAt(old).Save(ctx)
	require.NoError(t, err)
	pendingMail, err := c.MailTask.Create().
		SetKind(database.MailKindWelcome).SetRecipients("buyer@example.com").SetSubject("s").
		SetBody("personal link").SetOrderID(oldOrder.ID).SetCreatedAt(old).Save(ctx)
	require.NoError(t, err)

	oldExternal, err := c.ExternalPayment.Create().
		SetTransactionID("ext-old").SetAmount(500).SetMerchantTrns("Jane Doe").SetPaidAt(old).Save(ctx)
	require.NoError(t, err)
	recentExternal, err := c.ExternalPayment.Create().
		SetTransactionID("ext-recent").SetAmount(500).SetMerchantTrns("Jane Doe").SetPaidAt(recent).Save(ctx)
	require.NoError(t, err)

	enableAnonymization(t, true, false)
	anonymizeOldSales(now)

	o, err := c.Order.Get(ctx, oldOrder.ID)
	require.NoError(t, err)
	require.Nil(t, o.CustomerEmail)
	require.Nil(t, o.UserID)
	require.Empty(t, o.GiftRecipientName)
	require.Empty(t, o.GiftMessage)
	require.Equal(t, oldOrder.TransactionID, o.TransactionID)
	require.Equal(t, oldOrder.Verified, o.Verified)

	o, err = c.Order.Get(ctx, recentOrder.ID)
	require.NoError(t, err)
	require.NotNil(t, o.CustomerEmail)
	require.Equal(t, "buyer@example.com", *o.CustomerEmail)
	require.Equal(t, "Recipient", o.GiftRecipientName)

	v, err := c.AbonementVoucher.Get(ctx, redeemedVoucher.ID)
	require.NoError(t, err)
	require.Empty(t, v.BuyerEmail)
	require.Empty(t, v.RecipientName)
	require.Empty(t, v.Message)
	require.Nil(t, v.RedeemedEmail)

	v, err = c.AbonementVoucher.Get(ctx, activeVoucher.ID)
	require.NoError(t, err)
	require.Equal(t, "buyer@example.com", v.BuyerEmail)

	m, err := c.MailTask.Get(ctx, oldMail.ID)
	require.NoError(t, err)
	require.Empty(t, m.Recipients)
	require.Empty(t, m.Body)

	m, err = c.MailTask.Get(ctx, pendingMail.ID)
	require.NoError(t, err)
	require.Equal(t, "buyer@example.com", m.Recipients)

	e, err := c.ExternalPayment.Get(ctx, oldExternal.ID)
	require.NoError(t, err)
	require.Empty(t, e.MerchantTrns)
	require.Equal(t, 500, e.Amount)

	e, err = c.ExternalPayment.Get(ctx, recentExternal.ID)
	require.NoError(t, err)
	require.Equal(t, "Jane Doe", e.MerchantTrns)

	res, err := database.Db.AnonymizeSalesData(now.AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Zero(t, res.Total())
}

func enableAnonymization(t *testing.T, sales, vendors bool) {
	t.Helper()
	_, err := database.Db.EntClient.Settings.Update().
		SetSalesAnonymizationEnabled(sales).
		SetVendorAnonymizationEnabled(vendors).
		SetAnonymizationRetentionDays(90).
		Save(context.Background())
	require.NoError(t, err)
}

func TestAnonymizeOldSalesDisabled(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	ctx := context.Background()
	c := database.Db.EntClient

	vendor, err := c.Vendor.Query().First(ctx)
	require.NoError(t, err)
	old := time.Now().UTC().AddDate(0, 0, -400)
	o, err := c.Order.Create().
		SetVendorID(vendor.ID).SetVerified(true).SetTransactionTypeID(0).
		SetTransactionID("disabled").SetTimestamp(old).SetCustomerEmail("buyer@example.com").
		Save(ctx)
	require.NoError(t, err)

	enableAnonymization(t, false, false)
	anonymizeOldSales(time.Now())

	o, err = c.Order.Get(ctx, o.ID)
	require.NoError(t, err)
	require.NotNil(t, o.CustomerEmail)
	require.Equal(t, vendor.ID, o.VendorID)
}

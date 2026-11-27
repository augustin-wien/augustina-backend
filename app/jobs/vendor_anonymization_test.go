package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	entorderentry "github.com/augustin-wien/augustina-backend/ent/orderentry"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func TestAnonymizeVendorSales(t *testing.T) {
	require.NoError(t, database.Db.InitEmptyTestDb())
	ctx := context.Background()
	c := database.Db.EntClient

	vendorID, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Paid", LastName: "Out", Email: "paidout@vendor.com",
		LicenseID: null.StringFrom("anon-test"),
	})
	require.NoError(t, err)
	vendor, err := database.Db.GetVendor(vendorID)
	require.NoError(t, err)
	vendorAccount, err := database.Db.GetAccountByVendorID(vendorID)
	require.NoError(t, err)
	buyerID, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	orgaID, err := database.Db.GetAccountTypeID("Orga")
	require.NoError(t, err)
	item, err := c.Item.Query().First(ctx)
	require.NoError(t, err)

	now := time.Now().UTC()
	old := now.AddDate(0, 0, -200)

	createOrder := func(ts time.Time, tx string) *ent.Order {
		o, err := c.Order.Create().
			SetVendorID(vendorID).SetVerified(true).SetTransactionTypeID(0).
			SetTransactionID(tx).SetTimestamp(ts).Save(ctx)
		require.NoError(t, err)
		_, err = c.OrderEntry.Create().
			SetOrderID(o.ID).SetItemID(item.ID).SetQuantity(1).SetPrice(300).SetIsSale(true).
			SetSenderID(buyerID).SetReceiverID(vendorAccount.ID).Save(ctx)
		require.NoError(t, err)
		return o
	}
	createPayment := func(o *ent.Order, ts time.Time, sender, receiver, amount int, sale bool) *ent.Payment {
		p, err := c.Payment.Create().
			SetTimestamp(ts).SetAmount(amount).SetAuthorizedBy("").SetIsSale(sale).SetQuantity(1).
			SetPrice(amount).SetSenderID(sender).SetReceiverID(receiver).SetOrderID(o.ID).
			SetItemID(item.ID).Save(ctx)
		require.NoError(t, err)
		return p
	}

	paidOrder := createOrder(old, "anon-paid")
	sale := createPayment(paidOrder, old, buyerID, vendorAccount.ID, 300, true)
	licence := createPayment(paidOrder, old, vendorAccount.ID, orgaID, 150, false)

	payoutID, err := database.Db.CreatePaymentPayout(vendor, vendorAccount.ID, "admin", 150,
		[]database.Payment{{ID: sale.ID}, {ID: licence.ID}})
	require.NoError(t, err)
	_, err = c.Payment.UpdateOneID(payoutID).SetTimestamp(old).Save(ctx)
	require.NoError(t, err)

	openOrder := createOrder(old, "anon-open")
	open := createPayment(openOrder, old, buyerID, vendorAccount.ID, 300, true)

	abandoned := createOrder(old, "anon-abandoned")
	recentAbandoned := createOrder(now, "anon-recent")

	enableAnonymization(t, false, true)
	anonymizeOldSales(now)

	anonVendor, err := c.Vendor.Query().Where(entvendor.Licenseid(database.AnonymizedVendorLicenseID)).Only(ctx)
	require.NoError(t, err)
	require.True(t, anonVendor.Isdeleted)
	anonAccount, err := database.Db.GetAccountByVendorID(anonVendor.ID)
	require.NoError(t, err)

	p, err := c.Payment.Get(ctx, sale.ID)
	require.NoError(t, err)
	require.Equal(t, anonAccount.ID, p.ReceiverID)
	require.Equal(t, buyerID, p.SenderID)
	require.NotNil(t, p.PayoutID)
	require.NotEqual(t, payoutID, *p.PayoutID)

	p, err = c.Payment.Get(ctx, licence.ID)
	require.NoError(t, err)
	require.Equal(t, anonAccount.ID, p.SenderID)
	require.Equal(t, orgaID, p.ReceiverID)

	payout, err := c.Payment.Get(ctx, payoutID)
	require.NoError(t, err)
	require.Equal(t, vendorAccount.ID, payout.SenderID, "the payout stays with the vendor")
	require.Equal(t, 150, payout.Amount)

	o, err := c.Order.Get(ctx, paidOrder.ID)
	require.NoError(t, err)
	require.Equal(t, anonVendor.ID, o.VendorID)
	entry, err := c.OrderEntry.Query().Where(entorderentry.OrderID(paidOrder.ID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, anonAccount.ID, entry.ReceiverID)

	p, err = c.Payment.Get(ctx, open.ID)
	require.NoError(t, err)
	require.Equal(t, vendorAccount.ID, p.ReceiverID, "open payments stay with the vendor")
	require.Nil(t, p.PayoutID)
	o, err = c.Order.Get(ctx, openOrder.ID)
	require.NoError(t, err)
	require.Equal(t, vendorID, o.VendorID)

	o, err = c.Order.Get(ctx, abandoned.ID)
	require.NoError(t, err)
	require.Equal(t, anonVendor.ID, o.VendorID)
	o, err = c.Order.Get(ctx, recentAbandoned.ID)
	require.NoError(t, err)
	require.Equal(t, vendorID, o.VendorID)

	vendors, err := database.Db.ListVendors()
	require.NoError(t, err)
	for _, v := range vendors {
		require.NotEqual(t, database.AnonymizedVendorLicenseID, v.LicenseID.String)
	}

	res, err := database.Db.AnonymizeVendorData(now.AddDate(0, 0, -90), time.Date(now.Year()-7, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Zero(t, res.Total())

	anonymizeOldSales(old.AddDate(8, 0, 0))
	payout, err = c.Payment.Get(ctx, payoutID)
	require.NoError(t, err)
	require.Equal(t, anonAccount.ID, payout.SenderID, "the payout is anonymized after the retention period")
	p, err = c.Payment.Get(ctx, open.ID)
	require.NoError(t, err)
	require.Equal(t, vendorAccount.ID, p.ReceiverID)
}

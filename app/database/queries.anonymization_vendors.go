package database

import (
	"context"
	"fmt"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entaccount "github.com/augustin-wien/augustina-backend/ent/account"
	entorder "github.com/augustin-wien/augustina-backend/ent/order"
	entorderentry "github.com/augustin-wien/augustina-backend/ent/orderentry"
	entpayment "github.com/augustin-wien/augustina-backend/ent/payment"
	entvendor "github.com/augustin-wien/augustina-backend/ent/vendor"
)

// AnonymizedVendorLicenseID is the license ID of the placeholder vendor that anonymized sales are
// booked on. The placeholder is marked as deleted, so it is not listed with the vendors.
const AnonymizedVendorLicenseID = "ANONYMISIERT"

// anonymizedPayoutAuthorizedBy marks the placeholder payout that anonymized payments point to
const anonymizedPayoutAuthorizedBy = "anonymization"

// VendorAnonymizationResult counts the records that were detached from a vendor
type VendorAnonymizationResult struct {
	Payments int
	Orders   int
	Payouts  int
}

// Total returns the number of changed records
func (r VendorAnonymizationResult) Total() int {
	return r.Payments + r.Orders + r.Payouts
}

type anonymizedVendor struct {
	vendorID  int
	accountID int
	payoutID  int
}

// AnonymizeVendorData detaches sales from the vendors once they are no longer needed.
//
// Payments that were paid out before paidOutBefore, and the orders they belong to, are moved to a
// placeholder vendor and to a placeholder payout, so neither the account nor the payout tells whose
// sale it was. The payout itself (vendor, date, amount) is the receipt of the pass-through money
// and stays with the vendor until payoutsBefore. Payments that are not paid out yet are never
// touched, so vendor balances do not change. Orders without any payment made before paidOutBefore
// lose their vendor as well.
func (db *Database) AnonymizeVendorData(paidOutBefore, payoutsBefore time.Time) (res VendorAnonymizationResult, err error) {
	ctx := context.Background()

	anon, err := db.getOrCreateAnonymizedVendor(ctx)
	if err != nil {
		return res, fmt.Errorf("placeholder vendor: %w", err)
	}
	cashAccountID, err := db.GetAccountTypeID("Cash")
	if err != nil {
		return res, fmt.Errorf("cash account: %w", err)
	}
	vendorAccounts, err := db.EntClient.Account.Query().
		Where(entaccount.Type("Vendor"), entaccount.IDNEQ(anon.accountID)).
		Select(entaccount.FieldID, entaccount.FieldVendorID).
		All(ctx)
	if err != nil {
		return res, fmt.Errorf("vendor accounts: %w", err)
	}
	vendorAccountIDs := make([]int, 0, len(vendorAccounts))
	for _, a := range vendorAccounts {
		vendorAccountIDs = append(vendorAccountIDs, a.ID)
	}

	payouts, err := db.EntClient.Payment.Query().
		Where(
			entpayment.ReceiverID(cashAccountID),
			entpayment.SenderIDIn(vendorAccountIDs...),
			entpayment.TimestampLT(paidOutBefore),
			entpayment.HasChildren(),
		).
		Select(entpayment.FieldID, entpayment.FieldSenderID).
		All(ctx)
	if err != nil {
		return res, fmt.Errorf("payouts: %w", err)
	}
	for _, p := range payouts {
		payments, orders, err := db.detachPayout(ctx, p.ID, p.SenderID, anon)
		if err != nil {
			return res, fmt.Errorf("payout %d: %w", p.ID, err)
		}
		res.Payments += payments
		res.Orders += orders
	}

	orders, err := db.detachOrdersWithoutPayments(ctx, paidOutBefore, vendorAccountIDs, anon)
	if err != nil {
		return res, fmt.Errorf("orders without payments: %w", err)
	}
	res.Orders += orders

	res.Payouts, err = db.EntClient.Payment.Update().
		Where(
			entpayment.ReceiverID(cashAccountID),
			entpayment.SenderIDIn(vendorAccountIDs...),
			entpayment.TimestampLT(payoutsBefore),
		).
		SetSenderID(anon.accountID).
		Save(ctx)
	if err != nil {
		return res, fmt.Errorf("payouts before %s: %w", payoutsBefore.Format(time.DateOnly), err)
	}
	return res, nil
}

// detachPayout moves the payments of one payout and their orders from the vendor account to the
// placeholder. Orders that still have payments of the vendor in another, newer payout are kept.
func (db *Database) detachPayout(ctx context.Context, payoutID, vendorAccountID int, anon anonymizedVendor) (payments, orders int, err error) {
	tx, err := db.EntClient.Tx(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	orderIDs, err := tx.Payment.Query().
		Where(entpayment.PayoutID(payoutID), entpayment.OrderIDNotNil()).
		Unique(true).
		Select(entpayment.FieldOrderID).
		Ints(ctx)
	if err != nil {
		return 0, 0, err
	}

	if _, err = tx.Payment.Update().
		Where(entpayment.PayoutID(payoutID), entpayment.SenderID(vendorAccountID)).
		SetSenderID(anon.accountID).
		Save(ctx); err != nil {
		return 0, 0, err
	}
	if _, err = tx.Payment.Update().
		Where(entpayment.PayoutID(payoutID), entpayment.ReceiverID(vendorAccountID)).
		SetReceiverID(anon.accountID).
		Save(ctx); err != nil {
		return 0, 0, err
	}
	if payments, err = tx.Payment.Update().
		Where(entpayment.PayoutID(payoutID)).
		SetPayoutID(anon.payoutID).
		Save(ctx); err != nil {
		return 0, 0, err
	}

	if len(orderIDs) > 0 {
		var stillLinked []int
		stillLinked, err = tx.Payment.Query().
			Where(
				entpayment.OrderIDIn(orderIDs...),
				entpayment.Or(entpayment.SenderID(vendorAccountID), entpayment.ReceiverID(vendorAccountID)),
			).
			Unique(true).
			Select(entpayment.FieldOrderID).
			Ints(ctx)
		if err != nil {
			return 0, 0, err
		}
		if orders, err = detachOrders(ctx, tx, without(orderIDs, stillLinked), []int{vendorAccountID}, anon); err != nil {
			return 0, 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return payments, orders, nil
}

// detachOrdersWithoutPayments removes the vendor from orders that never got a payment, such as
// abandoned checkouts
func (db *Database) detachOrdersWithoutPayments(ctx context.Context, before time.Time, vendorAccountIDs []int, anon anonymizedVendor) (n int, err error) {
	tx, err := db.EntClient.Tx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	orderIDs, err := tx.Order.Query().
		Where(
			entorder.TimestampLT(before),
			entorder.VendorIDNEQ(anon.vendorID),
			entorder.Not(entorder.HasPayments()),
		).
		IDs(ctx)
	if err != nil {
		return 0, err
	}
	if n, err = detachOrders(ctx, tx, orderIDs, vendorAccountIDs, anon); err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

// detachOrders moves the given orders and their entries from the vendor to the placeholder
func detachOrders(ctx context.Context, tx *ent.Tx, orderIDs, vendorAccountIDs []int, anon anonymizedVendor) (int, error) {
	if len(orderIDs) == 0 {
		return 0, nil
	}
	n, err := tx.Order.Update().
		Where(entorder.IDIn(orderIDs...), entorder.VendorIDNEQ(anon.vendorID)).
		SetVendorID(anon.vendorID).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	if _, err = tx.OrderEntry.Update().
		Where(entorderentry.OrderIDIn(orderIDs...), entorderentry.SenderIDIn(vendorAccountIDs...)).
		SetSenderID(anon.accountID).
		Save(ctx); err != nil {
		return 0, err
	}
	if _, err = tx.OrderEntry.Update().
		Where(entorderentry.OrderIDIn(orderIDs...), entorderentry.ReceiverIDIn(vendorAccountIDs...)).
		SetReceiverID(anon.accountID).
		Save(ctx); err != nil {
		return 0, err
	}
	return n, nil
}

// getOrCreateAnonymizedVendor returns the placeholder vendor, its account and its payout
func (db *Database) getOrCreateAnonymizedVendor(ctx context.Context) (anon anonymizedVendor, err error) {
	tx, err := db.EntClient.Tx(ctx)
	if err != nil {
		return anon, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	v, err := tx.Vendor.Query().Where(entvendor.Licenseid(AnonymizedVendorLicenseID)).Only(ctx)
	if ent.IsNotFound(err) {
		v, err = tx.Vendor.Create().
			SetLicenseid(AnonymizedVendorLicenseID).
			SetFirstname("Anonymisiert").
			SetLastname("").
			SetEmail("").
			SetHasownemail(false).
			SetKeycloakid("").
			SetUrlid("").
			SetLanguage("").
			SetTelephone("").
			SetRegistrationdate("").
			SetVendorsince("").
			SetAccountproofurl("").
			SetDebt("").
			SetLastpayout(time.Time{}).
			SetIsdisabled(true).
			SetIsdeleted(true).
			Save(ctx)
	}
	if err != nil {
		return anon, err
	}
	anon.vendorID = v.ID

	acc, err := tx.Account.Query().Where(entaccount.VendorID(v.ID)).Only(ctx)
	if ent.IsNotFound(err) {
		acc, err = tx.Account.Create().
			SetName(AnonymizedVendorLicenseID).
			SetBalance(0).
			SetType("Vendor").
			SetVendorID(v.ID).
			Save(ctx)
	}
	if err != nil {
		return anon, err
	}
	anon.accountID = acc.ID

	cash, err := tx.Account.Query().Where(entaccount.Type("Cash")).First(ctx)
	if err != nil {
		return anon, err
	}
	payout, err := tx.Payment.Query().
		Where(entpayment.SenderID(acc.ID), entpayment.ReceiverID(cash.ID),
			entpayment.AuthorizedBy(anonymizedPayoutAuthorizedBy)).
		First(ctx)
	if ent.IsNotFound(err) {
		payout, err = tx.Payment.Create().
			SetSenderID(acc.ID).
			SetReceiverID(cash.ID).
			SetAmount(0).
			SetAuthorizedBy(anonymizedPayoutAuthorizedBy).
			SetIsSale(false).
			SetQuantity(0).
			SetPrice(0).
			SetTimestamp(time.Now()).
			Save(ctx)
	}
	if err != nil {
		return anon, err
	}
	anon.payoutID = payout.ID

	return anon, tx.Commit()
}

// without returns the values of all that are not in remove
func without(all, remove []int) []int {
	skip := make(map[int]struct{}, len(remove))
	for _, v := range remove {
		skip[v] = struct{}{}
	}
	out := make([]int, 0, len(all))
	for _, v := range all {
		if _, ok := skip[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

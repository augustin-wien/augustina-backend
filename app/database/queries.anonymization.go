package database

import (
	"context"
	"fmt"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entvoucher "github.com/augustin-wien/augustina-backend/ent/abonementvoucher"
	entexternalpayment "github.com/augustin-wien/augustina-backend/ent/externalpayment"
	"github.com/augustin-wien/augustina-backend/ent/mailtask"
	entorder "github.com/augustin-wien/augustina-backend/ent/order"
)

// MinAnonymizationRetentionDays is the shortest retention that can be set, so refunds and
// chargebacks can still be handled
const MinAnonymizationRetentionDays = 30

// SalesAnonymizationResult counts the records whose personal data was removed
type SalesAnonymizationResult struct {
	Orders           int
	Vouchers         int
	MailTasks        int
	ExternalPayments int
}

// Total returns the number of changed records
func (r SalesAnonymizationResult) Total() int {
	return r.Orders + r.Vouchers + r.MailTasks + r.ExternalPayments
}

// AnonymizeSalesData removes the personal data of sales made before the cutoff.
//
// Amounts, items, vendors, timestamps and the payment provider's transaction IDs stay untouched,
// so reports, payouts and the accounting keep working. Removed are the buyer's email address and
// Keycloak user ID on the order, gift texts, the addresses and texts of finished abonement
// vouchers, recipients and content of the mails sent for the order, and the free text VivaWallet
// stores with external payments. Active vouchers are kept as they can still be redeemed.
func (db *Database) AnonymizeSalesData(before time.Time) (res SalesAnonymizationResult, err error) {
	ctx := context.Background()
	tx, err := db.EntClient.Tx(ctx)
	if err != nil {
		return res, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if res.Orders, err = anonymizeOrders(ctx, tx, before); err != nil {
		return res, fmt.Errorf("anonymize orders: %w", err)
	}
	if res.Vouchers, err = anonymizeVouchers(ctx, tx, before); err != nil {
		return res, fmt.Errorf("anonymize vouchers: %w", err)
	}
	if res.MailTasks, err = anonymizeOrderMails(ctx, tx, before); err != nil {
		return res, fmt.Errorf("anonymize order mails: %w", err)
	}
	if res.ExternalPayments, err = anonymizeExternalPayments(ctx, tx, before); err != nil {
		return res, fmt.Errorf("anonymize external payments: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return SalesAnonymizationResult{}, err
	}
	return res, nil
}

func anonymizeOrders(ctx context.Context, tx *ent.Tx, before time.Time) (int, error) {
	return tx.Order.Update().
		Where(
			entorder.TimestampLT(before),
			entorder.Or(
				entorder.CustomerEmailNotNil(),
				entorder.UserIDNotNil(),
				entorder.GiftRecipientNameNEQ(""),
				entorder.GiftMessageNEQ(""),
			),
		).
		ClearCustomerEmail().
		ClearUserID().
		SetGiftRecipientName("").
		SetGiftMessage("").
		Save(ctx)
}

func anonymizeVouchers(ctx context.Context, tx *ent.Tx, before time.Time) (int, error) {
	return tx.AbonementVoucher.Update().
		Where(
			entvoucher.CreatedAtLT(before),
			entvoucher.StatusNEQ(VoucherStatusActive),
			entvoucher.Or(
				entvoucher.BuyerEmailNEQ(""),
				entvoucher.RecipientNameNEQ(""),
				entvoucher.MessageNEQ(""),
				entvoucher.RedeemedEmailNotNil(),
			),
		).
		SetBuyerEmail("").
		SetRecipientName("").
		SetMessage("").
		ClearRedeemedEmail().
		Save(ctx)
}

func anonymizeOrderMails(ctx context.Context, tx *ent.Tx, before time.Time) (int, error) {
	return tx.MailTask.Update().
		Where(
			mailtask.OrderIDNotNil(),
			mailtask.CreatedAtLT(before),
			mailtask.StatusIn(MailTaskSent, MailTaskFailed),
			mailtask.Or(
				mailtask.RecipientsNEQ(""),
				mailtask.BodyNEQ(""),
				mailtask.AttachmentsNotNil(),
				mailtask.LastErrorNotNil(),
			),
		).
		SetRecipients("").
		SetBody("").
		ClearAttachments().
		ClearLastError().
		Save(ctx)
}

func anonymizeExternalPayments(ctx context.Context, tx *ent.Tx, before time.Time) (int, error) {
	return tx.ExternalPayment.Update().
		Where(
			entexternalpayment.PaidAtLT(before),
			entexternalpayment.MerchantTrnsNEQ(""),
		).
		SetMerchantTrns("").
		Save(ctx)
}

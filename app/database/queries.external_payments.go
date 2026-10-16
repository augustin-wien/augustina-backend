package database

import (
	"context"
	"errors"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
	entexternalpayment "github.com/augustin-wien/augustina-backend/ent/externalpayment"
)

// ExternalPaymentUpdate holds what one VivaWallet webhook knows about an external payment.
// Zero values are left out, because the success and price webhooks each carry only part of it
// and arrive in either order.
type ExternalPaymentUpdate struct {
	TransactionID string
	OrderCode     string
	Amount        int
	Fee           int
	StatusID      string
	SourceCode    string
	SourceName    string
	TerminalID    int
	MerchantTrns  string
	PaidAt        time.Time
}

// RecordExternalPayment creates the external payment for the transaction or fills in the fields
// the update carries. Repeated webhook deliveries leave the payment unchanged.
func (db *Database) RecordExternalPayment(p ExternalPaymentUpdate) error {
	if p.TransactionID == "" {
		return errors.New("RecordExternalPayment: transaction id is empty")
	}
	err := db.recordExternalPayment(p)
	if ent.IsConstraintError(err) {
		// A concurrent webhook for the same transaction created it first
		err = db.recordExternalPayment(p)
	}
	if err != nil {
		log.Errorw("RecordExternalPayment: failed", "transaction_id", p.TransactionID, "error", err)
	}
	return err
}

func (db *Database) recordExternalPayment(p ExternalPaymentUpdate) error {
	ctx := context.Background()
	existing, err := db.EntClient.ExternalPayment.Query().
		Where(entexternalpayment.TransactionID(p.TransactionID)).
		Only(ctx)
	if ent.IsNotFound(err) {
		create := db.EntClient.ExternalPayment.Create().
			SetTransactionID(p.TransactionID).
			SetOrderCode(p.OrderCode).
			SetAmount(p.Amount).
			SetFee(p.Fee).
			SetStatusID(p.StatusID).
			SetSourceCode(p.SourceCode).
			SetSourceName(p.SourceName).
			SetTerminalID(p.TerminalID).
			SetMerchantTrns(p.MerchantTrns)
		if !p.PaidAt.IsZero() {
			create.SetPaidAt(p.PaidAt)
		}
		return create.Exec(ctx)
	}
	if err != nil {
		return err
	}

	update := existing.Update()
	if p.OrderCode != "" {
		update.SetOrderCode(p.OrderCode)
	}
	if p.Amount != 0 {
		update.SetAmount(p.Amount)
	}
	if p.Fee != 0 {
		update.SetFee(p.Fee)
	}
	if p.StatusID != "" {
		update.SetStatusID(p.StatusID)
	}
	if p.SourceCode != "" {
		update.SetSourceCode(p.SourceCode)
	}
	if p.SourceName != "" {
		update.SetSourceName(p.SourceName)
	}
	if p.TerminalID != 0 {
		update.SetTerminalID(p.TerminalID)
	}
	if p.MerchantTrns != "" {
		update.SetMerchantTrns(p.MerchantTrns)
	}
	if !p.PaidAt.IsZero() {
		update.SetPaidAt(p.PaidAt)
	}
	return update.Exec(ctx)
}

// ListExternalPayments returns the external payments paid within the period, newest first.
// A zero from or to leaves that side of the period open.
func (db *Database) ListExternalPayments(from, to time.Time) ([]*ent.ExternalPayment, error) {
	query := db.EntClient.ExternalPayment.Query()
	if !from.IsZero() {
		query.Where(entexternalpayment.PaidAtGTE(from))
	}
	if !to.IsZero() {
		query.Where(entexternalpayment.PaidAtLT(to))
	}
	payments, err := query.
		Order(ent.Desc(entexternalpayment.FieldPaidAt), ent.Desc(entexternalpayment.FieldID)).
		All(context.Background())
	if err != nil {
		log.Error("ListExternalPayments: ", err)
	}
	return payments, err
}

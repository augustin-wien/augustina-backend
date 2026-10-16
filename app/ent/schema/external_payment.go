package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// ExternalPayment holds the schema definition for the ExternalPayment entity.
//
// An external payment is a VivaWallet payment that was not made through this backend, such as a
// donation at a card terminal on the same VivaWallet account. It is recorded from the webhooks
// while donation tracking is enabled, so the backoffice can list these payments.
type ExternalPayment struct {
	ent.Schema
}

// Fields of the ExternalPayment.
func (ExternalPayment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("transaction_id").
			Unique(),
		field.String("order_code").
			Default(""),
		// Amounts in cents
		field.Int("amount").
			Default(0),
		field.Int("fee").
			Default(0),
		field.String("status_id").
			Default(""),
		field.String("source_code").
			Default(""),
		field.String("source_name").
			Default(""),
		field.Int("terminal_id").
			Default(0),
		field.String("merchant_trns").
			Default(""),
		field.Time("paid_at").
			Default(time.Now),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the ExternalPayment.
func (ExternalPayment) Edges() []ent.Edge {
	return nil
}

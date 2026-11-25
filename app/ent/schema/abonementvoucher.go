package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AbonementVoucher holds the schema definition for a gift voucher that can be
// redeemed for an online abonement.
type AbonementVoucher struct {
	ent.Schema
}

// Fields of the AbonementVoucher.
func (AbonementVoucher) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("code").
			NotEmpty().
			Unique(),
		field.Int("item_id"),
		field.Int("order_id").
			Optional().
			Nillable(),
		field.String("buyer_email").
			Default(""),
		field.String("recipient_name").
			Default(""),
		field.String("message").
			Default(""),
		field.Int("duration_months"),
		// active, redeemed or disabled
		field.String("status").
			Default("active"),
		field.Time("created_at"),
		field.Time("redeemed_at").
			Optional().
			Nillable(),
		field.String("redeemed_email").
			Optional().
			Nillable(),
		field.Int("redeemed_customer_id").
			Optional().
			Nillable(),
		field.Int("abonement_id").
			Optional().
			Nillable(),
	}
}

// Indexes of the AbonementVoucher.
func (AbonementVoucher) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id"),
	}
}

// Annotations of the AbonementVoucher.
func (AbonementVoucher) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "abonement_voucher"},
	}
}

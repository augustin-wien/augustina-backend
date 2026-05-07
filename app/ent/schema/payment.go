package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Payment holds the schema definition for the Payment entity.
type Payment struct {
	ent.Schema
}

// Fields of the Payment.
func (Payment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.Time("timestamp"),
		field.Int("amount"),
		field.String("authorized_by"),
		field.Bool("is_sale"),
		field.Int("quantity"),
		field.Int("price"),

		// Foreign keys
		field.Int("sender_id").StorageKey("sender"),
		field.Int("receiver_id").StorageKey("receiver"),
		field.Int("order_id").
			Optional().
			Nillable().
			StorageKey("paymentorder"),
		field.Int("order_entry_id").
			Optional().
			Nillable().
			StorageKey("order_entry"),
		field.Int("item_id").
			Optional().
			Nillable().
			StorageKey("item"),
		field.Int("payout_id").
			Optional().
			Nillable().
			StorageKey("payout"),
	}
}

// Edges of the Payment.
func (Payment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).
			Ref("payments").
			Unique().
			Field("order_id"),
		edge.To("children", Payment.Type).
			From("parent").
			Unique().
			Field("payout_id"),
	}
}

// Annotations of the Payment.
func (Payment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment"},
	}
}

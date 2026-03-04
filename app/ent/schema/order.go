package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Order holds the schema definition for the Order entity.
type Order struct {
	ent.Schema
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("order_code").
			Optional().
			Nillable(),
		field.String("transaction_id"),
		field.Bool("verified"),
		field.Time("verified_at").
			Optional().
			Nillable(),
		field.Int("transaction_type_id"),
		field.Time("timestamp"),
		field.String("user_id").
			Optional().
			Nillable().
			StorageKey("userid"),
		field.Int("vendor_id"), // This seems to be an ID, referring to Vendor?
		field.String("customer_email").
			Optional().
			Nillable().
			StorageKey("customeremail"),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entries", OrderEntry.Type),
		edge.To("payments", Payment.Type),
	}
}

// Annotations of the Order.
func (Order) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "paymentorder"},
	}
}

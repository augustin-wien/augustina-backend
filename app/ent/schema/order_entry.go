package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// OrderEntry holds the schema definition for the OrderEntry entity.
type OrderEntry struct {
	ent.Schema
}

// Fields of the OrderEntry.
func (OrderEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.Int("quantity"),
		field.Int("price"),
		field.Bool("is_sale"),
		// Foreign keys as fields if not using Edges for everything,
		// but Ent prefers Edges.
		// However, existing SQL uses IDs.
		field.Int("item_id").StorageKey("item"),
		field.Int("sender_id").StorageKey("sender"),
		field.Int("receiver_id").StorageKey("receiver"),
		field.Int("order_id").
			Optional().
			StorageKey("paymentorder"),
	}
}

// Edges of the OrderEntry.
func (OrderEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).
			Ref("entries").
			Unique().
			Field("order_id"),
		edge.To("item", Item.Type).
			Unique().
			Required().
			Field("item_id"),
		edge.To("sender", Account.Type).
			Unique().
			Required().
			Field("sender_id"),
		edge.To("receiver", Account.Type).
			Unique().
			Required().
			Field("receiver_id"),
	}
}

// Annotations of the OrderEntry.
func (OrderEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "orderentry"},
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Abonement holds the schema definition for the Abonement entity.
type Abonement struct {
	ent.Schema
}

// Fields of the Abonement.
func (Abonement) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.Int("customer_id").
			StorageKey("customer_id"),
		field.Int("item_id").
			StorageKey("abonement_item").
			Optional().
			Nillable(),
		field.Time("from_date").
			StorageKey("from_date"),
		field.Time("to_date").
			StorageKey("to_date"),
		field.String("status").
			Default("active").
			StorageKey("status"),
		field.Time("created_at").
			Optional().
			Nillable().
			StorageKey("created_at"),
		field.Time("updated_at").
			Optional().
			Nillable().
			StorageKey("updated_at"),
	}
}

// Edges of the Abonement.
func (Abonement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Ref("abonements").
			Unique().
			Field("customer_id").
			Required(),
		edge.To("item", Item.Type).
			Field("item_id").
			Unique(),
	}
}

// Annotations of the Abonement.
func (Abonement) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "abonement"},
	}
}

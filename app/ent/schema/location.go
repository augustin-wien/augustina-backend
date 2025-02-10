package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Location holds the schema definition for the Location entity.
type Location struct {
	ent.Schema
}

// Fields of the Location.
func (Location) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("name"),
		field.String("address"),
		field.Float("longitude").
			Default(0.1),
		field.Float("latitude").
			Default(0.1),
		field.String("zip"),
		field.String("working_time"),
	}
}

// Edges of the Location.
func (Location) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vendor", Vendor.Type).
			Ref("locations").
			Unique(), // Ensures each Location is associated with only one Vendor
	}
}

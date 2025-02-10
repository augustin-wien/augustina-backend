package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Comment holds the schema definition for the Comment entity.
type Comment struct {
	ent.Schema
}

// Fields of the Comment.
func (Comment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("comment"),
		field.Bool("warning"),
		field.Time("created_at"),
		field.Time("resolved_at"),
	}
}

// Edges of the Comment.
func (Comment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vendor", Vendor.Type).
			Ref("comments").
			Unique(), // Ensures each Location is associated with only one Vendor
	}
}

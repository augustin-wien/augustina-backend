package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Customer holds the schema definition for the Customer entity.
type Customer struct {
	ent.Schema
}

// Fields of the Customer.
func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("keycloakid").
			StorageKey("keycloakid"),
		field.String("email").
			Default(""),
		field.String("firstname").
			Default(""),
		field.String("lastname").
			Default(""),
		field.String("licensegroups").
			Default("").
			StorageKey("licensegroups"),
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

// Edges of the Customer.
func (Customer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("abonements", Abonement.Type),
	}
}

// Annotations of the Customer.
func (Customer) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "customer"},
	}
}

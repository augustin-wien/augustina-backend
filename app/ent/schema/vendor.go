package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Vendor holds the schema definition for the Vendor entity.
type Vendor struct {
	ent.Schema
}

// Fields of the Vendor.
func (Vendor) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("keycloakid"),
		field.String("urlid"),
		field.String("licenseid").
			Default("unknown"),
		field.String("firstname").
			Default("unknown"),
		field.String("lastname").
			Default("unknown"),
		field.String("email").
			Default("@augustina.cc"),
		field.Time("lastpayout"),
		field.Bool("isdisabled").
			Default(false),
		field.String("language"),
		field.String("telephone"),
		field.String("registrationdate"),
		field.String("vendorsince"),
		field.Bool("onlinemap").
			Default(false),
		field.Bool("hassmartphone").
			Default(false),
		field.Bool("hasbankaccount").
			Default(false),
		field.Bool("isdeleted").
			Default(false),
		field.String("accountproofurl"),
		field.String("debt"),
	}
}

// Edges of the Vendor.
func (Vendor) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("locations", Location.Type),
		edge.To("comments", Comment.Type),
	}
}

// Annotations of the User.
func (Vendor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "vendor"},
	}
}

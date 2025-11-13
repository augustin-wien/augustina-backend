package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Location holds the schema definition for the Location entity.
type PDF struct {
	ent.Schema
}

// Fields of the PDF.
func (PDF) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("path"),
		field.String("timestamp"),
	}
}

func (PDF) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "pdf"},
	}
}

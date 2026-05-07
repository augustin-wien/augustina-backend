package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// DBSettings holds the schema definition for the DBSettings entity.
type DBSettings struct {
	ent.Schema
}

// Fields of the DBSettings.
func (DBSettings) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.Bool("is_initialized"),
	}
}

// Annotations of the DBSettings.
func (DBSettings) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "db_settings"},
	}
}

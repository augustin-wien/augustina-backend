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
		// Month (YYYY-MM) of the last accounting report mail, so a restart does not send it twice
		field.String("accounting_report_sent_month").
			Default(""),
		// Last day (YYYY-MM-DD) of the last report exported or sent in the backoffice
		field.String("accounting_report_last_to").
			Default(""),
	}
}

// Annotations of the DBSettings.
func (DBSettings) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "db_settings"},
	}
}

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// MailTemplate holds the schema definition for the MailTemplate entity.
type MailTemplate struct {
	ent.Schema
}

// Fields of the MailTemplate.
func (MailTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.String("name").Unique(),
		field.String("subject"),
		field.String("body"),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

// Edges of the MailTemplate.
func (MailTemplate) Edges() []ent.Edge {
	return nil
}

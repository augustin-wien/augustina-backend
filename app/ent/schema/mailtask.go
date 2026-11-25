package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// MailTask holds the schema definition for an outgoing mail. Every mail the
// backend sends is stored as a task first, so its delivery state is kept and a
// failed mail can be sent again.
type MailTask struct {
	ent.Schema
}

// Fields of the MailTask.
func (MailTask) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		// What the mail is about, e.g. online_issue, welcome, digital_licence
		field.String("kind"),
		// Comma separated recipient addresses
		field.Text("recipients"),
		field.Text("subject"),
		field.Text("body"),
		// JSON encoded attachments
		field.Bytes("attachments").
			Optional().
			Nillable(),
		// pending, sending, sent or failed
		field.String("status").
			Default("pending"),
		field.Int("attempts").
			Default(0),
		field.Text("last_error").
			Optional().
			Nillable(),
		field.Time("next_attempt_at").
			Default(time.Now),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("sent_at").
			Optional().
			Nillable(),
		field.Int("order_id").
			Optional().
			Nillable(),
		field.Int("item_id").
			Optional().
			Nillable(),
	}
}

// Indexes of the MailTask.
func (MailTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "next_attempt_at"),
		index.Fields("item_id", "kind"),
		index.Fields("order_id"),
		index.Fields("created_at"),
	}
}

// Annotations of the MailTask.
func (MailTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "mail_task"},
	}
}

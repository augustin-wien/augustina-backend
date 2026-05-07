package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// PDFDownload holds the schema definition for the PDFDownload entity.
type PDFDownload struct {
	ent.Schema
}

// Fields of the PDFDownload.
func (PDFDownload) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("link_id").
			Unique(),
		field.Int("pdf_id"),
		field.Time("timestamp"),
		field.Bool("email_sent").
			Default(false),
		field.Time("last_download").
			Optional().
			Nillable(),
		field.Int("download_count").
			Default(0),
		field.Int("order_id").
			Optional().
			Nillable(),
		field.Int("item_id").
			Optional().
			Nillable(),
	}
}

// Edges of the PDFDownload.
// Note: We don't have edges to Order or Item yet because the user mentioned Order is not in schema.
func (PDFDownload) Edges() []ent.Edge {
	return nil
}

// Annotations of the PDFDownload.
func (PDFDownload) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "pdf_download"},
	}
}

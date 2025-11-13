package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Item holds the schema definition for the Item entity.
type Item struct {
	ent.Schema
}

// Fields of the Item.
func (Item) Fields() []ent.Field {
	fields := []ent.Field{
		field.Int("id").
			Positive(),
		field.String("Name").
			NotEmpty(),
		field.String("Description").
			NotEmpty(),
		field.Float("Price").
			Positive(),
		field.String("Image").
			NotEmpty(),
		field.Bool("Archived").
			Default(false),
		field.Bool("Disabled").
			StorageKey("disabled").
			Default(false),
		field.Bool("IsLicenseItem").
			StorageKey("islicenseitem").
			Default(false),
		field.String("LicenseGroup").
			StorageKey("licensegroup").
			Default("default"),
		field.Bool("IsPDFItem").
			StorageKey("ispdfitem").
			Default(false),
		field.Int("ItemOrder").
			StorageKey("itemorder").
			Default(0),
		field.String("ItemColor").
			StorageKey("itemcolor").
			Default("#FFFFFF"),
		field.String("ItemTextColor").
			StorageKey("itemtextcolor").
			Default("#000000"),
	}
	for _, f := range fields {
		f.Descriptor().Tag = `json:"` + f.Descriptor().Name + `"`
	}
	return fields
}

// Edges of the Item.
func (Item) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("LicenseItem", Item.Type).Unique().StorageKey(edge.Column("licenseitem")),
		edge.To("PDF", PDF.Type).Unique().StorageKey(edge.Column("pdf")),
	}
}

// Annotations of the User.
func (Item) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "item"},
	}
}

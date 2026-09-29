package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Campaign holds the schema definition for the Campaign entity.
//
// A campaign advertises one shop item in a one-time popup while it is enabled and within its
// period. views and clicks are plain counters so the backoffice can show a click rate.
type Campaign struct {
	ent.Schema
}

// Fields of the Campaign.
func (Campaign) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		// Internal name, only shown in the backoffice
		field.String("name"),
		// Not an edge: items are archived rather than deleted, and a campaign whose item is no
		// longer sold simply doesn't show up in the shop.
		field.Int("item_id"),
		field.String("title").
			Default(""),
		field.Text("text").
			Default(""),
		// A missing start/end leaves that side of the period open
		field.Time("starts_at").
			Optional().
			Nillable(),
		field.Time("ends_at").
			Optional().
			Nillable(),
		field.Bool("enabled").
			Default(false),
		field.Int("views").
			Default(0),
		field.Int("clicks").
			Default(0),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Campaign.
func (Campaign) Edges() []ent.Edge {
	return nil
}

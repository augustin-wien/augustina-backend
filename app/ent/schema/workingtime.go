package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WorkingTime holds per-day opening hours for a Vendor or Location.
type WorkingTime struct {
	ent.Schema
}

// Fields of the WorkingTime.
func (WorkingTime) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.Enum("day").
			Values("monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday", "weekdays", "weekend"),
		field.String("open_time").
			Optional(),
		field.String("close_time").
			Optional(),
		field.Bool("closed").
			Default(false),
	}
}

// Edges of the WorkingTime.
func (WorkingTime) Edges() []ent.Edge {
	return []ent.Edge{
		// WorkingTime is bound to a Location (online availability per location).
		edge.From("location", Location.Type).
			Ref("working_times").
			Unique().
			Required(),
	}
}

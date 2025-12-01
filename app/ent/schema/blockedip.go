package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// BlockedIP holds the schema definition for the BlockedIP entity.
type BlockedIP struct {
	ent.Schema
}

// Fields of the BlockedIP.
func (BlockedIP) Fields() []ent.Field {
	return []ent.Field{
		field.String("ip").
			Unique().
			NotEmpty(),
		field.Int("strikes").
			Default(0),
		field.Time("block_expires_at").
			Optional(),
		field.String("reason").
			Optional(),
	}
}

// Edges of the BlockedIP.
func (BlockedIP) Edges() []ent.Edge {
	return nil
}

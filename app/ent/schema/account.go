package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Account holds the schema definition for the Account entity.
type Account struct {
	ent.Schema
}

// Fields of the Account.
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("name").
			Optional(),
		field.Float("balance").
			Default(0),
		field.String("type"), // "Cash", "Orga", "UserAnon", "Paypal", "VivaWallet", "Vendor", "UserAuth", "Backoffice"
		field.String("user_id").
			Optional().
			StorageKey("userid"),
		field.Int("vendor_id").
			Optional().
			StorageKey("vendor"),
	}
}

// Edges of the Account.
func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vendor", Vendor.Type).
			Ref("accounts").
			Unique().
			Field("vendor_id"),
	}
}

// Annotations of the Account.
func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account"},
	}
}

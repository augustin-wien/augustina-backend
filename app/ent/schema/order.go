package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Order holds the schema definition for the Order entity.
type Order struct {
	ent.Schema
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Positive(),
		field.String("order_code").
			Optional().
			Nillable(),
		field.String("transaction_id"),
		field.Bool("verified"),
		field.Time("verified_at").
			Optional().
			Nillable(),
		field.Int("transaction_type_id"),
		field.Time("timestamp"),
		field.String("user_id").
			Optional().
			Nillable().
			StorageKey("userid"),
		field.Int("vendor_id"), // This seems to be an ID, referring to Vendor?
		field.String("customer_email").
			Optional().
			Nillable().
			StorageKey("customeremail"),
		field.Time("odoo_synced_at").
			Optional().
			Nillable(),
		field.String("odoo_sync_error").
			Optional().
			Nillable(),
		// Set once an unverified order is judged permanently abandoned (no
		// transaction ID after 20 minutes, or VivaWallet itself confirms no
		// payment exists) - see jobs/reconcile.go. Never set on an order
		// VivaWallet confirms was paid; the reconcile job verifies those.
		field.Time("invalidated_at").
			Optional().
			Nillable(),
		// Set when the reconcile job found the order paid at VivaWallet but could
		// not verify it, so the failure is reported once and not on every run
		field.Time("reconcile_failed_at").
			Optional().
			Nillable(),
		field.String("reconcile_error").
			Optional().
			Nillable(),
		// Customer opted in to the newsletter at checkout; copied to the
		// customer record once the order is verified.
		field.Bool("newsletter").
			Default(false),
		// EventId VivaWallet appended to the failure redirect of the last failed payment attempt
		field.Int("payment_event_id").
			Optional().
			Nillable(),
		field.Time("payment_failed_at").
			Optional().
			Nillable(),
		// Printed on the PDF of an abonement voucher bought with the order
		field.String("gift_recipient_name").
			Default(""),
		field.String("gift_message").
			Default(""),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entries", OrderEntry.Type),
		edge.To("payments", Payment.Type),
	}
}

// Annotations of the Order.
func (Order) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "paymentorder"},
	}
}

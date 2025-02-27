// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// CommentsColumns holds the columns for the "comments" table.
	CommentsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt, Increment: true},
		{Name: "comment", Type: field.TypeString},
		{Name: "warning", Type: field.TypeBool},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "resolved_at", Type: field.TypeTime},
		{Name: "vendor_comments", Type: field.TypeInt, Nullable: true},
	}
	// CommentsTable holds the schema information for the "comments" table.
	CommentsTable = &schema.Table{
		Name:       "comments",
		Columns:    CommentsColumns,
		PrimaryKey: []*schema.Column{CommentsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "comments_vendor_comments",
				Columns:    []*schema.Column{CommentsColumns[5]},
				RefColumns: []*schema.Column{VendorColumns[0]},
				OnDelete:   schema.SetNull,
			},
		},
	}
	// LocationsColumns holds the columns for the "locations" table.
	LocationsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt, Increment: true},
		{Name: "name", Type: field.TypeString},
		{Name: "address", Type: field.TypeString},
		{Name: "longitude", Type: field.TypeFloat64, Default: 0.1},
		{Name: "latitude", Type: field.TypeFloat64, Default: 0.1},
		{Name: "zip", Type: field.TypeString},
		{Name: "working_time", Type: field.TypeString},
		{Name: "vendor_locations", Type: field.TypeInt, Nullable: true},
	}
	// LocationsTable holds the schema information for the "locations" table.
	LocationsTable = &schema.Table{
		Name:       "locations",
		Columns:    LocationsColumns,
		PrimaryKey: []*schema.Column{LocationsColumns[0]},
		ForeignKeys: []*schema.ForeignKey{
			{
				Symbol:     "locations_vendor_locations",
				Columns:    []*schema.Column{LocationsColumns[7]},
				RefColumns: []*schema.Column{VendorColumns[0]},
				OnDelete:   schema.SetNull,
			},
		},
	}
	// VendorColumns holds the columns for the "vendor" table.
	VendorColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt, Increment: true},
		{Name: "keycloakid", Type: field.TypeString},
		{Name: "urlid", Type: field.TypeString},
		{Name: "licenseid", Type: field.TypeString, Default: "unknown"},
		{Name: "firstname", Type: field.TypeString, Default: "unknown"},
		{Name: "lastname", Type: field.TypeString, Default: "unknown"},
		{Name: "email", Type: field.TypeString, Default: "@augustina.cc"},
		{Name: "lastpayout", Type: field.TypeTime},
		{Name: "isdisabled", Type: field.TypeBool, Default: false},
		{Name: "language", Type: field.TypeString},
		{Name: "telephone", Type: field.TypeString},
		{Name: "registrationdate", Type: field.TypeString},
		{Name: "vendorsince", Type: field.TypeString},
		{Name: "onlinemap", Type: field.TypeBool, Default: false},
		{Name: "hassmartphone", Type: field.TypeBool, Default: false},
		{Name: "hasbankaccount", Type: field.TypeBool, Default: false},
		{Name: "isdeleted", Type: field.TypeBool, Default: false},
		{Name: "accountproofurl", Type: field.TypeString},
		{Name: "debt", Type: field.TypeString},
	}
	// VendorTable holds the schema information for the "vendor" table.
	VendorTable = &schema.Table{
		Name:       "vendor",
		Columns:    VendorColumns,
		PrimaryKey: []*schema.Column{VendorColumns[0]},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		CommentsTable,
		LocationsTable,
		VendorTable,
	}
)

func init() {
	CommentsTable.ForeignKeys[0].RefTable = VendorTable
	LocationsTable.ForeignKeys[0].RefTable = VendorTable
	VendorTable.Annotation = &entsql.Annotation{
		Table: "vendor",
	}
}

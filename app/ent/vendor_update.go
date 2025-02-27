// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/augustin-wien/augustina-backend/ent/comment"
	"github.com/augustin-wien/augustina-backend/ent/location"
	"github.com/augustin-wien/augustina-backend/ent/predicate"
	"github.com/augustin-wien/augustina-backend/ent/vendor"
)

// VendorUpdate is the builder for updating Vendor entities.
type VendorUpdate struct {
	config
	hooks    []Hook
	mutation *VendorMutation
}

// Where appends a list predicates to the VendorUpdate builder.
func (vu *VendorUpdate) Where(ps ...predicate.Vendor) *VendorUpdate {
	vu.mutation.Where(ps...)
	return vu
}

// SetKeycloakid sets the "keycloakid" field.
func (vu *VendorUpdate) SetKeycloakid(s string) *VendorUpdate {
	vu.mutation.SetKeycloakid(s)
	return vu
}

// SetNillableKeycloakid sets the "keycloakid" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableKeycloakid(s *string) *VendorUpdate {
	if s != nil {
		vu.SetKeycloakid(*s)
	}
	return vu
}

// SetUrlid sets the "urlid" field.
func (vu *VendorUpdate) SetUrlid(s string) *VendorUpdate {
	vu.mutation.SetUrlid(s)
	return vu
}

// SetNillableUrlid sets the "urlid" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableUrlid(s *string) *VendorUpdate {
	if s != nil {
		vu.SetUrlid(*s)
	}
	return vu
}

// SetLicenseid sets the "licenseid" field.
func (vu *VendorUpdate) SetLicenseid(s string) *VendorUpdate {
	vu.mutation.SetLicenseid(s)
	return vu
}

// SetNillableLicenseid sets the "licenseid" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableLicenseid(s *string) *VendorUpdate {
	if s != nil {
		vu.SetLicenseid(*s)
	}
	return vu
}

// SetFirstname sets the "firstname" field.
func (vu *VendorUpdate) SetFirstname(s string) *VendorUpdate {
	vu.mutation.SetFirstname(s)
	return vu
}

// SetNillableFirstname sets the "firstname" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableFirstname(s *string) *VendorUpdate {
	if s != nil {
		vu.SetFirstname(*s)
	}
	return vu
}

// SetLastname sets the "lastname" field.
func (vu *VendorUpdate) SetLastname(s string) *VendorUpdate {
	vu.mutation.SetLastname(s)
	return vu
}

// SetNillableLastname sets the "lastname" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableLastname(s *string) *VendorUpdate {
	if s != nil {
		vu.SetLastname(*s)
	}
	return vu
}

// SetEmail sets the "email" field.
func (vu *VendorUpdate) SetEmail(s string) *VendorUpdate {
	vu.mutation.SetEmail(s)
	return vu
}

// SetNillableEmail sets the "email" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableEmail(s *string) *VendorUpdate {
	if s != nil {
		vu.SetEmail(*s)
	}
	return vu
}

// SetLastpayout sets the "lastpayout" field.
func (vu *VendorUpdate) SetLastpayout(t time.Time) *VendorUpdate {
	vu.mutation.SetLastpayout(t)
	return vu
}

// SetNillableLastpayout sets the "lastpayout" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableLastpayout(t *time.Time) *VendorUpdate {
	if t != nil {
		vu.SetLastpayout(*t)
	}
	return vu
}

// SetIsdisabled sets the "isdisabled" field.
func (vu *VendorUpdate) SetIsdisabled(b bool) *VendorUpdate {
	vu.mutation.SetIsdisabled(b)
	return vu
}

// SetNillableIsdisabled sets the "isdisabled" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableIsdisabled(b *bool) *VendorUpdate {
	if b != nil {
		vu.SetIsdisabled(*b)
	}
	return vu
}

// SetLanguage sets the "language" field.
func (vu *VendorUpdate) SetLanguage(s string) *VendorUpdate {
	vu.mutation.SetLanguage(s)
	return vu
}

// SetNillableLanguage sets the "language" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableLanguage(s *string) *VendorUpdate {
	if s != nil {
		vu.SetLanguage(*s)
	}
	return vu
}

// SetTelephone sets the "telephone" field.
func (vu *VendorUpdate) SetTelephone(s string) *VendorUpdate {
	vu.mutation.SetTelephone(s)
	return vu
}

// SetNillableTelephone sets the "telephone" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableTelephone(s *string) *VendorUpdate {
	if s != nil {
		vu.SetTelephone(*s)
	}
	return vu
}

// SetRegistrationdate sets the "registrationdate" field.
func (vu *VendorUpdate) SetRegistrationdate(s string) *VendorUpdate {
	vu.mutation.SetRegistrationdate(s)
	return vu
}

// SetNillableRegistrationdate sets the "registrationdate" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableRegistrationdate(s *string) *VendorUpdate {
	if s != nil {
		vu.SetRegistrationdate(*s)
	}
	return vu
}

// SetVendorsince sets the "vendorsince" field.
func (vu *VendorUpdate) SetVendorsince(s string) *VendorUpdate {
	vu.mutation.SetVendorsince(s)
	return vu
}

// SetNillableVendorsince sets the "vendorsince" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableVendorsince(s *string) *VendorUpdate {
	if s != nil {
		vu.SetVendorsince(*s)
	}
	return vu
}

// SetOnlinemap sets the "onlinemap" field.
func (vu *VendorUpdate) SetOnlinemap(b bool) *VendorUpdate {
	vu.mutation.SetOnlinemap(b)
	return vu
}

// SetNillableOnlinemap sets the "onlinemap" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableOnlinemap(b *bool) *VendorUpdate {
	if b != nil {
		vu.SetOnlinemap(*b)
	}
	return vu
}

// SetHassmartphone sets the "hassmartphone" field.
func (vu *VendorUpdate) SetHassmartphone(b bool) *VendorUpdate {
	vu.mutation.SetHassmartphone(b)
	return vu
}

// SetNillableHassmartphone sets the "hassmartphone" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableHassmartphone(b *bool) *VendorUpdate {
	if b != nil {
		vu.SetHassmartphone(*b)
	}
	return vu
}

// SetHasbankaccount sets the "hasbankaccount" field.
func (vu *VendorUpdate) SetHasbankaccount(b bool) *VendorUpdate {
	vu.mutation.SetHasbankaccount(b)
	return vu
}

// SetNillableHasbankaccount sets the "hasbankaccount" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableHasbankaccount(b *bool) *VendorUpdate {
	if b != nil {
		vu.SetHasbankaccount(*b)
	}
	return vu
}

// SetIsdeleted sets the "isdeleted" field.
func (vu *VendorUpdate) SetIsdeleted(b bool) *VendorUpdate {
	vu.mutation.SetIsdeleted(b)
	return vu
}

// SetNillableIsdeleted sets the "isdeleted" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableIsdeleted(b *bool) *VendorUpdate {
	if b != nil {
		vu.SetIsdeleted(*b)
	}
	return vu
}

// SetAccountproofurl sets the "accountproofurl" field.
func (vu *VendorUpdate) SetAccountproofurl(s string) *VendorUpdate {
	vu.mutation.SetAccountproofurl(s)
	return vu
}

// SetNillableAccountproofurl sets the "accountproofurl" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableAccountproofurl(s *string) *VendorUpdate {
	if s != nil {
		vu.SetAccountproofurl(*s)
	}
	return vu
}

// SetDebt sets the "debt" field.
func (vu *VendorUpdate) SetDebt(s string) *VendorUpdate {
	vu.mutation.SetDebt(s)
	return vu
}

// SetNillableDebt sets the "debt" field if the given value is not nil.
func (vu *VendorUpdate) SetNillableDebt(s *string) *VendorUpdate {
	if s != nil {
		vu.SetDebt(*s)
	}
	return vu
}

// AddLocationIDs adds the "locations" edge to the Location entity by IDs.
func (vu *VendorUpdate) AddLocationIDs(ids ...int) *VendorUpdate {
	vu.mutation.AddLocationIDs(ids...)
	return vu
}

// AddLocations adds the "locations" edges to the Location entity.
func (vu *VendorUpdate) AddLocations(l ...*Location) *VendorUpdate {
	ids := make([]int, len(l))
	for i := range l {
		ids[i] = l[i].ID
	}
	return vu.AddLocationIDs(ids...)
}

// AddCommentIDs adds the "comments" edge to the Comment entity by IDs.
func (vu *VendorUpdate) AddCommentIDs(ids ...int) *VendorUpdate {
	vu.mutation.AddCommentIDs(ids...)
	return vu
}

// AddComments adds the "comments" edges to the Comment entity.
func (vu *VendorUpdate) AddComments(c ...*Comment) *VendorUpdate {
	ids := make([]int, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return vu.AddCommentIDs(ids...)
}

// Mutation returns the VendorMutation object of the builder.
func (vu *VendorUpdate) Mutation() *VendorMutation {
	return vu.mutation
}

// ClearLocations clears all "locations" edges to the Location entity.
func (vu *VendorUpdate) ClearLocations() *VendorUpdate {
	vu.mutation.ClearLocations()
	return vu
}

// RemoveLocationIDs removes the "locations" edge to Location entities by IDs.
func (vu *VendorUpdate) RemoveLocationIDs(ids ...int) *VendorUpdate {
	vu.mutation.RemoveLocationIDs(ids...)
	return vu
}

// RemoveLocations removes "locations" edges to Location entities.
func (vu *VendorUpdate) RemoveLocations(l ...*Location) *VendorUpdate {
	ids := make([]int, len(l))
	for i := range l {
		ids[i] = l[i].ID
	}
	return vu.RemoveLocationIDs(ids...)
}

// ClearComments clears all "comments" edges to the Comment entity.
func (vu *VendorUpdate) ClearComments() *VendorUpdate {
	vu.mutation.ClearComments()
	return vu
}

// RemoveCommentIDs removes the "comments" edge to Comment entities by IDs.
func (vu *VendorUpdate) RemoveCommentIDs(ids ...int) *VendorUpdate {
	vu.mutation.RemoveCommentIDs(ids...)
	return vu
}

// RemoveComments removes "comments" edges to Comment entities.
func (vu *VendorUpdate) RemoveComments(c ...*Comment) *VendorUpdate {
	ids := make([]int, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return vu.RemoveCommentIDs(ids...)
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (vu *VendorUpdate) Save(ctx context.Context) (int, error) {
	return withHooks(ctx, vu.sqlSave, vu.mutation, vu.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (vu *VendorUpdate) SaveX(ctx context.Context) int {
	affected, err := vu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (vu *VendorUpdate) Exec(ctx context.Context) error {
	_, err := vu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (vu *VendorUpdate) ExecX(ctx context.Context) {
	if err := vu.Exec(ctx); err != nil {
		panic(err)
	}
}

func (vu *VendorUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := sqlgraph.NewUpdateSpec(vendor.Table, vendor.Columns, sqlgraph.NewFieldSpec(vendor.FieldID, field.TypeInt))
	if ps := vu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := vu.mutation.Keycloakid(); ok {
		_spec.SetField(vendor.FieldKeycloakid, field.TypeString, value)
	}
	if value, ok := vu.mutation.Urlid(); ok {
		_spec.SetField(vendor.FieldUrlid, field.TypeString, value)
	}
	if value, ok := vu.mutation.Licenseid(); ok {
		_spec.SetField(vendor.FieldLicenseid, field.TypeString, value)
	}
	if value, ok := vu.mutation.Firstname(); ok {
		_spec.SetField(vendor.FieldFirstname, field.TypeString, value)
	}
	if value, ok := vu.mutation.Lastname(); ok {
		_spec.SetField(vendor.FieldLastname, field.TypeString, value)
	}
	if value, ok := vu.mutation.Email(); ok {
		_spec.SetField(vendor.FieldEmail, field.TypeString, value)
	}
	if value, ok := vu.mutation.Lastpayout(); ok {
		_spec.SetField(vendor.FieldLastpayout, field.TypeTime, value)
	}
	if value, ok := vu.mutation.Isdisabled(); ok {
		_spec.SetField(vendor.FieldIsdisabled, field.TypeBool, value)
	}
	if value, ok := vu.mutation.Language(); ok {
		_spec.SetField(vendor.FieldLanguage, field.TypeString, value)
	}
	if value, ok := vu.mutation.Telephone(); ok {
		_spec.SetField(vendor.FieldTelephone, field.TypeString, value)
	}
	if value, ok := vu.mutation.Registrationdate(); ok {
		_spec.SetField(vendor.FieldRegistrationdate, field.TypeString, value)
	}
	if value, ok := vu.mutation.Vendorsince(); ok {
		_spec.SetField(vendor.FieldVendorsince, field.TypeString, value)
	}
	if value, ok := vu.mutation.Onlinemap(); ok {
		_spec.SetField(vendor.FieldOnlinemap, field.TypeBool, value)
	}
	if value, ok := vu.mutation.Hassmartphone(); ok {
		_spec.SetField(vendor.FieldHassmartphone, field.TypeBool, value)
	}
	if value, ok := vu.mutation.Hasbankaccount(); ok {
		_spec.SetField(vendor.FieldHasbankaccount, field.TypeBool, value)
	}
	if value, ok := vu.mutation.Isdeleted(); ok {
		_spec.SetField(vendor.FieldIsdeleted, field.TypeBool, value)
	}
	if value, ok := vu.mutation.Accountproofurl(); ok {
		_spec.SetField(vendor.FieldAccountproofurl, field.TypeString, value)
	}
	if value, ok := vu.mutation.Debt(); ok {
		_spec.SetField(vendor.FieldDebt, field.TypeString, value)
	}
	if vu.mutation.LocationsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.LocationsTable,
			Columns: []string{vendor.LocationsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.RemovedLocationsIDs(); len(nodes) > 0 && !vu.mutation.LocationsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.LocationsTable,
			Columns: []string{vendor.LocationsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.LocationsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.LocationsTable,
			Columns: []string{vendor.LocationsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vu.mutation.CommentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.CommentsTable,
			Columns: []string{vendor.CommentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(comment.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.RemovedCommentsIDs(); len(nodes) > 0 && !vu.mutation.CommentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.CommentsTable,
			Columns: []string{vendor.CommentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(comment.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.CommentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.CommentsTable,
			Columns: []string{vendor.CommentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(comment.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, vu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{vendor.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return 0, err
	}
	vu.mutation.done = true
	return n, nil
}

// VendorUpdateOne is the builder for updating a single Vendor entity.
type VendorUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *VendorMutation
}

// SetKeycloakid sets the "keycloakid" field.
func (vuo *VendorUpdateOne) SetKeycloakid(s string) *VendorUpdateOne {
	vuo.mutation.SetKeycloakid(s)
	return vuo
}

// SetNillableKeycloakid sets the "keycloakid" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableKeycloakid(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetKeycloakid(*s)
	}
	return vuo
}

// SetUrlid sets the "urlid" field.
func (vuo *VendorUpdateOne) SetUrlid(s string) *VendorUpdateOne {
	vuo.mutation.SetUrlid(s)
	return vuo
}

// SetNillableUrlid sets the "urlid" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableUrlid(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetUrlid(*s)
	}
	return vuo
}

// SetLicenseid sets the "licenseid" field.
func (vuo *VendorUpdateOne) SetLicenseid(s string) *VendorUpdateOne {
	vuo.mutation.SetLicenseid(s)
	return vuo
}

// SetNillableLicenseid sets the "licenseid" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableLicenseid(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetLicenseid(*s)
	}
	return vuo
}

// SetFirstname sets the "firstname" field.
func (vuo *VendorUpdateOne) SetFirstname(s string) *VendorUpdateOne {
	vuo.mutation.SetFirstname(s)
	return vuo
}

// SetNillableFirstname sets the "firstname" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableFirstname(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetFirstname(*s)
	}
	return vuo
}

// SetLastname sets the "lastname" field.
func (vuo *VendorUpdateOne) SetLastname(s string) *VendorUpdateOne {
	vuo.mutation.SetLastname(s)
	return vuo
}

// SetNillableLastname sets the "lastname" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableLastname(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetLastname(*s)
	}
	return vuo
}

// SetEmail sets the "email" field.
func (vuo *VendorUpdateOne) SetEmail(s string) *VendorUpdateOne {
	vuo.mutation.SetEmail(s)
	return vuo
}

// SetNillableEmail sets the "email" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableEmail(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetEmail(*s)
	}
	return vuo
}

// SetLastpayout sets the "lastpayout" field.
func (vuo *VendorUpdateOne) SetLastpayout(t time.Time) *VendorUpdateOne {
	vuo.mutation.SetLastpayout(t)
	return vuo
}

// SetNillableLastpayout sets the "lastpayout" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableLastpayout(t *time.Time) *VendorUpdateOne {
	if t != nil {
		vuo.SetLastpayout(*t)
	}
	return vuo
}

// SetIsdisabled sets the "isdisabled" field.
func (vuo *VendorUpdateOne) SetIsdisabled(b bool) *VendorUpdateOne {
	vuo.mutation.SetIsdisabled(b)
	return vuo
}

// SetNillableIsdisabled sets the "isdisabled" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableIsdisabled(b *bool) *VendorUpdateOne {
	if b != nil {
		vuo.SetIsdisabled(*b)
	}
	return vuo
}

// SetLanguage sets the "language" field.
func (vuo *VendorUpdateOne) SetLanguage(s string) *VendorUpdateOne {
	vuo.mutation.SetLanguage(s)
	return vuo
}

// SetNillableLanguage sets the "language" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableLanguage(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetLanguage(*s)
	}
	return vuo
}

// SetTelephone sets the "telephone" field.
func (vuo *VendorUpdateOne) SetTelephone(s string) *VendorUpdateOne {
	vuo.mutation.SetTelephone(s)
	return vuo
}

// SetNillableTelephone sets the "telephone" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableTelephone(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetTelephone(*s)
	}
	return vuo
}

// SetRegistrationdate sets the "registrationdate" field.
func (vuo *VendorUpdateOne) SetRegistrationdate(s string) *VendorUpdateOne {
	vuo.mutation.SetRegistrationdate(s)
	return vuo
}

// SetNillableRegistrationdate sets the "registrationdate" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableRegistrationdate(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetRegistrationdate(*s)
	}
	return vuo
}

// SetVendorsince sets the "vendorsince" field.
func (vuo *VendorUpdateOne) SetVendorsince(s string) *VendorUpdateOne {
	vuo.mutation.SetVendorsince(s)
	return vuo
}

// SetNillableVendorsince sets the "vendorsince" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableVendorsince(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetVendorsince(*s)
	}
	return vuo
}

// SetOnlinemap sets the "onlinemap" field.
func (vuo *VendorUpdateOne) SetOnlinemap(b bool) *VendorUpdateOne {
	vuo.mutation.SetOnlinemap(b)
	return vuo
}

// SetNillableOnlinemap sets the "onlinemap" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableOnlinemap(b *bool) *VendorUpdateOne {
	if b != nil {
		vuo.SetOnlinemap(*b)
	}
	return vuo
}

// SetHassmartphone sets the "hassmartphone" field.
func (vuo *VendorUpdateOne) SetHassmartphone(b bool) *VendorUpdateOne {
	vuo.mutation.SetHassmartphone(b)
	return vuo
}

// SetNillableHassmartphone sets the "hassmartphone" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableHassmartphone(b *bool) *VendorUpdateOne {
	if b != nil {
		vuo.SetHassmartphone(*b)
	}
	return vuo
}

// SetHasbankaccount sets the "hasbankaccount" field.
func (vuo *VendorUpdateOne) SetHasbankaccount(b bool) *VendorUpdateOne {
	vuo.mutation.SetHasbankaccount(b)
	return vuo
}

// SetNillableHasbankaccount sets the "hasbankaccount" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableHasbankaccount(b *bool) *VendorUpdateOne {
	if b != nil {
		vuo.SetHasbankaccount(*b)
	}
	return vuo
}

// SetIsdeleted sets the "isdeleted" field.
func (vuo *VendorUpdateOne) SetIsdeleted(b bool) *VendorUpdateOne {
	vuo.mutation.SetIsdeleted(b)
	return vuo
}

// SetNillableIsdeleted sets the "isdeleted" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableIsdeleted(b *bool) *VendorUpdateOne {
	if b != nil {
		vuo.SetIsdeleted(*b)
	}
	return vuo
}

// SetAccountproofurl sets the "accountproofurl" field.
func (vuo *VendorUpdateOne) SetAccountproofurl(s string) *VendorUpdateOne {
	vuo.mutation.SetAccountproofurl(s)
	return vuo
}

// SetNillableAccountproofurl sets the "accountproofurl" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableAccountproofurl(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetAccountproofurl(*s)
	}
	return vuo
}

// SetDebt sets the "debt" field.
func (vuo *VendorUpdateOne) SetDebt(s string) *VendorUpdateOne {
	vuo.mutation.SetDebt(s)
	return vuo
}

// SetNillableDebt sets the "debt" field if the given value is not nil.
func (vuo *VendorUpdateOne) SetNillableDebt(s *string) *VendorUpdateOne {
	if s != nil {
		vuo.SetDebt(*s)
	}
	return vuo
}

// AddLocationIDs adds the "locations" edge to the Location entity by IDs.
func (vuo *VendorUpdateOne) AddLocationIDs(ids ...int) *VendorUpdateOne {
	vuo.mutation.AddLocationIDs(ids...)
	return vuo
}

// AddLocations adds the "locations" edges to the Location entity.
func (vuo *VendorUpdateOne) AddLocations(l ...*Location) *VendorUpdateOne {
	ids := make([]int, len(l))
	for i := range l {
		ids[i] = l[i].ID
	}
	return vuo.AddLocationIDs(ids...)
}

// AddCommentIDs adds the "comments" edge to the Comment entity by IDs.
func (vuo *VendorUpdateOne) AddCommentIDs(ids ...int) *VendorUpdateOne {
	vuo.mutation.AddCommentIDs(ids...)
	return vuo
}

// AddComments adds the "comments" edges to the Comment entity.
func (vuo *VendorUpdateOne) AddComments(c ...*Comment) *VendorUpdateOne {
	ids := make([]int, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return vuo.AddCommentIDs(ids...)
}

// Mutation returns the VendorMutation object of the builder.
func (vuo *VendorUpdateOne) Mutation() *VendorMutation {
	return vuo.mutation
}

// ClearLocations clears all "locations" edges to the Location entity.
func (vuo *VendorUpdateOne) ClearLocations() *VendorUpdateOne {
	vuo.mutation.ClearLocations()
	return vuo
}

// RemoveLocationIDs removes the "locations" edge to Location entities by IDs.
func (vuo *VendorUpdateOne) RemoveLocationIDs(ids ...int) *VendorUpdateOne {
	vuo.mutation.RemoveLocationIDs(ids...)
	return vuo
}

// RemoveLocations removes "locations" edges to Location entities.
func (vuo *VendorUpdateOne) RemoveLocations(l ...*Location) *VendorUpdateOne {
	ids := make([]int, len(l))
	for i := range l {
		ids[i] = l[i].ID
	}
	return vuo.RemoveLocationIDs(ids...)
}

// ClearComments clears all "comments" edges to the Comment entity.
func (vuo *VendorUpdateOne) ClearComments() *VendorUpdateOne {
	vuo.mutation.ClearComments()
	return vuo
}

// RemoveCommentIDs removes the "comments" edge to Comment entities by IDs.
func (vuo *VendorUpdateOne) RemoveCommentIDs(ids ...int) *VendorUpdateOne {
	vuo.mutation.RemoveCommentIDs(ids...)
	return vuo
}

// RemoveComments removes "comments" edges to Comment entities.
func (vuo *VendorUpdateOne) RemoveComments(c ...*Comment) *VendorUpdateOne {
	ids := make([]int, len(c))
	for i := range c {
		ids[i] = c[i].ID
	}
	return vuo.RemoveCommentIDs(ids...)
}

// Where appends a list predicates to the VendorUpdate builder.
func (vuo *VendorUpdateOne) Where(ps ...predicate.Vendor) *VendorUpdateOne {
	vuo.mutation.Where(ps...)
	return vuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (vuo *VendorUpdateOne) Select(field string, fields ...string) *VendorUpdateOne {
	vuo.fields = append([]string{field}, fields...)
	return vuo
}

// Save executes the query and returns the updated Vendor entity.
func (vuo *VendorUpdateOne) Save(ctx context.Context) (*Vendor, error) {
	return withHooks(ctx, vuo.sqlSave, vuo.mutation, vuo.hooks)
}

// SaveX is like Save, but panics if an error occurs.
func (vuo *VendorUpdateOne) SaveX(ctx context.Context) *Vendor {
	node, err := vuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (vuo *VendorUpdateOne) Exec(ctx context.Context) error {
	_, err := vuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (vuo *VendorUpdateOne) ExecX(ctx context.Context) {
	if err := vuo.Exec(ctx); err != nil {
		panic(err)
	}
}

func (vuo *VendorUpdateOne) sqlSave(ctx context.Context) (_node *Vendor, err error) {
	_spec := sqlgraph.NewUpdateSpec(vendor.Table, vendor.Columns, sqlgraph.NewFieldSpec(vendor.FieldID, field.TypeInt))
	id, ok := vuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Vendor.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := vuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, vendor.FieldID)
		for _, f := range fields {
			if !vendor.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != vendor.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := vuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := vuo.mutation.Keycloakid(); ok {
		_spec.SetField(vendor.FieldKeycloakid, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Urlid(); ok {
		_spec.SetField(vendor.FieldUrlid, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Licenseid(); ok {
		_spec.SetField(vendor.FieldLicenseid, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Firstname(); ok {
		_spec.SetField(vendor.FieldFirstname, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Lastname(); ok {
		_spec.SetField(vendor.FieldLastname, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Email(); ok {
		_spec.SetField(vendor.FieldEmail, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Lastpayout(); ok {
		_spec.SetField(vendor.FieldLastpayout, field.TypeTime, value)
	}
	if value, ok := vuo.mutation.Isdisabled(); ok {
		_spec.SetField(vendor.FieldIsdisabled, field.TypeBool, value)
	}
	if value, ok := vuo.mutation.Language(); ok {
		_spec.SetField(vendor.FieldLanguage, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Telephone(); ok {
		_spec.SetField(vendor.FieldTelephone, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Registrationdate(); ok {
		_spec.SetField(vendor.FieldRegistrationdate, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Vendorsince(); ok {
		_spec.SetField(vendor.FieldVendorsince, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Onlinemap(); ok {
		_spec.SetField(vendor.FieldOnlinemap, field.TypeBool, value)
	}
	if value, ok := vuo.mutation.Hassmartphone(); ok {
		_spec.SetField(vendor.FieldHassmartphone, field.TypeBool, value)
	}
	if value, ok := vuo.mutation.Hasbankaccount(); ok {
		_spec.SetField(vendor.FieldHasbankaccount, field.TypeBool, value)
	}
	if value, ok := vuo.mutation.Isdeleted(); ok {
		_spec.SetField(vendor.FieldIsdeleted, field.TypeBool, value)
	}
	if value, ok := vuo.mutation.Accountproofurl(); ok {
		_spec.SetField(vendor.FieldAccountproofurl, field.TypeString, value)
	}
	if value, ok := vuo.mutation.Debt(); ok {
		_spec.SetField(vendor.FieldDebt, field.TypeString, value)
	}
	if vuo.mutation.LocationsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.LocationsTable,
			Columns: []string{vendor.LocationsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.RemovedLocationsIDs(); len(nodes) > 0 && !vuo.mutation.LocationsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.LocationsTable,
			Columns: []string{vendor.LocationsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.LocationsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.LocationsTable,
			Columns: []string{vendor.LocationsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(location.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vuo.mutation.CommentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.CommentsTable,
			Columns: []string{vendor.CommentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(comment.FieldID, field.TypeInt),
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.RemovedCommentsIDs(); len(nodes) > 0 && !vuo.mutation.CommentsCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.CommentsTable,
			Columns: []string{vendor.CommentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(comment.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.CommentsIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2M,
			Inverse: false,
			Table:   vendor.CommentsTable,
			Columns: []string{vendor.CommentsColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(comment.FieldID, field.TypeInt),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Vendor{config: vuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, vuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{vendor.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	vuo.mutation.done = true
	return _node, nil
}

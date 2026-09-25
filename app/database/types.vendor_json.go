package database

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"gopkg.in/guregu/null.v4"
)

var nullStringType = reflect.TypeOf(null.String{})

// UnmarshalJSON accepts `false` for the vendor's text fields and treats it as empty.
// Odoo serializes unset char fields as `false`, which would otherwise make the whole
// update fail with "cannot unmarshal bool into Go struct field Vendor.Language".
func (v *Vendor) UnmarshalJSON(data []byte) error {
	type vendorAlias Vendor

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// Not an object: let the default decoder produce its usual error
		return json.Unmarshal(data, (*vendorAlias)(v))
	}

	changed := false
	vendorType := reflect.TypeOf(Vendor{})
	for key, value := range raw {
		if !bytes.Equal(bytes.TrimSpace(value), []byte("false")) {
			continue
		}
		// encoding/json matches keys case-insensitively, so do the same here
		field, ok := vendorType.FieldByNameFunc(func(name string) bool { return strings.EqualFold(name, key) })
		if !ok {
			continue
		}
		switch {
		case field.Type.Kind() == reflect.String:
			raw[key] = json.RawMessage(`""`)
			changed = true
		case field.Type == nullStringType:
			raw[key] = json.RawMessage(`null`)
			changed = true
		}
	}

	if changed {
		var err error
		if data, err = json.Marshal(raw); err != nil {
			return err
		}
	}
	return json.Unmarshal(data, (*vendorAlias)(v))
}

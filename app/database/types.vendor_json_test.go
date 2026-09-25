package database

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVendorUnmarshalJSONAcceptsFalseForTextFields(t *testing.T) {
	payload := `{
		"ID": 7,
		"FirstName": "Anna",
		"language": false,
		"Telephone": false,
		"Debt": false,
		"LicenseID": false,
		"AccountProofUrl": false,
		"OnlineMap": false,
		"HasBankAccount": true
	}`

	var vendor Vendor
	require.NoError(t, json.Unmarshal([]byte(payload), &vendor))

	require.Equal(t, 7, vendor.ID)
	require.Equal(t, "Anna", vendor.FirstName)
	require.Equal(t, "", vendor.Language)
	require.Equal(t, "", vendor.Telephone)
	require.Equal(t, "", vendor.Debt)
	require.False(t, vendor.LicenseID.Valid)
	require.False(t, vendor.AccountProofUrl.Valid)
	require.False(t, vendor.OnlineMap)
	require.True(t, vendor.HasBankAccount)
}

func TestVendorUnmarshalJSONStillRejectsTrueForTextFields(t *testing.T) {
	var vendor Vendor
	err := json.Unmarshal([]byte(`{"Language": true}`), &vendor)
	require.Error(t, err)
}

func TestVendorUnmarshalJSONRoundTrip(t *testing.T) {
	var vendor Vendor
	require.NoError(t, json.Unmarshal([]byte(`{"FirstName":"Anna","Language":"deutsch","IsBlocked":true,"BlockedNote":"x"}`), &vendor))

	encoded, err := json.Marshal(vendor)
	require.NoError(t, err)

	var decoded Vendor
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, vendor, decoded)
}

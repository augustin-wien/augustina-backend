package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateItemNormalParsesFields(t *testing.T) {
	item, err := updateItemNormal(map[string][]string{
		"Name":          {"Zeitung"},
		"Description":   {"Die neue Ausgabe"},
		"Price":         {"300"},
		"IsLicenseItem": {"false"},
		"IsPDFItem":     {"true"},
		"Archived":      {"false"},
		"Disabled":      {"true"},
		"LicenseItem":   {"7"},
		"ID":            {"3"},
		"ItemColor":     {"#fff"},
		"ItemTextColor": {"#000"},
		"PDF":           {"9"},
		"LicenseGroup":  {"digital_edition"},
		"ItemOrder":     {"4"},
		"Type":          {"issue"},
	})
	require.NoError(t, err)
	require.Equal(t, "Zeitung", item.Name)
	require.Equal(t, 300, item.Price)
	require.True(t, item.IsPDFItem)
	require.True(t, item.Disabled)
	require.EqualValues(t, 7, item.LicenseItem.Int64)
	require.Equal(t, 3, item.ID)
	require.Equal(t, "#fff", item.ItemColor.String)
	require.Equal(t, "#000", item.ItemTextColor.String)
	require.EqualValues(t, 9, item.PDF.Int64)
	require.Equal(t, "digital_edition", item.LicenseGroup.String)
	require.Equal(t, 4, item.ItemOrder)
	require.Equal(t, "issue", item.Type)

	for _, key := range []string{"Price", "IsLicenseItem", "IsPDFItem", "Archived", "Disabled", "LicenseItem", "ID", "PDF"} {
		t.Run(key, func(t *testing.T) {
			_, err := updateItemNormal(map[string][]string{key: {"x"}})
			require.Error(t, err)
		})
	}
}

func TestParseLicenseCost(t *testing.T) {
	cost, err := parseLicenseCost(map[string][]string{"license_cost": {"150"}})
	require.NoError(t, err)
	require.Equal(t, 150, cost)

	cost, err = parseLicenseCost(map[string][]string{"LicenseCost": {"80"}})
	require.NoError(t, err)
	require.Equal(t, 80, cost)

	for _, fields := range []map[string][]string{
		{},
		{"license_cost": {""}},
		{"license_cost": {"abc"}},
		{"license_cost": {"0"}},
		{"license_cost": {"-5"}},
	} {
		_, err := parseLicenseCost(fields)
		require.Error(t, err, fields)
	}
}

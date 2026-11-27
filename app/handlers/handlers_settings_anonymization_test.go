package handlers

import (
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsAnonymization(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	putSettings(t, map[string]string{"SalesAnonymizationEnabled": "maybe"}, nil, 400)
	putSettings(t, map[string]string{"VendorAnonymizationEnabled": "maybe"}, nil, 400)
	putSettings(t, map[string]string{"AnonymizationRetentionDays": "many"}, nil, 400)
	putSettings(t, map[string]string{"AnonymizationRetentionDays": "29"}, nil, 400)

	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	require.False(t, settings.SalesAnonymizationEnabled)
	require.False(t, settings.VendorAnonymizationEnabled)
	require.Equal(t, 90, settings.AnonymizationRetentionDays)

	putSettings(t, map[string]string{
		"SalesAnonymizationEnabled":  "true",
		"VendorAnonymizationEnabled": "true",
		"AnonymizationRetentionDays": "120",
	}, nil, 200)

	settings, err = database.Db.GetSettings()
	require.NoError(t, err)
	require.True(t, settings.SalesAnonymizationEnabled)
	require.True(t, settings.VendorAnonymizationEnabled)
	require.Equal(t, 120, settings.AnonymizationRetentionDays)
}

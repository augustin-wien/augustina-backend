package keycloak_test

import (
	"os"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func initKeycloakForTest(t *testing.T) *keycloak.Keycloak {
	t.Helper()
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		require.NoError(t, os.Chdir(".."))
	}
	config.InitConfig()
	config.Config.SMTPSenderAddress = ""
	require.NoError(t, keycloak.InitializeOauthServer())
	return &keycloak.KeycloakClient
}

func userGroupNames(t *testing.T, k *keycloak.Keycloak, userID string) []string {
	t.Helper()
	groups, err := k.GetUserGroups(userID)
	require.NoError(t, err)
	names := []string{}
	for _, g := range groups {
		if g.Name != nil {
			names = append(names, *g.Name)
		}
	}
	return names
}

func TestSyncLicenseGroupsDiffToKeycloak(t *testing.T) {
	k := initKeycloakForTest(t)

	suffix := utils.RandomString(6)
	keep := "test_sync_keep_" + suffix
	remove := "test_sync_remove_" + suffix
	add := "test_sync_add_" + suffix
	email := "license_sync_" + suffix + "@example.com"

	userID, created, err := k.GetOrCreateCustomerUser("", email, "Sync", "Test")
	require.NoError(t, err)
	require.True(t, created)
	t.Cleanup(func() { _ = k.DeleteUser(email) })

	// Found again by Keycloak ID and by email
	again, created, err := k.GetOrCreateCustomerUser(userID, "", "", "")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, userID, again)
	again, created, err = k.GetOrCreateCustomerUser("unknown-id", email, "", "")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, userID, again)
	_, _, err = k.GetOrCreateCustomerUser("", "", "", "")
	require.Error(t, err)

	require.NoError(t, k.AssignDigitalLicenseGroup(userID, keep))
	require.NoError(t, k.AssignDigitalLicenseGroup(userID, remove))
	require.Subset(t, userGroupNames(t, k, userID), []string{keep, remove})

	// A group that only exists in our database is removed without error
	require.NoError(t, k.SyncLicenseGroupsDiffToKeycloak(userID,
		[]string{keep, remove, "test_sync_never_in_keycloak_" + suffix, ""},
		[]string{keep, add, ""}))

	names := userGroupNames(t, k, userID)
	require.Contains(t, names, keep)
	require.Contains(t, names, add)
	require.NotContains(t, names, remove)

	require.Error(t, k.AssignDigitalLicenseGroup("", keep))
}

func TestVerifyEmailsAndLogin(t *testing.T) {
	k := initKeycloakForTest(t)

	email := "verify_" + utils.RandomString(6) + "@example.com"
	_, err := k.CreateUser(email, "V", "E", email, "pw")
	require.NoError(t, err)
	t.Cleanup(func() { _ = k.DeleteUser(email) })

	// Without an SMTP sender no mail is triggered, but the user must exist
	require.NoError(t, k.SendVerifyEmail(email))
	require.NoError(t, k.SendVerifyEmailVendor(email))
	require.Error(t, k.SendVerifyEmail("does_not_exist_"+utils.RandomString(6)+"@example.com"))

	_, err = k.Login("no-such-admin", "wrong")
	require.Error(t, err)
}

package database

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// mockAssigner records calls to AssignDigitalLicenseGroup for assertion in tests.
type mockAssigner struct {
	calls []assignCall
	err   error
}

type assignCall struct {
	userID       string
	licenseGroup string
}

func (m *mockAssigner) AssignDigitalLicenseGroup(userID, licenseGroup string) error {
	m.calls = append(m.calls, assignCall{userID, licenseGroup})
	return m.err
}

func newTestService(assigner DigitalLicenseAssigner) *AbonementService {
	return &AbonementService{db: &Db, keycloakAssigner: assigner}
}

func TestSyncAbonementLicensesToKeycloak_NoGroups(t *testing.T) {
	mock := &mockAssigner{}
	svc := newTestService(mock)

	err := svc.SyncAbonementLicensesToKeycloak("user-123", []string{})

	require.NoError(t, err)
	require.Empty(t, mock.calls)
}

func TestSyncAbonementLicensesToKeycloak_SkipsEmptyGroup(t *testing.T) {
	mock := &mockAssigner{}
	svc := newTestService(mock)

	err := svc.SyncAbonementLicensesToKeycloak("user-123", []string{"", "digital_edition"})

	require.NoError(t, err)
	require.Len(t, mock.calls, 1)
	require.Equal(t, "user-123", mock.calls[0].userID)
	require.Equal(t, "digital_edition", mock.calls[0].licenseGroup)
}

func TestSyncAbonementLicensesToKeycloak_MultipleGroups(t *testing.T) {
	mock := &mockAssigner{}
	svc := newTestService(mock)

	groups := []string{"digital_edition", "analog_edition"}
	err := svc.SyncAbonementLicensesToKeycloak("user-abc", groups)

	require.NoError(t, err)
	require.Len(t, mock.calls, 2)
	require.Equal(t, assignCall{"user-abc", "digital_edition"}, mock.calls[0])
	require.Equal(t, assignCall{"user-abc", "analog_edition"}, mock.calls[1])
}

func TestSyncAbonementLicensesToKeycloak_PropagatesError(t *testing.T) {
	mock := &mockAssigner{err: errors.New("keycloak unavailable")}
	svc := newTestService(mock)

	err := svc.SyncAbonementLicensesToKeycloak("user-xyz", []string{"digital_edition"})

	require.ErrorContains(t, err, "keycloak unavailable")
	require.Len(t, mock.calls, 1)
}

func TestProcessAbonementLicenseGroupsForDate_GrantsPublishedIssue(t *testing.T) {
	require.NoError(t, Db.InitEmptyTestDb())

	// The abonement product is no longer for sale (disabled); its subscribers
	// still get new issues
	aboItemID, _, err := Db.CreateAbonementItemWithLicense(Item{
		Name: "Jahresabo Veröffentlichung", Description: "Annual subscription", Price: 2400,
		LicenseGroup: null.StringFrom("abo_publication_test"),
	}, 50)
	require.NoError(t, err)

	newCustomer := func(email, keycloakID string) *Customer {
		c, err := Db.CreateCustomer(&Customer{Email: email, KeycloakID: keycloakID, LicenseGroups: []string{}})
		require.NoError(t, err)
		return c
	}
	active := newCustomer("active-abo@example.com", "kc-active")
	expired := newCustomer("expired-abo@example.com", "kc-expired")

	_, err = Db.CreateAbonement(&Abonement{
		CustomerID: active.ID, ItemID: aboItemID, Status: "active",
		FromDate: time.Now().AddDate(0, -1, 0), ToDate: time.Now().AddDate(0, 11, 0),
	})
	require.NoError(t, err)
	_, err = Db.CreateAbonement(&Abonement{
		CustomerID: expired.ID, ItemID: aboItemID, Status: "active",
		FromDate: time.Now().AddDate(-2, 0, 0), ToDate: time.Now().AddDate(-1, 0, 0),
	})
	require.NoError(t, err)

	mock := &mockAssigner{}
	require.NoError(t, newTestService(mock).ProcessAbonementLicenseGroupsForDate(time.Now(), "9628"))

	got, err := Db.GetCustomerByID(active.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"abo_publication_test", "9628"}, got.LicenseGroups)
	require.Contains(t, mock.calls, assignCall{"kc-active", "9628"})

	got, err = Db.GetCustomerByID(expired.ID)
	require.NoError(t, err)
	require.Empty(t, got.LicenseGroups)
	for _, call := range mock.calls {
		require.NotEqual(t, "kc-expired", call.userID)
	}
}

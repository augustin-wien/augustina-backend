package database

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
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

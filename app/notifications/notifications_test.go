package notifications

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationsWithoutServices(t *testing.T) {
	t.Setenv("NOTIFICATIONS_EMAIL_ENABLED", "false")

	client := InitNotifications()
	require.NotNil(t, client)
	require.Same(t, client, NotificationsClient.Client)

	// Without configured services sending is a no-op and must not panic
	NotificationsClient.SendNotification("subject", "message")
	NotificationsClient.SendErrorNotification("subject", "message")

	// A client without notify instance initialises one on first use
	n := &Notifications{}
	n.SendErrorNotification("subject", "message")
	require.NotNil(t, n.Client)

	var nilClient *Notifications
	require.NotPanics(t, func() { nilClient.SendErrorNotification("subject", "message") })
}

func TestInitNotificationsWithEmail(t *testing.T) {
	t.Setenv("NOTIFICATIONS_EMAIL_ENABLED", "true")
	t.Setenv("NOTIFICATIONS_EMAIL_SENDER", "alerts@example.com")
	t.Setenv("NOTIFICATIONS_EMAIL_SERVER", "127.0.0.1")
	t.Setenv("NOTIFICATIONS_EMAIL_PORT", "1")
	t.Setenv("NOTIFICATIONS_EMAIL_RECEIVER", "ops@example.com")

	client := InitNotifications()
	require.NotNil(t, client)
	// The mail server is unreachable; the error is printed, not returned
	NotificationsClient.SendNotification("subject", "message")
}

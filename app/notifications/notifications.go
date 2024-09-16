package notifications

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/matrix"
	"maunium.net/go/mautrix/id"
)

var NotificationsClient Notifications

type Notifications struct {
	Client        *notify.Notify
	SentryEnabled bool
}

func InitNotifications(enableSentry bool) *notify.Notify {
	client := notify.New()
	if os.Getenv("NOTIFICATIONS_EMAIL_ENABLED") == "true" {
		email := mail.New(os.Getenv("NOTIFICATIONS_EMAIL_SENDER"), os.Getenv("NOTIFICATIONS_EMAIL_SERVER")+":"+os.Getenv("NOTIFICATIONS_EMAIL_PORT"))
		email.AuthenticateSMTP(os.Getenv("NOTIFICATIONS_EMAIL_SENDER"), os.Getenv("NOTIFICATIONS_EMAIL_USER"), os.Getenv("NOTIFICATIONS_EMAIL_PASSWORD"), os.Getenv("NOTIFICATIONS_EMAIL_SERVER"))
		email.AddReceivers(os.Getenv("NOTIFICATIONS_EMAIL_RECEIVER"))
		client.UseServices(email)
	}

	if os.Getenv("NOTIFICATIONS_MATRIX_ENABLED") == "true" {
		user := id.NewUserID(os.Getenv("NOTIFICATIONS_MATRIX_USER_ID"), os.Getenv("NOTIFICATIONS_MATRIX_HOME_SERVER"))
		room := id.RoomID(os.Getenv("NOTIFICATIONS_MATRIX_ROOM_ID"))
		matrix, err := matrix.New(user, room, os.Getenv("NOTIFICATIONS_MATRIX_HOME_SERVER"), os.Getenv("NOTIFICATIONS_MATRIX_ACCESS_TOKEN"))
		if err != nil {
			fmt.Printf("Error: matrix.New() failed: %s", err.Error())
		}
		client.UseServices(matrix)
	}
	client.Disabled = false
	NotificationsClient.Client = client
	NotificationsClient.SentryEnabled = enableSentry
	return client
}

// SendNotification sends a notification
func (n *Notifications) SendNotification(subject, message string) {
	if n != nil && n.SentryEnabled {
		sentry.CaptureMessage(message)
	}
	prefix := "[" + os.Getenv("NOTIFICATIONS_EMAIL_PREFIX") + "] "
	err := n.Client.Send(context.Background(), prefix+subject, message)
	if err != nil {
		fmt.Println(err)
	}
}

// SendErrorNotification sends an error notification
func (n *Notifications) SendErrorNotification(subject, message string) {
	if n != nil && n.SentryEnabled {
		sentry.CaptureMessage(message)
	}
	prefix := "[" + os.Getenv("NOTIFICATIONS_PREFIX") + "-Error] "
	if n == nil {
		fmt.Printf("Error: NotificationsClient is nil")
		return
	}
	client := n.Client
	if client == nil {
		client = InitNotifications(n.SentryEnabled)
		n.Client = client
	}
	err := client.Send(context.Background(), prefix+subject, message)
	if err != nil {
		fmt.Println(err)
	}
}

// Sync syncs the notifications, needed for zap logger
func (n Notifications) Sync() error {
	// nothing to sync for notifications
	return nil
}

// Write receives the notifications from the zap logger
func (n Notifications) Write(p []byte) (l int, err error) {
	message := string(p)
	if strings.Contains(message, "ERROR") {
		go n.SendErrorNotification("Error", message)
	}
	return len(p), nil
}

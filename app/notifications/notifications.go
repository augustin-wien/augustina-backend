package notifications

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/mail"
)

var NotificationsClient Notifications
var notificationsMu sync.Mutex

type Notifications struct {
	Client *notify.Notify
}

// InitNotifications sets up the email notification client. Sentry reporting is handled
// separately by utils.GetLogger's alert core, not here.
func InitNotifications() *notify.Notify {
	// make initialization concurrent-safe
	notificationsMu.Lock()
	defer notificationsMu.Unlock()

	client := notify.New()
	if os.Getenv("NOTIFICATIONS_EMAIL_ENABLED") == "true" {
		email := mail.New(os.Getenv("NOTIFICATIONS_EMAIL_SENDER"), os.Getenv("NOTIFICATIONS_EMAIL_SERVER")+":"+os.Getenv("NOTIFICATIONS_EMAIL_PORT"))
		email.AuthenticateSMTP(os.Getenv("NOTIFICATIONS_EMAIL_SENDER"), os.Getenv("NOTIFICATIONS_EMAIL_USER"), os.Getenv("NOTIFICATIONS_EMAIL_PASSWORD"), os.Getenv("NOTIFICATIONS_EMAIL_SERVER"))
		email.AddReceivers(os.Getenv("NOTIFICATIONS_EMAIL_RECEIVER"))
		client.UseServices(email)
	}
	client.Disabled = false
	NotificationsClient.Client = client
	return client
}

// SendNotification sends a notification
func (n *Notifications) SendNotification(subject, message string) {
	prefix := "[" + os.Getenv("NOTIFICATIONS_EMAIL_PREFIX") + "] "
	err := n.Client.Send(context.Background(), prefix+subject, message)
	if err != nil {
		fmt.Println(err)
	}
}

// SendErrorNotification sends an error notification
func (n *Notifications) SendErrorNotification(subject, message string) {
	prefix := "[" + os.Getenv("NOTIFICATIONS_PREFIX") + "-Error] "
	if n == nil {
		fmt.Printf("Error: NotificationsClient is nil")
		return
	}
	client := n.Client
	if client == nil {
		client = InitNotifications()
		n.Client = client
	}
	err := client.Send(context.Background(), prefix+subject, message)
	if err != nil {
		fmt.Println(err)
	}
}

package main

import (
	"net/http"
	"time"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/handlers"
	"github.com/augustin-wien/augustina-backend/keycloak"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/notifications"
	"github.com/augustin-wien/augustina-backend/utils"

	"github.com/getsentry/sentry-go"
)

var log = utils.GetLogger()

func main() {
	// Initialize config
	config.InitConfig()
	conf := config.Config
	sentryEnabled := conf.SentryDSN != ""
	notifications.InitNotifications(sentryEnabled)

	log.Info("Starting Augustin Server v", conf.Version)

	// Initialize Keycloak client
	err := keycloak.InitializeOauthServer()
	if err != nil {
		log.Fatal("Keycloak: ", err)
	}

	// Initialize database
	go func() {
		err = database.Db.InitDb()
		if err != nil {
			log.Fatal("Db init:", err)
		}
	}()
	if conf.SentryDSN != "" {
		err = sentry.Init(sentry.ClientOptions{
			Dsn: conf.SentryDSN,
			// Enable printing of SDK debug messages.
			// Useful when getting started or trying to figure something out.
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		sentry.CaptureMessage("Server started")
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)

	mailer.Init()
	// Initialize server
	log.Info("Listening on port ", conf.Port)
	err = http.ListenAndServe(":"+conf.Port, handlers.GetRouter())
	if err != nil {
		log.Fatal("Http-server: ", err)
	}
}

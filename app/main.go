package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	if err := config.InitConfig(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	conf := config.Config

	// Validate critical config and fail fast if missing
	if err := conf.Validate(); err != nil {
		log.Fatalf("configuration validation failed: %v", err)
	}

	sentryEnabled := conf.SentryDSN != ""
	notifications.InitNotifications(sentryEnabled)

	log.Info("Starting Augustin Server v", conf.Version)

	// Initialize Keycloak client
	if err := keycloak.InitializeOauthServer(); err != nil {
		log.Fatal("Keycloak: ", err)
	}

	// Initialize database synchronously so server starts only when DB is ready
	if err := database.Db.InitDb(); err != nil {
		log.Fatal("Db init:", err)
	}

	if conf.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{Dsn: conf.SentryDSN}); err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		sentry.CaptureMessage("Server started")
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)

	mailer.Init()

	// Initialize server with graceful shutdown
	srv := &http.Server{
		Addr:              ":" + conf.Port,
		Handler:           handlers.GetRouter(),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       25 * time.Second,
		WriteTimeout:      25 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start server in background
	go func() {
		log.Info("Listening on port ", conf.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Http-server: ", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Info("Server exiting")
}

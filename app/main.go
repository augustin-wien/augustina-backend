package main

import (
	"augustin/config"
	"augustin/database"
	"augustin/handlers"
	"augustin/keycloak"
	"augustin/mailer"
	"augustin/notifications"
	"augustin/utils"
	"net/http"
)

var log = utils.GetLogger()

func main() {
	// Initialize config
	config.InitConfig()
	conf := config.Config
	notifications.InitNotifications()

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

	mailer.Init()
	// Initialize server
	log.Info("Listening on port ", conf.Port)
	err = http.ListenAndServe(":"+conf.Port, handlers.GetRouter())
	if err != nil {
		log.Fatal("Http-server: ", err)
	}
}

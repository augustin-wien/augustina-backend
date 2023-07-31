package main

import (
	"augustin/config"
	"augustin/database"
	"augustin/handlers"
	"augustin/keycloak"
	"augustin/utils"
	"net/http"
)

var log = utils.GetLogger()
var conf = config.Config

func main() {
	log.Info("Starting Augustin Server v", conf.Version)

	// Initialize Keycloak client
	keycloak.InitializeOauthServer()

	// Initialize database
	go database.Db.InitDb()

	// Initialize server
	log.Info("Listening on port ", conf.Port)
	err := http.ListenAndServe(":"+conf.Port, handlers.GetRouter())
	if err != nil {
		log.Fatal(err)
	}
}

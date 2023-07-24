package main

import (
	"net/http"

	"augustin/database"
	"augustin/handlers"
	"augustin/keycloak"
	"augustin/utils"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var log = utils.InitLog()

func main() {
	log.Info("Starting Augustin Server v0.0.1")

	// load .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Debug(".env file not found")
	}

	// Initialize Keycloak client
	keycloak.InitializeOauthServer()

	// Initialize database
	go database.InitDb()

	// Initialize server
	s := handlers.CreateNewServer()
	s.MountHandlers()
	log.Info("Server started on port 3000")
	err = http.ListenAndServe(":3000", s.Router)
	if err != nil {
		log.Fatal("Server stopped", zap.Error(err))
	}
}

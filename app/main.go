package main

import (
	"net/http"

	"augustin/database"
	"augustin/handlers"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"augustin/keycloak"
)

func main() {
	initLog()
	log.Info("Starting Augustin Server v0.0.1")
	// load .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Debug(".env file not found")
	}
	// Initialize Keycloak client
	keycloak.InitializeOauthServer()
	go database.InitDb()
	s := handlers.CreateNewServer()
	s.MountHandlers()
	log.Info("Server started on port 3000")
	err = http.ListenAndServe(":3000", s.Router)
	log.Error("Server stopped ", err)
}

func initLog() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
}

package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"augustin/database"
	"augustin/keycloack"
	"augustin/middlewares"
)

func initLog() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	// customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
}
func main() {
	// load .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Error("Error loading .env file")
	}
	initLog()
	log.Info("Starting Augustin Server v0.0.1")
	// Initialize Keycloak client
	keycloack.InitializeOauthServer()
	go database.InitDb()
	s := CreateNewServer()
	s.MountHandlers()
	log.Info("Server started on port 3000")
	err = http.ListenAndServe(":3000", s.Router)
	log.Error("Server stopped ", err)
}

// HelloWorld api Handler
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		log.Error("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(greeting))
}

type Server struct {
	Router *chi.Mux
	// Db, config can be added here
}

func CreateNewServer() *Server {
	s := &Server{}
	s.Router = chi.NewRouter()
	return s
}

func (s *Server) MountHandlers() {
	// Mount all Middleware here
	s.Router.Use(middleware.Logger)
	s.Router.Use(middlewares.AuthMiddleware)
	s.Router.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://localhost*", "http://localhost*", os.Getenv("FRONTEND_URL")},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Mount all handlers here
	s.Router.Get("/hello", HelloWorld)

}

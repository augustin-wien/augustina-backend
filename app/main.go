package main

import (
	"net/http"
	"os"

	"augustin/database"
	"augustin/handlers"
	"augustin/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"augustin/keycloack"

	httpSwagger "github.com/swaggo/http-swagger"
)

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
		log.Fatal(err)
	}
	log.Info("HelloWorld ", greeting)
}

type Server struct {
	Router *chi.Mux
	// Db, config can be added here
}

func initLog() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
}

func CreateNewServer() *Server {
	s := &Server{}
	s.Router = chi.NewRouter()
	return s
}

func (s *Server) MountHandlers() {
	// Mount all Middleware here
	s.Router.Use(middleware.Logger)
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
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middlewares.AuthMiddleware)

	// Mount all handlers here
	s.Router.Get("/api/hello/", handlers.HelloWorld)

	s.Router.Get("/api/settings/", handlers.Settings)

	s.Router.Get("/api/vendor/", handlers.Vendors)

	s.Router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:3000/docs/swagger.json"),
	))

	// Mount static file server in img folder
	fs := http.FileServer(http.Dir("img"))
	s.Router.Handle("/img/*", http.StripPrefix("/img/", fs))

	fs = http.FileServer(http.Dir("docs"))
	s.Router.Handle("/docs/*", http.StripPrefix("/docs/", fs))

}

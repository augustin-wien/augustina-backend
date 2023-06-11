package main

import (
	"net/http"

	"augustin/database"
	"augustin/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"

	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	initLog()
	log.Info("Starting Augustin Backend Server v0.0.1")
	go database.InitDb()
	s := CreateNewServer()
	s.MountHandlers()
	log.Info("Server started on port 3000")
	err := http.ListenAndServe(":3000", s.Router)
	if err != nil {
		log.Fatal(err)
	}
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
	s.Router.Use(middleware.Recoverer)

	// Mount all handlers here
	s.Router.Get("/api/hello/", handlers.HelloWorld)

	s.Router.Get("/payments", handlers.GetPayments)
	s.Router.Post("/payments", handlers.CreatePayments)

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

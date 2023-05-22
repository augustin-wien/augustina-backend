package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"

	"augustin/database"
)

func initLog() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	// customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
}
func main() {
	initLog()
	log.Info("Starting Augustin Server v0.0.1")
	go database.InitDb()
	s := CreateNewServer()
	s.MountHandlers()
	log.Info("Server started on port 3000")
	http.ListenAndServe(":3000", s.Router)
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

	// Mount all handlers here
	s.Router.Get("/hello", HelloWorld)

}

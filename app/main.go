package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// var (
// 	port = flag.String("port", ":3000", "Port to listen on")
// )

// func helloHandler(w http.ResponseWriter, r *http.Request) {

//     if r.URL.Path != "/hello" {
//         http.Error(w, "404 not found.", http.StatusNotFound)
//         return
//     }

//     if r.Method != "GET" {
//         http.Error(w, "Method is not supported.", http.StatusNotFound)
//         return
//     }

//     fmt.Fprintf(w, "Hello World!")
// }

func main() {
	s := CreateNewServer()
	s.MountHandlers()
	http.ListenAndServe(":3000", s.Router)
}

// HelloWorld api Handler
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
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

package main

import (
	"encoding/json"
	"net/http"

	"augustin/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"
)

type setting struct {
	Color string  `json:"color"`
	Logo  string  `json:"logo"`
	Price float64 `json:"price"`
}

type vendor struct {
	Credit   float64 `json:"credit"`
	QRcode   string  `json:"qrcode"`
	IDnumber string  `json:"id-number"`
}

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

func Setting(w http.ResponseWriter, r *http.Request) {
	marshal_struct, err := json.Marshal(setting{Color: "red", Logo: "/img/Augustin-Logo-Rechteck.jpg", Price: 3.14})
	if err != nil {
		log.Info(err)
		return
	}
	w.Write([]byte(marshal_struct))
}

func Vendor(w http.ResponseWriter, r *http.Request) {
	marshal_struct, err := json.Marshal(vendor{Credit: 1.61, QRcode: "/img/Augustin-QR-Code.png", IDnumber: "123456789"})
	if err != nil {
		log.Info(err)
		return
	}
	w.Write([]byte(marshal_struct))
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

	s.Router.Get("/settings", Setting)

	s.Router.Get("/vendor", Vendor)

	fs := http.FileServer(http.Dir("img"))
	s.Router.Handle("/img/*", http.StripPrefix("/img/", fs))

}

package handlers

import (
	"augustin/database"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// HelloWorld api Handler
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
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

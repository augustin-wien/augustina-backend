package handlers

import (
	"augustin/database"
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// HelloWorld API Handler fetching data from database
func HelloWorld(w http.ResponseWriter, r *http.Request) {
	greeting, err := database.Db.GetHelloWorld()
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(greeting))
}

// Setting API Handler fetching data without database
func Settings(w http.ResponseWriter, r *http.Request) {
	marshal_struct, err := json.Marshal(Setting{Color: "red", Logo: "/img/Augustin-Logo-Rechteck.jpg", Price: 3.14})
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(marshal_struct))
}

// Vendor API Handler fetching data without database
func Vendors(w http.ResponseWriter, r *http.Request) {
	marshal_struct, err := json.Marshal(Vendor{Credit: 1.61, QRcode: "/img/Augustin-QR-Code.png", IDnumber: "123456789"})
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(marshal_struct))
}

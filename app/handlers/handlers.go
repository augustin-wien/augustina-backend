package handlers

import (
	"augustin/database"
	"augustin/structs"
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

// GetPayments API Handler fetching data from database
func GetPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := database.Db.GetPayments()
	if err != nil {
		log.Errorf("GetPayments DB Error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	marshal_struct, err := json.Marshal(payments)
	if err != nil {
		log.Errorf("JSON conversion failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(marshal_struct))
}

// CreatePayments API Handler creating multiple payments
func CreatePayments(w http.ResponseWriter, r *http.Request) {
	var paymentBatch structs.PaymentBatch
	err := json.NewDecoder(r.Body).Decode(&paymentBatch)
	if err != nil {
		log.Errorf("JSON conversion failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = database.Db.CreatePayments(paymentBatch.Payments)
	if err != nil {
		log.Errorf("CreatePayments query failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Setting API Handler fetching data without database
func Settings(w http.ResponseWriter, r *http.Request) {
	marshal_struct, err := json.Marshal(structs.Setting{Color: "red", Logo: "/img/Augustin-Logo-Rechteck.jpg", Price: 3.14})
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(marshal_struct))
}

// Vendor API Handler fetching data without database
func Vendors(w http.ResponseWriter, r *http.Request) {
	marshal_struct, err := json.Marshal(structs.Vendor{Credit: 1.61, QRcode: "/img/Augustin-QR-Code.png", IDnumber: "123456789"})
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(marshal_struct))
}

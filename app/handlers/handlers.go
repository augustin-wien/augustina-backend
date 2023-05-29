//	@title			Swagger Example API
//	@version		0.0.1
//	@description	This is a sample server celler server.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:3000
//	@BasePath	/swagger/

//	@securityDefinitions.basic	BasicAuth

//	@externalDocs.description	OpenAPI
//	@externalDocs.url			https://swagger.io/resources/open-api/

package handlers

import (
	"augustin/database"
	"encoding/json"
	"net/http"

	_ "github.com/swaggo/files"        // swagger embed files
	_ "github.com/swaggo/http-swagger" // http-swagger middleware

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

// ListAccounts godoc
//
//	@Summary		List settings
//	@Description	get settings
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			q	query	string	false	"name search by q"	Format(email)
//	@Success		200	{array}	handlers.Setting
//	@Router			/settings [get]
//
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

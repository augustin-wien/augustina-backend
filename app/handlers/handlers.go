//	@title			Augustin Swagger
//	@version		0.0.1
//	@description	This swagger describes every endpoint of this project.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	GNU Affero General Public License
//	@license.url	https://www.gnu.org/licenses/agpl-3.0.txt

//	@host		localhost:3000
//	@BasePath	/api/

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

// ReturnHelloWorld godoc
//
//	 	@Summary 		Return HelloWorld
//		@Description	Return HelloWorld as sample API call
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Router			/hello/ [get]
//
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

// ReturnSettings godoc
//
//	 	@Summary 		Return settings
//		@Description	Return settings about the web-shop
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	handlers.Setting
//		@Router			/settings/ [get]
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

// ReturnVendorInformation godoc
//
//	 	@Summary 		Return vendor information
//		@Description	Return information for the vendor
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	handlers.Vendor
//		@Router			/vendor/ [get]
//
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

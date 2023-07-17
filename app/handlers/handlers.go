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
	"augustin/structs"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	_ "github.com/swaggo/files"        // swagger embed files
	_ "github.com/swaggo/http-swagger" // http-swagger middleware

	_ "github.com/swaggo/files" // swagger embed files

	"augustin/database"
	"augustin/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	httpSwagger "github.com/swaggo/http-swagger"
)

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
	s.Router.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://localhost*", "http://localhost*", os.Getenv("FRONTEND_URL")},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	s.Router.Use(middleware.Recoverer)

	// Mount all handlers here
	s.Router.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * 1000000000)) // 60 seconds
		r.Use(middlewares.AuthMiddleware)
		r.Get("/api/auth/hello/", HelloWorld)
	})
	s.Router.Get("/api/hello/", HelloWorld)

	s.Router.Get("/api/payments/", GetPayments)
	s.Router.Post("/api/payments/", CreatePayments)

	s.Router.Get("/api/settings/", GetSettings)

	s.Router.Get("/api/vendor/", Vendors)

	s.Router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:3000/docs/swagger.json"),
	))

	// Mount static file server in img folder
	fs := http.FileServer(http.Dir("img"))
	s.Router.Handle("/img/*", http.StripPrefix("/img/", fs))

	fs = http.FileServer(http.Dir("docs"))
	s.Router.Handle("/docs/*", http.StripPrefix("/docs/", fs))

}

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
	// greeting, err := database.Db.GetHelloWorld()
	// if err != nil {
	// 	log.Errorf("QueryRow failed: %v\n", err)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	w.Write([]byte("Hello, world!"))
}


// UpdateItem requires a multipart form
// https://www.sobyte.net/post/2022-03/go-multipart-form-data/
func UpdateItem(w http.ResponseWriter, r *http.Request) (err error) {

	// Read multipart form
	r.ParseMultipartForm(32 << 20)
	mForm := r.MultipartForm

	// Handle normal fields
	var item structs.Item
	fields := mForm.Value
	err = mapstructure.Decode(fields, &item)
    if err != nil {
        panic(err)
    }

    // Get file from image field
    file, header, err := r.FormFile("Image")
    if err != nil {
        panic(err)
    }
    defer file.Close()

	// Debugging
    name := strings.Split(header.Filename, ".")
	log.Info("Uploading %s\n", name[0])

	// Save file
	path := "/img/"+header.Filename
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	io.Copy(f, file)
	item.Image = path

	// Save item to database
	err = database.Db.UpdateItem(item)
	if err != nil {
		panic(err)
	}

    return err
}


// CreatePayments godoc
//
//	 	@Summary 		Get all payments
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	structs.Payment
//		@Router			/payments [get]
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

// CreatePayments godoc
//
//	 	@Summary 		Create a set of payments
//		@Description    {"Payments":[{"Sender": 1, "Receiver":1, "Type":1,"Amount":1.00]}
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	structs.PaymentType
//		@Router			/payments [post]
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

// ReturnSettings godoc
//
//	 	@Summary 		Return settings
//		@Description	Return settings about the web-shop
//		@Tags			core
//		@Accept			json
//		@Produce		json
//		@Success		200	{array}	structs.Settings
//		@Router			/settings/ [get]
//
// Get Settings API Handler fetching data without database
func GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := database.Db.GetSettings()
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	marshal_struct, err := json.Marshal(settings)
	if err != nil {
		log.Errorf("JSON conversion failed: %v\n", err)
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
//		@Success		200	{array}	structs.Vendor
//		@Router			/vendor/ [get]
//
// Vendor API Handler fetching data without database
func Vendors(w http.ResponseWriter, r *http.Request) {
	vendors, err := database.Db.GetVendorSettings()
	if err != nil {
		log.Errorf("QueryRow failed: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(vendors))
}

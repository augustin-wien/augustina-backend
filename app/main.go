package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initialize Keycloak client
		keycloakClient := gocloak.NewClient("http://keycloak:8080/")
		ctx := context.Background()

		// Authenticate the user
		token, err := keycloakClient.LoginClient(ctx, "go-client", "9OGqiDdguQHhPQ90MgPV7hEKFEE5A5jB", "augustin")
		if err != nil {
			http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
			log.Errorf("Failed to authenticate: %v", err.Error())
			return
		}
		user_token := strings.Split(r.Header.Get("Authorization"), " ")[1]
		log.Info("get user token ", user_token)

		// Validate the token
		// Todo

		// Get the user by username
		user, err := keycloakClient.GetUserByID(ctx, token.AccessToken, "augustin", "user001")
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// Check if the user has the required role
		hasRole, err := keycloakClient.GetRoleMappingByUserID(ctx, token.AccessToken, "augustin", *user.ID)
		if err != nil {
			http.Error(w, "Failed to check user roles", http.StatusInternalServerError)
			return
		}
		log.Info("Check MappingsRepresentation type", hasRole)

		// TODO
		// If the user has the required role, proceed to the next handler
		// if hasRole == "go-admin" {
		// 	next.ServeHTTP(w, r)
		// } else {
		// 	http.Error(w, "Unauthorized", http.StatusUnauthorized)
		// }
	})
}

func (s *Server) MountHandlers() {
	// Mount all Middleware here
	s.Router.Use(middleware.Logger)
	s.Router.Use(AuthMiddleware)

	// Mount all handlers here
	s.Router.Get("/hello", HelloWorld)

}

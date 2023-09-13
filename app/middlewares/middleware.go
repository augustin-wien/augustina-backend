package middlewares

import (
	"augustin/keycloak"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// AuthMiddleware is a middleware to check if the request is authorized
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// ignore for options request
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return // skip
		}

		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userToken := strings.Split(r.Header.Get("Authorization"), " ")[1]
		userinfo, err := keycloak.KeycloakClient.GetUserInfo(userToken)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Info("userinfo ", userinfo, err)
		next.ServeHTTP(w, r)
	})
}

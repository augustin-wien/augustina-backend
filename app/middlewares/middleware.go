package middlewares

import (
	"augustin/keycloack"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

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
		user_token := strings.Split(r.Header.Get("Authorization"), " ")[1]
		userinfo, err := keycloack.KeycloakClient.GetUserInfo(user_token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Info("userinfo ", userinfo, err)
		next.ServeHTTP(w, r)
	})
}

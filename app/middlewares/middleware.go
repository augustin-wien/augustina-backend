package middlewares

import (
	"augustin/keycloak"
	"augustin/utils"
	"net/http"
	"strings"
)

var log = utils.GetLogger()

// AuthMiddleware is a middleware to check if the request is authorized
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// ignore for options request
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return // skip
		}

		if r.Header.Get("Authorization") == "" {
			log.Info("No Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		splitToken := strings.Split(r.Header.Get("Authorization"), " ")
		userToken := strings.Split(r.Header.Get("Authorization"), " ")[0]
		if len(splitToken) == 2 {
			userToken = splitToken[1]
		}
		// unset all possible user headers for security reasons
		r.Header.Set("X-Auth-User-Validated", "false")
		r.Header.Del("X-Auth-User")
		r.Header.Del("X-Auth-Roles-vendor")
		r.Header.Del("X-Auth-Roles-admin")

		userinfo, err := keycloak.KeycloakClient.GetUserInfo(userToken)
		if err != nil {
			log.Info("Error getting userinfo ", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// set user headers
		r.Header.Set("X-Auth-User", *userinfo.Sub)
		r.Header.Set("X-Auth-User-Validated", "true")

		// set user roles headers
		userRoles, err := keycloak.KeycloakClient.GetUserRoles(*userinfo.Sub)
		if err != nil {
			log.Info("Error getting userRoles ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		for _, role := range userRoles {
			r.Header.Add("X-Auth-Roles-"+*role.Name, *role.Name)
		}
		next.ServeHTTP(w, r)
	})
}

// VendorAuthMiddleware is a middleware to check if the request is authorized as vendor
func VendorAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// ignore for options request
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return // skip
		}

		if r.Header.Get("X-Auth-User-Validated") == "false" {
			log.Info("No Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Auth-Roles-vendor") == "" || r.Header.Get("X-Auth-Roles-admin") == "" {
			log.Info("No Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AdminAuthMiddleware is a middleware to check if the request is authorized as admin
func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// ignore for options request
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return // skip
		}

		if r.Header.Get("X-Auth-User-Validated") == "false" {
			log.Info("No Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Auth-Roles-admin") == "" {
			log.Info("No Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

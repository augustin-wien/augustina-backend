package middlewares

import (
	"augustin/keycloak"
	"augustin/utils"
	"errors"
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
			log.Info("Unauthorization: No Authorization header on auth from incoming request ", utils.ReadUserIP(r))
			utils.ErrorJSON(w, errors.New("Unauthorized"), http.StatusUnauthorized)
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
		r.Header.Del("X-Auth-User-Email")
		r.Header.Del("X-Auth-Roles-vendor")
		r.Header.Del("X-Auth-Roles-admin")
		r.Header.Del("X-Auth-Groups-Vendors")
		r.Header.Del("X-Auth-Groups-Admins")

		userinfo, err := keycloak.KeycloakClient.GetUserInfo(userToken)
		if err != nil {
			log.Info("Error getting userinfo ", err)
			utils.ErrorJSON(w, errors.New("Unauthorized"), http.StatusUnauthorized)
			return
		}

		// set user headers
		r.Header.Set("X-Auth-User", *userinfo.Sub)
		r.Header.Set("X-Auth-User-Name", *userinfo.PreferredUsername)
		r.Header.Set("X-Auth-User-Email", *userinfo.Email)
		r.Header.Set("X-Auth-User-Validated", "true")

		// set user roles headers
		userRoles, err := keycloak.KeycloakClient.GetUserRoles(*userinfo.Sub)
		if err != nil {
			log.Info("Error getting userRoles ", err)
			utils.ErrorJSON(w, errors.New("internal Server Error"), http.StatusInternalServerError)
			return
		}

		for _, role := range userRoles {
			r.Header.Add("X-Auth-Roles-"+*role.Name, *role.Name)
		}
		userGroups, err := keycloak.KeycloakClient.GetUserGroups(*userinfo.Sub)
		if err != nil {
			log.Info("Error getting userGroups ", err)
			utils.ErrorJSON(w, errors.New("internal Server Error"), http.StatusInternalServerError)
			return
		}
		for _, group := range userGroups {
			r.Header.Add("X-Auth-Groups-"+*group.Name, *group.Name)
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
			log.Info("VendorAuthMiddleware: No validated user", r.Header.Get("X-Auth-User"))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Header.Get("X-Auth-Groups-Vendors") != "" || r.Header.Get("X-Auth-Roles-admin") != "" {
			next.ServeHTTP(w, r)
		} else {
			log.Info("VendorAuthMiddleware: user is missing vendor role with user id ", r.Header.Get("X-Auth-User"))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
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
			log.Info("AdminAuthMiddleware: No validated user", r.Header.Get("X-Auth-User"))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("X-Auth-Roles-admin") == "" {
			log.Infof("AdminAuthMiddleware: User %v has no admin role", r.Header.Get("X-Auth-User"))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

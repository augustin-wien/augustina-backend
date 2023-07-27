package handlers

import (
	"net/http"
	"os"

	_ "github.com/swaggo/files" // swagger embed files

	"augustin/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	httpSwagger "github.com/swaggo/http-swagger"
)

// GetRouter creates a new chi Router and mounts all handlers
func GetRouter() (r *chi.Mux) {
	r = chi.NewRouter()

	// Mount all Middleware here
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://localhost*", "http://localhost*", os.Getenv("FRONTEND_URL")},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.Recoverer)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * 1000000000)) // 60 seconds
		r.Use(middlewares.AuthMiddleware)
		r.Get("/api/auth/hello/", HelloWorld)
	})

	// Public routes
	r.Get("/api/hello/", HelloWorld)
	r.Get("/api/payments/", GetPayments)
	r.Post("/api/payments/", CreatePayments)
	r.Post("/api/transaction/", CreateTransaction)
	r.Get("/api/settings/", getSettings)
	r.Get("/api/vendor/", Vendors)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:3000/docs/swagger.json"),
	))

	// Mount static file server in img folder
	fs := http.FileServer(http.Dir("img"))
	r.Handle("/img/*", http.StripPrefix("/img/", fs))

	fs = http.FileServer(http.Dir("docs"))
	r.Handle("/docs/*", http.StripPrefix("/docs/", fs))

	return r
}

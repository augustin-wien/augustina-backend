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

	// Check that FRONTEND_URL environment variable is set
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		log.Fatal("FRONTEND_URL environment variable is not set")
	}

	// Define allowed origins
	allowedOrigins := []string{
		"http://localhost:*",  // Any open port on localhost without SSL
		"https://localhost:*", // Any open port on localhost with SSL
		frontendURL,           // Frontend URL from environment variable
	}

	// CORS handler configuration
	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	// Use CORS handler with Chi router
	r.Use(corsHandler)

	r.Use(middleware.Recoverer)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * 1000000000)) // 60 seconds
		r.Use(middlewares.AuthMiddleware)
		r.Get("/api/auth/hello/", HelloWorldAuth)
	})

	// Public routes
	r.Get("/api/hello/", HelloWorld)
	r.Route("/api/settings", func(r chi.Router) {
		r.Get("/", getSettings)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AdminAuthMiddleware)
			r.Put("/", updateSettings)
			r.Put("/css/", updateCSS)
		})
	})

	// Vendors
	r.Route("/api/vendors", func(r chi.Router) {
		r.Get("/check/{licenseID}/", CheckVendorsLicenseID)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AdminAuthMiddleware)
			r.Get("/", ListVendors)
			r.Post("/", CreateVendor)

			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", UpdateVendor)
				r.Delete("/", DeleteVendor)
				r.Get("/", GetVendor)
			})
		})
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.VendorAuthMiddleware)
			r.Get("/me/", GetVendorOverview)
		})
	})

	// Items
	r.Route("/api/items", func(r chi.Router) {
		r.Get("/", ListItems)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AdminAuthMiddleware)
			r.Get("/backoffice/", ListItemsBackoffice)
			r.Post("/", CreateItem)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", UpdateItem)
				r.Delete("/", DeleteItem)
			})
		})
	})

	// Payment orders
	r.Route("/api/orders", func(r chi.Router) {
		r.Post("/", CreatePaymentOrder)
		r.Get("/verify/", VerifyPaymentOrder)
	})

	// Payments
	r.Route("/api/payments", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AdminAuthMiddleware)
			r.Post("/", CreatePayment)
			r.Post("/batch/", CreatePayments)
			r.Get("/", ListPayments)
			r.Get("/forpayout/", ListPaymentsForPayout)
			r.Get("/statistics/", ListPaymentsStatistics)
			r.Post("/payout/", CreatePaymentPayout)
		})
	})

	// Payment service providers
	r.Route("/api/webhooks/vivawallet", func(r chi.Router) {
		r.Post("/success/", VivaWalletWebhookSuccess)
		r.Get("/success/", VivaWalletVerificationKey)
		r.Post("/failure/", VivaWalletWebhookFailure)
		r.Get("/failure/", VivaWalletVerificationKey)
		r.Post("/price/", VivaWalletWebhookPrice)
		r.Get("/price/", VivaWalletVerificationKey)
	})

	// Online Map
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AdminAuthMiddleware)
		r.Get("/api/map/", GetVendorLocations)
	})

	// PDF Upload
	r.Route("/api/pdf", func(r chi.Router) {
		r.Get("/{id}/validate/", validatePDFLink)
		r.Get("/{id}/", downloadPDF)
	})

	// Flour integration
	r.Route("/api/flour", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.FlourAuthMiddleware)
		r.Route("/vendors", func(r chi.Router) {

			r.Put("/license/{licenseID}/", UpdateVendorByLicenseID)
			r.Get("/license/{licenseID}/", GetVendorByLicenseID)

			r.Put("/{id}/", UpdateVendor)
			r.Delete("/{id}/", DeleteVendor)
			r.Get("/{id}/", GetVendor)
			r.Post("/", CreateVendor)

		})
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:3000/docs/swagger.json"),
	))

	// Mount static file servers in img folder
	fsImg := http.FileServer(http.Dir("img"))
	r.Handle("/img/*", http.StripPrefix("/img/", fsImg))

	// Mount style.css file server and always serve the same file
	fsCSS := http.FileServer(http.Dir("public"))
	r.Handle("/public/*", http.StripPrefix("/public/", fsCSS))

	// Docs file server is used for swagger documentation
	fsDocs := http.FileServer(http.Dir("docs"))
	r.Handle("/docs/*", http.StripPrefix("/docs/", fsDocs))

	return r
}

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
		r.Post("/", CreatePayment)
		r.Post("/batch/", CreatePayments)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AdminAuthMiddleware)
			r.Get("/", ListPayments)
			r.Get("/forpayout/", ListPaymentsForPayout)
			r.Post("/payout/", CreatePaymentPayout)
		})
	})

	// Payment service providers
	r.Route("/api/webhooks/vivawallet", func(r chi.Router) {
		// todo: add auth middleware for viva wallet ip addresses
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
		r.Get("/api/map/", GetLocationData)
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:3000/docs/swagger.json"),
	))

	// Mount static file servers in img folder
	fsImg := http.FileServer(http.Dir("img"))
	r.Handle("/img/*", http.StripPrefix("/img/", fsImg))
	// Docs file server is used for swagger documentation
	fsDocs := http.FileServer(http.Dir("docs"))
	r.Handle("/docs/*", http.StripPrefix("/docs/", fsDocs))

	return r
}

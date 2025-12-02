package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	_ "github.com/swaggo/files" // swagger embed files

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/go-chi/httprate"
	httpSwagger "github.com/swaggo/http-swagger"
)

// GetRouter creates a new chi Router and mounts all handlers
func GetRouter() (r *chi.Mux) {
	r = chi.NewRouter()
	// Mount all Middleware here
	// Add request IDs and structured request logging middleware
	r.Use(middleware.RequestID)
	r.Use(middlewares.RequestLogger)

	// Security middlewares: Block suspicious IPs and requests
	r.Use(middlewares.FilterBlockedIPs)
	r.Use(middlewares.BlockSuspiciousRequests)
	r.Use(middlewares.BlockBadUserAgents)
	r.Use(middlewares.BlockFakeBrowsers)
	r.Use(middlewares.BlockMaliciousPatterns)

	// Basic rate limiting: limit by IP to 100 requests per minute (tunable)
	r.Use(httprate.LimitByIP(500, 1*time.Minute))

	// Check that FRONTEND_URL is configured
	frontendURL := config.Config.FrontendURL
	if frontendURL == "" {
		log.Fatal("FRONTEND_URL is not set in config")
	}

	// Sanitize and validate frontend URL from config to avoid accidentally allowing wildcards
	// Use a conservative AllowOriginFunc instead of permissive origin globs. This prevents
	// unintended wildcard matching like "http://localhost:*" which may be treated inconsistently
	// by different CORS implementations.
	allowOriginFunc := func(r *http.Request, origin string) bool {
		if origin == frontendURL {
			return true
		}
		// Allow localhost origins (both http and https) but require the origin to start with
		// the scheme and host to prevent wildcards in the config value itself.
		if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost") {
			return true
		}
		return false
	}

	corsHandler := cors.Handler(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
		AllowOriginFunc:  allowOriginFunc,
	})

	// Use CORS handler with Chi router
	r.Use(corsHandler)

	r.Use(middleware.Recoverer)

	// Basic security headers to harden the application surface
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")
			// Reduce MIME based attacks
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Content Security Policy - restrict what resources can be loaded by the browser
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			// Referrer policy
			w.Header().Set("Referrer-Policy", "no-referrer")
			// XSS protection (legacy, harmless to enable)
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			// HSTS only when TLS is in use
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}
			next.ServeHTTP(w, r)
		})
	})

	// Health endpoints for load balancers
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check DB readiness
		if database.Db.Dbpool == nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := database.Db.Dbpool.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second)) // 60 seconds
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
			r.Get("/{vendorid}/locations/", ListVendorLocations)
			r.Post("/{vendorid}/locations/", CreateVendorLocation)
			r.Patch("/{vendorid}/locations/{id}/", UpdateVendorLocation)
			r.Delete("/{vendorid}/locations/{id}/", DeleteVendorLocation)
			r.Get("/{vendorid}/comments/", ListVendorComments)
			r.Post("/{vendorid}/comments/", CreateVendorComment)
			r.Delete("/{vendorid}/comments/{id}/", DeleteVendorComment)
			r.Patch("/{vendorid}/comments/{id}/", UpdateVendorComment)

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
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AdminAuthMiddleware)
			r.Get("/unverified/", ListUnverifiedOrders)
		})
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

	// Mail templates management
	r.Route("/api/mail-templates", func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AdminAuthMiddleware)
		r.Get("/", ListMailTemplates)
		r.Get("/{name}/", GetMailTemplate)
		r.Group(func(r chi.Router) {
			r.Post("/", CreateOrUpdateMailTemplate)
			r.Post("/{name}/send/", SendMailTemplateTest)
			r.Delete("/{name}/", DeleteMailTemplate)
		})
	})

	// Flour integration
	if config.Config.FlourWebhookURL != "" {
		log.Info("Flour integration enabled")

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
			r.Route("/payments", func(r chi.Router) {
				r.Post("/payout/", CreatePaymentPayout)
			})
		})

	}
	// Swagger documentation - only expose in development mode to avoid leaking API docs or
	// generated files that may contain example credentials. The `Development` flag is set
	// from the environment in `config.Config`.
	if config.Config.Development {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("http://localhost:3000/docs/swagger.json"),
		))
	}

	// Mount static file servers in img folder
	fsImg := http.FileServer(http.Dir("img"))
	r.Handle("/img/*", http.StripPrefix("/img/", fsImg))

	// Mount style.css file server and always serve the same file
	fsCSS := http.FileServer(http.Dir("public"))
	r.Handle("/public/*", http.StripPrefix("/public/", fsCSS))

	// Only serve repository docs in development. Serving repo documentation from the
	// application in production can leak sensitive example data (credentials, tokens,
	// verification keys). Keep docs available for local development but do not expose
	// them publicly when `Development` is false.
	if config.Config.Development {
		fsDocs := http.FileServer(http.Dir("docs"))
		r.Handle("/docs/*", http.StripPrefix("/docs/", fsDocs))
	}

	return r
}

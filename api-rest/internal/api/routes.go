package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"stealthcompany.com/api-rest/internal/metrics"
)

// SetupRoutes configures and returns the HTTP router
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Add middleware to all routes
	r.Use(metrics.MetricsMiddleware)
	r.Use(AuthMiddleware) // JWT authentication middleware
	r.Use(TenantChannelMiddleware)

	// Note: Couchbase connections are now created per-request to avoid globals

	// Public routes (no authentication required)
	r.HandleFunc("/", RootHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Authentication routes (no tenant required)
	keycloakConfig, err := NewKeycloakConfig()
	if err != nil {
		// Log error but continue - auth routes will use dummy config
		keycloakConfig = &KeycloakConfig{}
	}
	ConfigureAuthRoutes(r, keycloakConfig)

	// Tenant-based API routes
	apiRouter := r.PathPrefix("/api/{tenant}").Subrouter()

	// FHIR resource endpoints for specific tenant
	apiRouter.HandleFunc("/encounters", ListResourcesHandler("Encounter")).Methods("GET")
	apiRouter.HandleFunc("/encounters/{id}", GetResourceByIDHandler("Encounter")).Methods("GET")
	apiRouter.HandleFunc("/patients", ListResourcesHandler("Patient")).Methods("GET")
	apiRouter.HandleFunc("/patients/{id}", GetResourceByIDHandler("Patient")).Methods("GET")
	apiRouter.HandleFunc("/practitioners", ListResourcesHandler("Practitioner")).Methods("GET")
	apiRouter.HandleFunc("/practitioners/{id}", GetResourceByIDHandler("Practitioner")).Methods("GET")

	// Review request endpoint for specific tenant
	apiRouter.HandleFunc("/review-request", ReviewRequestHandler).Methods("POST")

	// Legacy routes for backward compatibility (will be deprecated)
	legacyRouter := r.PathPrefix("/legacy").Subrouter()
	legacyRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Legacy routes still use X-Tenant-ID header
			next.ServeHTTP(w, r)
		})
	})
	legacyRouter.HandleFunc("/encounters", ListResourcesHandler("Encounter")).Methods("GET")
	legacyRouter.HandleFunc("/encounters/{id}", GetResourceByIDHandler("Encounter")).Methods("GET")
	legacyRouter.HandleFunc("/patients", ListResourcesHandler("Patient")).Methods("GET")
	legacyRouter.HandleFunc("/patients/{id}", GetResourceByIDHandler("Patient")).Methods("GET")
	legacyRouter.HandleFunc("/practitioners", ListResourcesHandler("Practitioner")).Methods("GET")
	legacyRouter.HandleFunc("/practitioners/{id}", GetResourceByIDHandler("Practitioner")).Methods("GET")
	legacyRouter.HandleFunc("/review-request", ReviewRequestHandler).Methods("POST")

	return r
}

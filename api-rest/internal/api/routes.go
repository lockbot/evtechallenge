package api

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/metrics"
)

// SetupRoutes configures and returns the HTTP router
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Add metrics middleware to all routes
	r.Use(metrics.MetricsMiddleware)

	// Add authentication middleware to all routes
	r.Use(AuthMiddleware)

	// Initialize Couchbase (non-fatal if fails; endpoints will report unavailable)
	err := InitCouchbase()
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize Couchbase; endpoints will be unavailable")
	}

	// Routes
	r.HandleFunc("/", HelloHandler).Methods("GET")
	r.HandleFunc("/hello", HelloHandler).Methods("GET")
	r.HandleFunc("/all-good", AllGoodHandler).Methods("POST")

	// FHIR resource endpoints
	r.HandleFunc("/encounters", ListResourcesHandler("Encounter")).Methods("GET")
	r.HandleFunc("/encounters/{id}", GetResourceByIDHandler("Encounter")).Methods("GET")
	r.HandleFunc("/patients", ListResourcesHandler("Patient")).Methods("GET")
	r.HandleFunc("/patients/{id}", GetResourceByIDHandler("Patient")).Methods("GET")
	r.HandleFunc("/practitioners", ListResourcesHandler("Practitioner")).Methods("GET")
	r.HandleFunc("/practitioners/{id}", GetResourceByIDHandler("Practitioner")).Methods("GET")

	// Review request endpoint
	r.HandleFunc("/review-request", ReviewRequestHandler).Methods("POST")

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Health check endpoint (no auth required)
	r.HandleFunc("/health", HealthCheckHandler).Methods("GET")

	return r
}

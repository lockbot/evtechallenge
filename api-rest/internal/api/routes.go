package api

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"stealthcompany.com/api-rest/internal/metrics"
)

// SetupRoutes configures and returns the HTTP router
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Add middleware to all routes
	r.Use(metrics.MetricsMiddleware)
	r.Use(TenantChannelMiddleware)

	// Note: Couchbase connections are now created per-request to avoid globals

	// Routes
	r.HandleFunc("/", RootHandler).Methods("GET")

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

	return r
}

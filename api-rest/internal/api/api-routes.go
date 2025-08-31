package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/api/internal/metrics" // Update with your actual module name
)

// AllGoodRequest represents the expected JSON payload
type AllGoodRequest struct {
	Yes bool `json:"yes"`
}

// helloHandler returns a simple hello world message
func helloHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("Hello endpoint called")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{
		"message": "Hello, World!",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// allGoodHandler expects {"yes": true} or returns business error
func allGoodHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("All-good endpoint called")

	if r.Method != http.MethodPost {
		log.Warn().
			Str("method", r.Method).
			Msg("Method not allowed on all-good endpoint")

		metrics.RecordAllGoodRequest("method_not_allowed")

		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	var req AllGoodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to decode JSON request")

		metrics.RecordAllGoodRequest("invalid_json")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON format",
		})
		return
	}

	if !req.Yes {
		log.Warn().
			Bool("received_yes", req.Yes).
			Msg("Business validation failed - yes must be true")

		metrics.RecordAllGoodRequest("validation_failed")

		w.WriteHeader(http.StatusUnprocessableEntity) // 422 - Business logic error
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Business validation failed",
			"message": "Field 'yes' must be true",
		})
		return
	}

	log.Info().
		Bool("yes", req.Yes).
		Msg("All good request processed successfully")

	metrics.RecordAllGoodRequest("success")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "All good!",
		"status":  "success",
	})
}

// SetupRoutes configures and returns the HTTP router
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Add metrics middleware to all routes
	r.Use(metrics.MetricsMiddleware)

	// Routes
	r.HandleFunc("/", helloHandler).Methods("GET")
	r.HandleFunc("/hello", helloHandler).Methods("GET")
	r.HandleFunc("/all-good", allGoodHandler).Methods("POST")

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	return r
}

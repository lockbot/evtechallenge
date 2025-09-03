package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// TenantWarmthMiddleware checks if a tenant is warm before allowing access to FHIR routes
func TenantWarmthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip middleware for non-FHIR routes
		if !isFHIRRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Get tenant ID from request
		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		// Check if tenant is warm
		tgm := GetTenantGoroutineManager()
		if tgm == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "tenant manager not initialized"})
			return
		}

		if !tgm.IsTenantWarm(tenantID) {
			log.Warn().
				Str("tenant", tenantID).
				Str("path", r.URL.Path).
				Msg("Tenant not warm, rejecting request")

			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "tenant not warm",
				"message": "Please call /warm-up-tenant first to initialize this tenant",
				"tenant":  tenantID,
				"status":  "cold",
			})
			return
		}

		// Record activity to keep tenant warm
		tgm.RecordActivity(tenantID)

		// Tenant is warm, proceed with request
		next.ServeHTTP(w, r)
	})
}

// isFHIRRoute checks if the given path is a FHIR route that requires tenant warmth
func isFHIRRoute(path string) bool {
	// FHIR routes that require tenant to be warm
	fhirRoutes := []string{
		"/encounters",
		"/encounter",
		"/patients",
		"/patient",
		"/practitioners",
		"/practitioner",
		"/review-request",
	}

	path = strings.ToLower(path)
	for _, route := range fhirRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}

	return false
}

// WarmUpRequiredResponse returns a standardized response when tenant warm-up is required
func WarmUpRequiredResponse(w http.ResponseWriter, tenantID string) {
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":           "tenant not warm",
		"message":         "Please call POST /warm-up-tenant to initialize this tenant",
		"tenant":          tenantID,
		"status":          "cold",
		"required_action": "warm_up",
	})
}

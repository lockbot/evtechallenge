package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// TenantChannelMiddleware routes requests through tenant channels if available
func TenantChannelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			// If no tenant ID, fallback to direct processing
			next.ServeHTTP(w, r)
			return
		}

		// Special handling for endpoints that don't require tenant warm-up
		if r.URL.Path == "/warm-up-tenant" || r.URL.Path == "/" || r.URL.Path == "/hello" || r.URL.Path == "/all-good" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		if channels, exists := GetTenantChannels(tenantID); exists {
			// Reset timer for this request
			channels.ResetTimer()

			// All requests go through handlers - they handle channel routing
			next.ServeHTTP(w, r)
		} else {
			// Tenant not warmed up - return error asking to warm up
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Tenant not warmed up",
				"message": "Please call /warm-up-tenant first",
			})
		}
	})
}

// extractID extracts the ID from a path like "/encounters/123"
func extractID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

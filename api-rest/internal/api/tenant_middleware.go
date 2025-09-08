package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// TenantChannelMiddleware routes requests through tenant channels if available
func TenantChannelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Special handling for endpoints that don't require tenant warm-up
		if r.URL.Path == "/" || r.URL.Path == "/metrics" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip tenant processing for auth routes
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			// If no tenant ID, fallback to direct processing
			next.ServeHTTP(w, r)
			return
		}

		if channels, exists := GetTenantChannels(tenantID); exists {
			// Check if channels are pseudo-closed
			if channels.pseudoClosed {
				// Channels are pseudo-closed, need to reactivate them
				channels = AutoWarmUpTenant(tenantID)
			}

			// Reset timer for this request
			channels.ResetTimer()

			// All requests go through handlers - they handle channel routing
			next.ServeHTTP(w, r)
		} else {
			// Auto-warm-up tenant on first request
			channels = AutoWarmUpTenant(tenantID)
			if channels == nil {
				// Auto-warm-up failed - return error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Failed to warm up tenant",
					"message": "Unable to initialize tenant channels",
				})
				return
			}

			// Reset timer for this request
			channels.ResetTimer()

			// All requests go through handlers - they handle channel routing
			next.ServeHTTP(w, r)
		}
	})
}

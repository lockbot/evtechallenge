package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/dal"
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

		// Ensure tenant scope exists and is ready
		if err := ensureTenantScope(r.Context(), tenantID); err != nil {
			log.Error().
				Err(err).
				Str("tenantID", tenantID).
				Msg("Failed to ensure tenant scope")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Failed to initialize tenant scope",
				"message": "Unable to access tenant data",
			})
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

// ensureTenantScope ensures that a tenant scope exists and is ready for use
func ensureTenantScope(ctx context.Context, tenantID string) error {
	// Get the database connection
	conn, err := dal.GetConnectionWithRetry()
	if err != nil {
		return err
	}
	defer dal.ReturnConnection(conn)

	// Create scope model and ensure tenant scope exists
	scopeModel := dal.NewScopeModel(conn)
	err = scopeModel.EnsureTenantScope(ctx, tenantID)
	if err != nil {
		return err
	}

	// Set the query context for this tenant
	bucketName := conn.GetBucketName()
	queryContext := fmt.Sprintf("default:%s.%s", bucketName, tenantID)
	SetTenantQueryContext(tenantID, queryContext)

	return nil
}

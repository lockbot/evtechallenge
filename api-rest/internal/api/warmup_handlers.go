package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// WarmUpTenantHandler handles POST /warm-up-tenant
func WarmUpTenantHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := GetTenantFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	log.Info().Str("tenant", tenantID).Msg("Warm-up request received")

	// Get the global tenant goroutine manager
	// TODO: This should be injected or accessed via a global instance
	tgm := GetTenantGoroutineManager()
	if tgm == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant manager not initialized"})
		return
	}

	// Warm up the tenant
	err = tgm.WarmUpTenant(tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant", tenantID).Msg("Failed to warm up tenant")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to warm up tenant"})
		return
	}

	// Get tenant status for response
	status := tgm.GetTenantStatus(tenantID)

	response := map[string]interface{}{
		"status":    "warm-up initiated",
		"tenant":    tenantID,
		"ready":     status.Ready,
		"warmedAt":  status.WarmedAt.Format(time.RFC3339),
		"message":   status.Message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info().Str("tenant", tenantID).Msg("Warm-up request completed successfully")
}

// GetTenantStatusHandler handles GET /tenant-status/{tenantID}
func GetTenantStatusHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := GetTenantFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Get the global tenant goroutine manager
	tgm := GetTenantGoroutineManager()
	if tgm == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant manager not initialized"})
		return
	}

	// Get tenant status
	status := tgm.GetTenantStatus(tenantID)
	if status == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "tenant not found"})
		return
	}

	response := map[string]interface{}{
		"tenant":      status.TenantID,
		"ready":       status.Ready,
		"warmedAt":    status.WarmedAt.Format(time.RFC3339),
		"lastRequest": status.LastRequest.Format(time.RFC3339),
		"message":     status.Message,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

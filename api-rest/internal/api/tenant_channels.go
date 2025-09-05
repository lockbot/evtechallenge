package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// TenantChannels represents the channel-based concurrency system for a tenant
type TenantChannels struct {
	getEncounterCh      chan RequestMessage
	listEncountersCh    chan RequestMessage
	getPatientCh        chan RequestMessage
	listPatientsCh      chan RequestMessage
	getPractitionerCh   chan RequestMessage
	listPractitionersCh chan RequestMessage
	reviewCh            chan RequestMessage
	cooldownCh          chan struct{}
	timerResetCh        chan struct{}
	responsePool        *ResponsePool
}

// RequestMessage contains the request data and response channel key
type RequestMessage struct {
	TenantID    string
	Entity      string
	ID          string
	ResponseKey string
	Page        int
	Count       int
}

// ResponseMessage contains the response data
type ResponseMessage struct {
	Data  interface{}
	Error error
}

var tenantChannels = make(map[string]*TenantChannels)
var channelsMutex sync.RWMutex

// WarmUpTenantHandler creates and starts channel-based processing for a tenant
func WarmUpTenantHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := GetTenantFromRequest(r)
	if err != nil {
		log.Warn().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("Invalid tenant ID in warm-up request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	channelsMutex.Lock()
	defer channelsMutex.Unlock()

	// Create channels for this tenant
	channels := &TenantChannels{
		getEncounterCh:      make(chan RequestMessage),
		listEncountersCh:    make(chan RequestMessage),
		getPatientCh:        make(chan RequestMessage),
		listPatientsCh:      make(chan RequestMessage),
		getPractitionerCh:   make(chan RequestMessage),
		listPractitionersCh: make(chan RequestMessage),
		reviewCh:            make(chan RequestMessage),
		cooldownCh:          make(chan struct{}),
		timerResetCh:        make(chan struct{}),
		responsePool:        NewResponsePool(5),
	}

	tenantChannels[tenantID] = channels

	// Start worker goroutine
	go channels.processMessages()

	// Start timer management goroutine
	go channels.manageTimer()

	log.Info().
		Str("tenant", tenantID).
		Msg("Tenant warmed up successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Tenant warmed up successfully",
		"status":  "success",
	})
}

// manageTimer handles the 10-minute timer with reset capability
func (tc *TenantChannels) manageTimer() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Timer expired - send cooldown signal
			tc.cooldownCh <- struct{}{}
			return
		case <-tc.timerResetCh:
			// Reset timer - create new ticker
			ticker.Stop()
			ticker = time.NewTicker(10 * time.Minute)
		}
	}
}

// GetTenantChannels returns the channels for a tenant if they exist
func GetTenantChannels(tenantID string) (*TenantChannels, bool) {
	channelsMutex.RLock()
	defer channelsMutex.RUnlock()
	channels, exists := tenantChannels[tenantID]
	return channels, exists
}

// ResetTimer resets the 10-minute timer for a tenant
func (tc *TenantChannels) ResetTimer() {
	select {
	case tc.timerResetCh <- struct{}{}:
		// Timer reset successfully
	default:
		// Channel might be closed, ignore
	}
}

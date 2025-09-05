package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/metrics"
)

// TenantChannels represents the channel-based concurrency system for a tenant
type TenantChannels struct {
	getEncounterCh      chan struct{ entity, id string }
	listEncountersCh    chan struct{ entity, id string }
	getPatientCh        chan struct{ entity, id string }
	listPatientsCh      chan struct{ entity, id string }
	getPractitionerCh   chan struct{ entity, id string }
	listPractitionersCh chan struct{ entity, id string }
	reviewCh            chan struct{ entity, id string }
	cooldownCh          chan struct{}
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
		getEncounterCh:      make(chan struct{ entity, id string }),
		listEncountersCh:    make(chan struct{ entity, id string }),
		getPatientCh:        make(chan struct{ entity, id string }),
		listPatientsCh:      make(chan struct{ entity, id string }),
		getPractitionerCh:   make(chan struct{ entity, id string }),
		listPractitionersCh: make(chan struct{ entity, id string }),
		reviewCh:            make(chan struct{ entity, id string }),
		cooldownCh:          make(chan struct{}),
	}

	tenantChannels[tenantID] = channels

	// Start worker goroutine
	go channels.processMessages()

	// Start 10-minute timer goroutine
	go func() {
		time.Sleep(10 * time.Minute)
		channels.cooldownCh <- struct{}{}
	}()

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

// Worker function that processes messages from channels
func (tc *TenantChannels) processMessages() {
	defer func() {
		// Graceful shutdown - close all channels
		close(tc.getEncounterCh)
		close(tc.listEncountersCh)
		close(tc.getPatientCh)
		close(tc.listPatientsCh)
		close(tc.getPractitionerCh)
		close(tc.listPractitionersCh)
		close(tc.reviewCh)
		close(tc.cooldownCh)
	}()

	for {
		select {
		case _, ok := <-tc.getEncounterCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process encounter request
			metrics.RecordChannelOperation("get_encounter", time.Since(start))
		case _, ok := <-tc.listEncountersCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process list encounters request
			metrics.RecordChannelOperation("list_encounters", time.Since(start))
		case _, ok := <-tc.getPatientCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process patient request
			metrics.RecordChannelOperation("get_patient", time.Since(start))
		case _, ok := <-tc.listPatientsCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process list patients request
			metrics.RecordChannelOperation("list_patients", time.Since(start))
		case _, ok := <-tc.getPractitionerCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process practitioner request
			metrics.RecordChannelOperation("get_practitioner", time.Since(start))
		case _, ok := <-tc.listPractitionersCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process list practitioners request
			metrics.RecordChannelOperation("list_practitioners", time.Since(start))
		case _, ok := <-tc.reviewCh:
			if !ok {
				continue
			}
			start := time.Now()
			// Process review request
			metrics.RecordChannelOperation("review_request", time.Since(start))
		case <-tc.cooldownCh:
			// Handle cooldown signal - stop goroutine
			return
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

package api

import (
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
	pseudoClosed        bool
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

// Global state management for all tenant channels
type TenantChannelManager struct {
	channels map[string]*TenantChannels
}

var tenantChannelManager = &TenantChannelManager{
	channels: make(map[string]*TenantChannels),
}

// AutoWarmUpTenant automatically warms up a tenant on first request
func AutoWarmUpTenant(tenantID string) *TenantChannels {
	// Check if channels exist
	channels, exists := tenantChannelManager.channels[tenantID]

	if exists {
		// Check if this tenant's channels are pseudo-closed
		if channels.pseudoClosed {
			// Just reset the flag, don't create new channels
			channels.pseudoClosed = false
			log.Info().
				Str("tenant", tenantID).
				Msg("Tenant channels pseudo-closed, resetting flag")
			// Restart both goroutines since they were stopped
			go channels.processMessages()
			go channels.manageTimer()
			return channels
		}
		return channels
	}

	// Double-check in case another goroutine created them while we were waiting
	if channels, exists := tenantChannelManager.channels[tenantID]; exists {
		if channels.pseudoClosed {
			channels.pseudoClosed = false
			log.Info().
				Str("tenant", tenantID).
				Msg("Tenant channels pseudo-closed, resetting flag")
			// Restart both goroutines since they were stopped
			go channels.processMessages()
			go channels.manageTimer()
			return channels
		}
		return channels
	}

	// Create channels for this tenant
	channels = &TenantChannels{
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
		pseudoClosed:        false,
	}

	tenantChannelManager.channels[tenantID] = channels

	// Start worker goroutine
	go channels.processMessages()

	// Start timer management goroutine
	go channels.manageTimer()

	log.Info().
		Str("tenant", tenantID).
		Msg("Tenant auto-warmed up on first request")

	return channels
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
	channels, exists := tenantChannelManager.channels[tenantID]
	return channels, exists
}

// ResetTimer resets the 10-minute timer for a tenant
func (tc *TenantChannels) ResetTimer() {
	select {
	case tc.timerResetCh <- struct{}{}:
		// Timer reset successfully
	default:
		// Channel might be pseudo-closed, ignore
	}
}

// SetPseudoClosed sets the pseudo-closed flag for a specific tenant
func (tc *TenantChannels) SetPseudoClosed() {
	tc.pseudoClosed = true
	log.Info().Msg("Tenant channels marked as pseudo-closed")
}

// CleanupAllChannels performs graceful shutdown cleanup
func CleanupAllChannels() {
	for tenantID, channels := range tenantChannelManager.channels {
		channels.cleanupChannels()
		log.Info().Str("tenant", tenantID).Msg("Tenant channels cleaned up")
	}

	// Clear the map
	tenantChannelManager.channels = make(map[string]*TenantChannels)
	log.Info().Msg("All tenant channels cleaned up during shutdown")
}

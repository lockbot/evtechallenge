package api

import (
	"context"
	"time"

	"stealthcompany.com/api-rest/internal/metrics"
)

// processMessages is the main worker function that processes messages from channels
func (tc *TenantChannels) processMessages() {
	defer tc.cleanupChannels()

	for {
		select {
		case msg, ok := <-tc.getEncounterCh:
			tc.handleChannelMessage(msg, ok, "get_encounter", tc.processGetEncounter)
		case msg, ok := <-tc.listEncountersCh:
			tc.handleChannelMessage(msg, ok, "list_encounters", tc.processListEncounters)
		case msg, ok := <-tc.getPatientCh:
			tc.handleChannelMessage(msg, ok, "get_patient", tc.processGetPatient)
		case msg, ok := <-tc.listPatientsCh:
			tc.handleChannelMessage(msg, ok, "list_patients", tc.processListPatients)
		case msg, ok := <-tc.getPractitionerCh:
			tc.handleChannelMessage(msg, ok, "get_practitioner", tc.processGetPractitioner)
		case msg, ok := <-tc.listPractitionersCh:
			tc.handleChannelMessage(msg, ok, "list_practitioners", tc.processListPractitioners)
		case msg, ok := <-tc.reviewCh:
			tc.handleChannelMessage(msg, ok, "review_request", tc.processReviewRequest)
		case <-tc.cooldownCh:
			// Handle cooldown signal - stop goroutine
			return
		}
	}
}

// handleChannelMessage is a helper function that processes a single channel message
func (tc *TenantChannels) handleChannelMessage(
	msg RequestMessage,
	ok bool,
	operation string,
	processor func(RequestMessage) ResponseMessage,
) {
	if !ok {
		return
	}

	start := time.Now()
	response := processor(msg)
	tc.sendResponse(msg.ResponseKey, response)
	metrics.RecordChannelOperation(operation, time.Since(start))
}

// sendResponse sends a response back through the response pool
func (tc *TenantChannels) sendResponse(responseKey string, response ResponseMessage) {
	if respCh, exists := tc.responsePool.GetChannelByKey(responseKey); exists {
		respCh.ch <- response
		tc.responsePool.ReturnChannel(respCh)
	}
}

// cleanupChannels gracefully closes all channels
func (tc *TenantChannels) cleanupChannels() {
	close(tc.getEncounterCh)
	close(tc.listEncountersCh)
	close(tc.getPatientCh)
	close(tc.listPatientsCh)
	close(tc.getPractitionerCh)
	close(tc.listPractitionersCh)
	close(tc.reviewCh)
	close(tc.cooldownCh)
	close(tc.timerResetCh)
}

// Processing functions for each request type

func (tc *TenantChannels) processGetEncounter(msg RequestMessage) ResponseMessage {
	data, err := getResourceByID(context.Background(), msg.TenantID, msg.Entity, msg.ID)
	return ResponseMessage{Data: data, Error: err}
}

func (tc *TenantChannels) processListEncounters(msg RequestMessage) ResponseMessage {
	data, err := listResources(context.Background(), msg.TenantID, msg.Entity, msg.Page, msg.Count)
	return ResponseMessage{Data: data, Error: err}
}

func (tc *TenantChannels) processGetPatient(msg RequestMessage) ResponseMessage {
	data, err := getResourceByID(context.Background(), msg.TenantID, msg.Entity, msg.ID)
	return ResponseMessage{Data: data, Error: err}
}

func (tc *TenantChannels) processListPatients(msg RequestMessage) ResponseMessage {
	data, err := listResources(context.Background(), msg.TenantID, msg.Entity, msg.Page, msg.Count)
	return ResponseMessage{Data: data, Error: err}
}

func (tc *TenantChannels) processGetPractitioner(msg RequestMessage) ResponseMessage {
	data, err := getResourceByID(context.Background(), msg.TenantID, msg.Entity, msg.ID)
	return ResponseMessage{Data: data, Error: err}
}

func (tc *TenantChannels) processListPractitioners(msg RequestMessage) ResponseMessage {
	data, err := listResources(context.Background(), msg.TenantID, msg.Entity, msg.Page, msg.Count)
	return ResponseMessage{Data: data, Error: err}
}

func (tc *TenantChannels) processReviewRequest(msg RequestMessage) ResponseMessage {
	// Parse entityID back to resourceType and resourceID
	resourceType := msg.Entity
	resourceID := msg.ID

	// If ID contains "/", extract the actual ID part
	if resourceID != "" && len(resourceID) > 0 {
		// The ID is already in format "ResourceType/ID", extract just the ID part
		if lastSlash := len(resourceID) - 1; lastSlash >= 0 {
			for i := lastSlash; i >= 0; i-- {
				if resourceID[i] == '/' {
					resourceID = resourceID[i+1:]
					break
				}
			}
		}
	}

	data, err := processReviewRequest(context.Background(), msg.TenantID, resourceType, resourceID)
	return ResponseMessage{Data: data, Error: err}
}

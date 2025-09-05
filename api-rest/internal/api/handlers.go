package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/metrics"
)

// HelloHandler returns a simple hello world message
func HelloHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := GetTenantFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Str("tenant", tenantID).
		Msg("Hello endpoint called")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{
		"message": "Hello, World!",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// AllGoodHandler expects {"yes": true} or returns business error
func AllGoodHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := GetTenantFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Str("tenant", tenantID).
		Msg("All-good endpoint called")

	if r.Method != http.MethodPost {
		log.Warn().
			Str("method", r.Method).
			Str("tenant", tenantID).
			Msg("Method not allowed on all-good endpoint")

		metrics.RecordAllGoodRequest("method_not_allowed")

		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	var req AllGoodRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant", tenantID).
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
			Str("tenant", tenantID).
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
		Str("tenant", tenantID).
		Msg("All good request processed successfully")

	metrics.RecordAllGoodRequest("success")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "All good!",
		"status":  "success",
	})
}

// GetResourceByIDHandler handles GET /{resource}/{id}
func GetResourceByIDHandler(resourceType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			log.Warn().
				Err(err).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Invalid tenant ID in request")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			log.Warn().
				Str("tenant", tenantID).
				Str("resourceType", resourceType).
				Msg("Missing resource ID in request")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing id"})
			return
		}

		// Check if tenant is warmed up and send to channel
		if channels, exists := GetTenantChannels(tenantID); exists {
			// Get response channel from pool
			respCh := channels.responsePool.GetChannel()
			responseKey := respCh.key

			// Send request to appropriate channel
			switch resourceType {
			case "Encounter":
				channels.getEncounterCh <- RequestMessage{tenantID, resourceType, id, responseKey, 0, 0}
			case "Patient":
				channels.getPatientCh <- RequestMessage{tenantID, resourceType, id, responseKey, 0, 0}
			case "Practitioner":
				channels.getPractitionerCh <- RequestMessage{tenantID, resourceType, id, responseKey, 0, 0}
			default:
				channels.responsePool.ReturnChannel(respCh)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "unsupported resource type"})
				return
			}

			// Wait for response from channel
			select {
			case response := <-respCh.ch:
				if response.Error != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": response.Error.Error()})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response.Data)
			case <-time.After(30 * time.Second):
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}
		} else {
			// Tenant not warmed up
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Tenant not warmed up",
				"message": "Please call /warm-up-tenant first",
			})
		}
	}
}

// ListResourcesHandler handles GET /{resource}
func ListResourcesHandler(resourceType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			log.Warn().
				Err(err).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Invalid tenant ID in request")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		// Parse pagination parameters
		countParam := r.URL.Query().Get("count")
		pageParam := r.URL.Query().Get("page")

		// Simple pagination validation (default values)
		page := 1
		count := 10
		if pageParam != "" {
			if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
				page = p
			}
		}
		if countParam != "" {
			if c, err := strconv.Atoi(countParam); err == nil && c > 0 && c <= 100 {
				count = c
			}
		}

		// Check if tenant is warmed up and send to channel
		if channels, exists := GetTenantChannels(tenantID); exists {
			// Get response channel from pool
			respCh := channels.responsePool.GetChannel()
			responseKey := respCh.key

			// Send request to appropriate channel
			switch resourceType {
			case "Encounter":
				channels.listEncountersCh <- RequestMessage{tenantID, resourceType, "", responseKey, page, count}
			case "Patient":
				channels.listPatientsCh <- RequestMessage{tenantID, resourceType, "", responseKey, page, count}
			case "Practitioner":
				channels.listPractitionersCh <- RequestMessage{tenantID, resourceType, "", responseKey, page, count}
			default:
				channels.responsePool.ReturnChannel(respCh)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "unsupported resource type"})
				return
			}

			// Wait for response from channel
			select {
			case response := <-respCh.ch:
				if response.Error != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": response.Error.Error()})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response.Data)
			case <-time.After(30 * time.Second):
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}
		} else {
			// Tenant not warmed up
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Tenant not warmed up",
				"message": "Please call /warm-up-tenant first",
			})
		}
	}
}

// ReviewRequestHandler handles POST /review-request
func ReviewRequestHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := GetTenantFromRequest(r)
	if err != nil {
		log.Warn().
			Err(err).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("Invalid tenant ID in request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	if r.Method != http.MethodPost {
		log.Warn().
			Str("method", r.Method).
			Str("tenant", tenantID).
			Msg("Method not allowed on review request endpoint")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	var req ReviewRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant", tenantID).
			Msg("Failed to decode review request JSON")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// Validate and normalize entity type
	entity := strings.ToLower(strings.TrimSpace(req.Entity))
	var resourceType string
	switch entity {
	case "encounter", "encounters":
		resourceType = "Encounter"
	case "patient", "patients":
		resourceType = "Patient"
	case "practitioner", "practitioners":
		resourceType = "Practitioner"
	default:
		log.Warn().
			Str("entity", req.Entity).
			Str("tenant", tenantID).
			Msg("Invalid entity type in review request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid entity"})
		return
	}

	if req.ID == "" {
		log.Warn().
			Str("tenant", tenantID).
			Str("resourceType", resourceType).
			Msg("Missing ID in review request")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing id"})
		return
	}

	// Check if tenant is warmed up and send to channel
	if channels, exists := GetTenantChannels(tenantID); exists {
		// Get response channel from pool
		respCh := channels.responsePool.GetChannel()
		responseKey := respCh.key

		// Send request to review channel with concatenated entity/ID
		entityID := resourceType + "/" + req.ID
		channels.reviewCh <- RequestMessage{tenantID, resourceType, entityID, responseKey, 0, 0}

		// Wait for response from channel
		select {
		case response := <-respCh.ch:
			if response.Error != nil {
				if strings.Contains(response.Error.Error(), "not found") {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]string{"error": "resource not found"})
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": response.Error.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response.Data)
		case <-time.After(30 * time.Second):
			http.Error(w, "Request timeout", http.StatusRequestTimeout)
		}
	} else {
		// Tenant not warmed up
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Tenant not warmed up",
			"message": "Please call /warm-up-tenant first",
		})
	}
}

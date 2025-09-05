package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/dal"
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
		start := time.Now()
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

		// Create connection and resource model
		conn, err := dal.GetConnOrGenConn()
		if err != nil {
			log.Error().
				Err(err).
				Str("tenant", tenantID).
				Msg("Failed to create database connection")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "database connection failed"})
			return
		}
		defer dal.ReturnConnection(conn) // Return connection to pool

		resourceModel := dal.NewResourceModel(conn)

		var doc map[string]interface{}
		switch resourceType {
		case "Encounter":
			encounterModel := dal.NewEncounterModel(resourceModel)
			doc, err = encounterModel.GetByID(r.Context(), id)
		case "Patient":
			patientModel := dal.NewPatientModel(resourceModel)
			doc, err = patientModel.GetByID(r.Context(), id)
		case "Practitioner":
			practitionerModel := dal.NewPractitionerModel(resourceModel)
			doc, err = practitionerModel.GetByID(r.Context(), id)
		default:
			log.Error().
				Str("resourceType", resourceType).
				Str("tenant", tenantID).
				Msg("Unsupported resource type")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "unsupported resource type"})
			return
		}

		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Debug().
					Str("id", id).
					Str("resourceType", resourceType).
					Str("tenant", tenantID).
					Msg("Resource not found")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "resource not found"})
				return
			}
			log.Error().
				Err(err).
				Str("id", id).
				Str("resourceType", resourceType).
				Str("tenant", tenantID).
				Msg("Failed to retrieve resource")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to retrieve resource"})
			return
		}

		// Check review status for this tenant
		reviewModel := dal.NewReviewModel(resourceModel)
		reviewInfo := reviewModel.GetReviewInfo(r.Context(), tenantID, resourceType, id)

		response := ResponseWithReview{
			Reviewed:   reviewInfo.Reviewed,
			ReviewTime: reviewInfo.ReviewTime,
			Data:       doc,
		}

		log.Info().
			Str("id", id).
			Str("resourceType", resourceType).
			Str("tenant", tenantID).
			Bool("reviewed", reviewInfo.Reviewed).
			Msg("Resource retrieved successfully")

		// Record performance metrics
		duration := time.Since(start)
		metrics.RecordHTTPRequest(r.Method, "/"+strings.ToLower(resourceType)+"/{id}", http.StatusOK, duration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ListResourcesHandler handles GET /{resource}
func ListResourcesHandler(resourceType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
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

		var page, count int
		var model interface {
			ValidatePaginationParams(string, string) (int, int, error)
			List(context.Context, int, int) (*dal.PaginatedResponse, error)
		}

		// Create connection and resource model
		conn, err := dal.GetConnOrGenConn()
		if err != nil {
			log.Error().
				Err(err).
				Str("tenant", tenantID).
				Msg("Failed to create database connection")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "database connection failed"})
			return
		}
		defer dal.ReturnConnection(conn) // Return connection to pool

		resourceModel := dal.NewResourceModel(conn)

		// Get appropriate model and validate pagination
		switch resourceType {
		case "Encounter":
			encounterModel := dal.NewEncounterModel(resourceModel)
			page, count, err = encounterModel.ValidatePaginationParams(pageParam, countParam)
			model = encounterModel
		case "Patient":
			patientModel := dal.NewPatientModel(resourceModel)
			page, count, err = patientModel.ValidatePaginationParams(pageParam, countParam)
			model = patientModel
		case "Practitioner":
			practitionerModel := dal.NewPractitionerModel(resourceModel)
			page, count, err = practitionerModel.ValidatePaginationParams(pageParam, countParam)
			model = practitionerModel
		default:
			log.Error().
				Str("resourceType", resourceType).
				Str("tenant", tenantID).
				Msg("Unsupported resource type")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "unsupported resource type"})
			return
		}

		if err != nil {
			log.Error().
				Err(err).
				Str("resourceType", resourceType).
				Str("tenant", tenantID).
				Msg("Failed to validate pagination parameters")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid pagination parameters"})
			return
		}

		// Get resources using the model
		paginatedResponse, err := model.List(r.Context(), page, count)
		if err != nil {
			log.Error().
				Err(err).
				Str("resourceType", resourceType).
				Str("tenant", tenantID).
				Int("page", page).
				Int("count", count).
				Msg("Failed to list resources")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "query failed"})
			return
		}

		// Add review information to each resource
		reviewModel := dal.NewReviewModel(resourceModel)
		for i := range paginatedResponse.Data {
			// Extract just the ID part from the document ID (remove resource type prefix)
			documentID := paginatedResponse.Data[i].ID
			resourceID := documentID
			if strings.Contains(documentID, "/") {
				parts := strings.Split(documentID, "/")
				resourceID = parts[len(parts)-1] // Get the last part (the actual ID)
			}

			reviewInfo := reviewModel.GetReviewInfo(r.Context(), tenantID, resourceType, resourceID)
			// Add review info to the Resource field so it appears in the JSON response
			paginatedResponse.Data[i].Resource["reviewed"] = reviewInfo.Reviewed
			if reviewInfo.Reviewed {
				paginatedResponse.Data[i].Resource["reviewTime"] = reviewInfo.ReviewTime
				paginatedResponse.Data[i].Resource["entityType"] = reviewInfo.EntityType
				paginatedResponse.Data[i].Resource["entityID"] = reviewInfo.EntityID
			}
		}

		log.Info().
			Str("resourceType", resourceType).
			Str("tenant", tenantID).
			Int("page", page).
			Int("count", count).
			Int("resultCount", len(paginatedResponse.Data)).
			Msg("Resources listed successfully")

		// Record performance metrics
		duration := time.Since(start)
		metrics.RecordHTTPRequest(r.Method, "/"+strings.ToLower(resourceType), http.StatusOK, duration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(paginatedResponse)
	}
}

// ReviewRequestHandler handles POST /review-request
func ReviewRequestHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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

	// Create connection and resource model
	conn, err := dal.GetConnOrGenConn()
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant", tenantID).
			Msg("Failed to create database connection")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "database connection failed"})
		return
	}
	defer dal.ReturnConnection(conn) // Return connection to pool

	resourceModel := dal.NewResourceModel(conn)
	reviewModel := dal.NewReviewModel(resourceModel)
	err = reviewModel.CreateReviewRequest(r.Context(), tenantID, resourceType, req.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Debug().
				Str("id", req.ID).
				Str("resourceType", resourceType).
				Str("tenant", tenantID).
				Msg("Resource not found for review request")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "resource not found"})
			return
		}

		log.Error().
			Err(err).
			Str("id", req.ID).
			Str("resourceType", resourceType).
			Str("tenant", tenantID).
			Msg("Failed to create review request")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to save review"})
		return
	}

	entityKey := resourceType + "/" + req.ID
	log.Info().
		Str("tenant", tenantID).
		Str("entityKey", entityKey).
		Msg("Review request created successfully")

	response := map[string]string{
		"status":   "review requested",
		"tenant":   tenantID,
		"entity":   entityKey,
		"reviewed": "true",
	}

	// Record performance metrics
	duration := time.Since(start)
	metrics.RecordHTTPRequest(r.Method, "/review-request", http.StatusOK, duration)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

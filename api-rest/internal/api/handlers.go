package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/couchbase/gocb/v2"
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
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing id"})
			return
		}
		bucket := GetBucket()
		if bucket == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "database not initialized"})
			return
		}
		// Get the tenant collection
		collection, err := GetTenantCollection(tenantID)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get tenant collection"})
			return
		}

		key := resourceType + "/" + id
		res, err := collection.Get(key, nil)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "resource not found"})
			return
		}
		var doc map[string]interface{}
		err = res.Content(&doc)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to decode document"})
			return
		}

		// Get review info directly from the document
		reviewed, _ := doc["reviewed"].(bool)
		reviewTime, _ := doc["reviewTime"].(string)

		response := ResponseWithReview{
			Reviewed:   reviewed,
			ReviewTime: reviewTime,
			Data:       doc,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ListResourcesHandler handles GET /{resource}
func ListResourcesHandler(resourceType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := GetTenantFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}
		cluster := GetCluster()
		if cluster == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "database not initialized"})
			return
		}
		// Pagination parameters
		countParam := r.URL.Query().Get("count")
		pageParam := r.URL.Query().Get("page")

		// Default values
		count := 100
		page := 1

		// Parse count parameter
		if countParam != "" {
			if v, err := strconv.Atoi(countParam); err == nil && v > 0 && v <= 10000 {
				count = v
			}
		}

		// Parse page parameter
		if pageParam != "" {
			if v, err := strconv.Atoi(pageParam); err == nil && v > 0 {
				page = v
			}
		}

		// Calculate offset
		offset := (page - 1) * count

		// Get the tenant collection for the query
		collection, err := GetTenantCollection(tenantID)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get tenant collection"})
			return
		}

		// Query the tenant collection directly
		// In Couchbase, we can query collections using the collection name in the FROM clause
		collectionName := collection.Name()
		q := "SELECT META(d).id AS id, d AS resource FROM `" + GetBucketName() + "`.`" + collectionName + "` AS d WHERE d.`resourceType` = $rt LIMIT " + strconv.Itoa(count) + " OFFSET " + strconv.Itoa(offset)
		rows, err := cluster.Query(q, &gocb.QueryOptions{NamedParameters: map[string]interface{}{"rt": resourceType}})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()
		var out []QueryRow
		for rows.Next() {
			var rr QueryRow
			err := rows.Row(&rr)
			if err != nil {
				continue
			}

			out = append(out, rr)
		}

		// Prepare paginated response
		response := map[string]interface{}{
			"data": out,
			"pagination": map[string]interface{}{
				"page":       page,
				"count":      count,
				"offset":     offset,
				"totalItems": len(out),
				"hasNext":    len(out) == count,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ReviewRequestHandler handles POST /review-request
func ReviewRequestHandler(w http.ResponseWriter, r *http.Request) {
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
	bucket := GetBucket()
	if bucket == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "database not initialized"})
		return
	}
	var req ReviewRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	entity := strings.ToLower(strings.TrimSpace(req.Entity))
	var rt string
	switch entity {
	case "encounter", "encounters":
		rt = "Encounter"
	case "patient", "patients":
		rt = "Patient"
	case "practitioner", "practitioners":
		rt = "Practitioner"
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid entity"})
		return
	}
	if req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing id"})
		return
	}

	// Create the review request
	err = CreateReviewRequest(tenantID, rt, req.ID)
	if err != nil {
		if strings.Contains(err.Error(), "resource not found") {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "resource not found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to save review"})
		return
	}

	entityKey := rt + "/" + req.ID
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "review requested",
		"tenant":   tenantID,
		"entity":   entityKey,
		"reviewed": "true",
	})
}

// HealthCheckHandler provides a simple health check endpoint
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "evtechallenge-api",
	})
}

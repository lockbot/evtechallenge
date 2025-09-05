package dal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// ResourceModel represents the database model for FHIR resources
type ResourceModel struct {
	conn *Connection
}

// NewResourceModel creates a new resource model
func NewResourceModel(conn *Connection) *ResourceModel {
	return &ResourceModel{
		conn: conn,
	}
}

// QueryRow represents a row from N1QL query results
type QueryRow struct {
	ID       string                 `json:"id"`
	Resource map[string]interface{} `json:"resource"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page  int
	Count int
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       []QueryRow             `json:"data"`
	Pagination map[string]interface{} `json:"pagination"`
}

// GetResource retrieves a FHIR resource from Couchbase
func (rm *ResourceModel) GetResource(ctx context.Context, docID string) (map[string]interface{}, error) {
	start := time.Now()
	result, err := rm.conn.GetBucket().DefaultCollection().Get(docID, &gocb.GetOptions{Context: ctx})
	duration := time.Since(start)

	if err != nil {
		log.Warn().
			Err(err).
			Str("doc_id", docID).
			Msg("Resource not found")
		return nil, fmt.Errorf("resource not found: %w", err)
	}

	var data map[string]interface{}
	err = result.Content(&data)
	if err != nil {
		log.Error().
			Err(err).
			Str("doc_id", docID).
			Msg("Failed to decode resource")
		return nil, fmt.Errorf("failed to decode resource: %w", err)
	}

	log.Debug().
		Str("doc_id", docID).
		Dur("duration", duration).
		Msg("Successfully retrieved resource")
	return data, nil
}

// ListResources retrieves a paginated list of resources
func (rm *ResourceModel) ListResources(ctx context.Context, resourceType string, params PaginationParams) (*PaginatedResponse, error) {
	// Validate and set defaults
	if params.Count <= 0 || params.Count > 10000 {
		params.Count = 100
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	offset := (params.Page - 1) * params.Count

	log.Debug().
		Str("resourceType", resourceType).
		Int("page", params.Page).
		Int("count", params.Count).
		Int("offset", offset).
		Msg("Querying resources")

	query := "SELECT META(d).id AS id, d AS resource FROM `" + rm.conn.GetBucketName() +
		"` AS d WHERE d.`resourceType` = $rt ORDER BY META(d).id LIMIT " + strconv.Itoa(params.Count) +
		" OFFSET " + strconv.Itoa(offset)

	rows, err := rm.conn.GetCluster().Query(query, &gocb.QueryOptions{
		Context:         ctx,
		NamedParameters: map[string]interface{}{"rt": resourceType},
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("query", query).
			Msg("Query failed")
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []QueryRow
	for rows.Next() {
		var row QueryRow
		err := rows.Row(&row)
		if err != nil {
			log.Warn().
				Err(err).
				Msg("Failed to decode query row")
			continue
		}
		results = append(results, row)
	}

	response := &PaginatedResponse{
		Data: results,
		Pagination: map[string]interface{}{
			"page":       params.Page,
			"count":      params.Count,
			"offset":     offset,
			"totalItems": len(results),
			"hasNext":    len(results) == params.Count,
		},
	}

	log.Debug().
		Str("resourceType", resourceType).
		Int("resultCount", len(results)).
		Msg("Resources queried successfully")

	return response, nil
}

// UpsertResource upserts a FHIR resource to Couchbase
func (rm *ResourceModel) UpsertResource(ctx context.Context, docID string, data map[string]interface{}) error {
	start := time.Now()
	_, err := rm.conn.GetBucket().DefaultCollection().Upsert(docID, data, &gocb.UpsertOptions{Context: ctx})
	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("doc_id", docID).
			Msg("Failed to upsert resource")
		return fmt.Errorf("failed to upsert resource %s: %w", docID, err)
	}

	log.Debug().
		Str("doc_id", docID).
		Dur("duration", duration).
		Msg("Successfully upserted resource")
	return nil
}

// ResourceExists checks if a resource exists in Couchbase
func (rm *ResourceModel) ResourceExists(ctx context.Context, docID string) (bool, error) {
	start := time.Now()
	_, err := rm.conn.GetBucket().DefaultCollection().Get(docID, &gocb.GetOptions{Context: ctx})
	duration := time.Since(start)

	if err != nil {
		// Check if it's a key not found error
		if strings.Contains(err.Error(), "key not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check resource existence %s: %w", docID, err)
	}

	log.Debug().
		Str("doc_id", docID).
		Dur("duration", duration).
		Msg("Resource exists")
	return true, nil
}

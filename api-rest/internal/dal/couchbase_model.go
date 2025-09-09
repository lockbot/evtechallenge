package dal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// executeQueryWithContext executes a N1QL query with proper tenant isolation
// Tenant isolation is handled by explicit bucket.scope.collection paths in queries
func executeQueryWithContext(ctx context.Context, conn *Connection, tenantScope, query string) (*gocb.QueryResult, error) {
	// Execute the query directly - tenant isolation is handled by explicit scope/collection paths
	return conn.GetCluster().Query(query, &gocb.QueryOptions{Context: ctx})
}

// ResourceModel represents the database model for FHIR resources
type ResourceModel struct {
	conn        *Connection
	tenantScope string
}

// NewResourceModel creates a new resource model
func NewResourceModel(conn *Connection) *ResourceModel {
	return &ResourceModel{
		conn:        conn,
		tenantScope: "_default", // Default to default scope
	}
}

// NewResourceModelWithTenant creates a new resource model for a specific tenant
func NewResourceModelWithTenant(conn *Connection, tenantScope string) *ResourceModel {
	return &ResourceModel{
		conn:        conn,
		tenantScope: tenantScope,
	}
}

// getCollectionForResource returns the appropriate collection for a resource type
func (rm *ResourceModel) getCollectionForResource(resourceType string) *gocb.Collection {
	scope := rm.conn.GetBucket().Scope(rm.tenantScope)

	switch resourceType {
	case "Encounter":
		return scope.Collection("encounters")
	case "Patient":
		return scope.Collection("patients")
	case "Practitioner":
		return scope.Collection("practitioners")
	default:
		// Fallback to default collection
		return scope.Collection("defaulty")
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
	// Extract resource type from docID (e.g., "Encounter/123" -> "Encounter")
	resourceType := strings.Split(docID, "/")[0]
	collection := rm.getCollectionForResource(resourceType)

	start := time.Now()
	result, err := collection.Get(docID, &gocb.GetOptions{Context: ctx})
	duration := time.Since(start)

	if err != nil {
		log.Warn().
			Err(err).
			Str("doc_id", docID).
			Str("tenant_scope", rm.tenantScope).
			Str("collection", resourceType).
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
		Str("tenant_scope", rm.tenantScope).
		Str("collection", resourceType).
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

	// Use scoped collection query instead of bucket-wide query
	collectionName := strings.ToLower(resourceType) + "s" // encounters, patients, practitioners
	query := fmt.Sprintf("SELECT META(d).id AS id, d AS resource FROM `%s`.`%s`.`%s` AS d ORDER BY META(d).id LIMIT %d OFFSET %d",
		rm.conn.GetBucketName(), rm.tenantScope, collectionName, params.Count, offset)

	rows, err := executeQueryWithContext(ctx, rm.conn, rm.tenantScope, query)
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
	// Extract resource type from docID (e.g., "Encounter/123" -> "Encounter")
	resourceType := strings.Split(docID, "/")[0]
	collection := rm.getCollectionForResource(resourceType)

	start := time.Now()
	_, err := collection.Upsert(docID, data, &gocb.UpsertOptions{Context: ctx})
	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("doc_id", docID).
			Str("tenant_scope", rm.tenantScope).
			Str("collection", resourceType).
			Msg("Failed to upsert resource")
		return fmt.Errorf("failed to upsert resource %s: %w", docID, err)
	}

	log.Debug().
		Str("doc_id", docID).
		Str("tenant_scope", rm.tenantScope).
		Str("collection", resourceType).
		Dur("duration", duration).
		Msg("Successfully upserted resource")
	return nil
}

// ResourceExists checks if a resource exists in Couchbase
func (rm *ResourceModel) ResourceExists(ctx context.Context, docID string) (bool, error) {
	// Extract resource type from docID (e.g., "Encounter/123" -> "Encounter")
	resourceType := strings.Split(docID, "/")[0]
	collection := rm.getCollectionForResource(resourceType)

	start := time.Now()
	_, err := collection.Get(docID, &gocb.GetOptions{Context: ctx})
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
		Str("tenant_scope", rm.tenantScope).
		Str("collection", resourceType).
		Dur("duration", duration).
		Msg("Resource exists")
	return true, nil
}

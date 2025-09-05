package dal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir-client/internal/metrics"
)

var (
	indexesCreated bool
	indexMutex     sync.Mutex
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

// createIndexesIfNeeded creates secondary indexes if they don't exist
func (rm *ResourceModel) createIndexesIfNeeded(ctx context.Context) error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if indexesCreated {
		return nil
	}

	log.Info().Msg("Creating secondary indexes for efficient querying...")

	// Create indexes using N1QL queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_resourceType ON `" + rm.conn.GetBucketName() + "`(resourceType)",
		"CREATE INDEX IF NOT EXISTS idx_id ON `" + rm.conn.GetBucketName() + "`(id)",
		"CREATE INDEX IF NOT EXISTS idx_subjectPatientId ON `" + rm.conn.GetBucketName() + "`(subjectPatientId)",
		"CREATE INDEX IF NOT EXISTS idx_practitionerIds ON `" + rm.conn.GetBucketName() + "`(practitionerIds)",
		"CREATE INDEX IF NOT EXISTS idx_resourceType_id ON `" + rm.conn.GetBucketName() + "`(resourceType, id)",
	}

	for _, indexQuery := range indexes {
		_, err := rm.conn.GetCluster().Query(indexQuery, &gocb.QueryOptions{Context: ctx})
		if err != nil {
			log.Warn().Err(err).Str("query", indexQuery).Msg("Failed to create index (may already exist)")
		} else {
			log.Debug().Str("query", indexQuery).Msg("Index created successfully")
		}
	}

	indexesCreated = true
	log.Info().Msg("Secondary indexes creation completed")
	return nil
}

// UpsertResource upserts a FHIR resource to Couchbase
func (rm *ResourceModel) UpsertResource(ctx context.Context, docID string, data map[string]interface{}) error {
	// Create indexes on first upsert
	if err := rm.createIndexesIfNeeded(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to create indexes, continuing with upsert")
	}

	start := time.Now()
	_, err := rm.conn.bucket.DefaultCollection().Upsert(docID, data, nil)
	duration := time.Since(start)

	if err != nil {
		metrics.RecordCouchbaseOperation("upsert", "error")
		metrics.RecordCouchbaseOperationDuration("upsert", duration)
		return fmt.Errorf("failed to upsert resource %s: %w", docID, err)
	}

	metrics.RecordCouchbaseOperation("upsert", "success")
	metrics.RecordCouchbaseOperationDuration("upsert", duration)

	log.Debug().Str("doc_id", docID).Msg("Successfully upserted resource")
	return nil
}

// GetResource retrieves a FHIR resource from Couchbase
func (rm *ResourceModel) GetResource(ctx context.Context, docID string) (map[string]interface{}, error) {
	start := time.Now()
	result, err := rm.conn.bucket.DefaultCollection().Get(docID, nil)
	duration := time.Since(start)

	if err != nil {
		metrics.RecordCouchbaseOperation("get", "error")
		metrics.RecordCouchbaseOperationDuration("get", duration)
		return nil, fmt.Errorf("failed to get resource %s: %w", docID, err)
	}

	var data map[string]interface{}
	err = result.Content(&data)
	if err != nil {
		metrics.RecordCouchbaseOperation("get", "error")
		metrics.RecordCouchbaseOperationDuration("get", duration)
		return nil, fmt.Errorf("failed to decode resource %s: %w", docID, err)
	}

	metrics.RecordCouchbaseOperation("get", "success")
	metrics.RecordCouchbaseOperationDuration("get", duration)

	log.Debug().Str("doc_id", docID).Msg("Successfully retrieved resource")
	return data, nil
}

// ResourceExists checks if a resource exists in Couchbase
func (rm *ResourceModel) ResourceExists(ctx context.Context, docID string) (bool, error) {
	start := time.Now()
	_, err := rm.conn.bucket.DefaultCollection().Get(docID, nil)
	duration := time.Since(start)

	if err != nil {
		// Check if it's a key not found error
		if err.Error() == "key not found" || err.Error() == "document not found" {
			metrics.RecordCouchbaseOperation("get", "miss")
			metrics.RecordCouchbaseOperationDuration("get", duration)
			return false, nil
		}
		metrics.RecordCouchbaseOperation("get", "error")
		metrics.RecordCouchbaseOperationDuration("get", duration)
		return false, fmt.Errorf("failed to check resource existence %s: %w", docID, err)
	}

	metrics.RecordCouchbaseOperation("get", "success")
	metrics.RecordCouchbaseOperationDuration("get", duration)
	return true, nil
}

// CountResourcesByType counts resources by resource type
func (rm *ResourceModel) CountResourcesByType(ctx context.Context, resourceType string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) as count FROM `%s` WHERE `resourceType` = '%s'", rm.conn.bucketName, resourceType)
	rows, err := rm.conn.cluster.Query(query, nil)
	if err != nil {
		log.Warn().Err(err).Str("resource_type", resourceType).Msg("Failed to count resources")
		return 0, fmt.Errorf("failed to count resources of type %s: %w", resourceType, err)
	}
	defer rows.Close()

	var result struct {
		Count int64 `json:"count"`
	}
	if rows.Next() {
		err = rows.Row(&result)
		if err != nil {
			log.Warn().Err(err).Str("resource_type", resourceType).Msg("Failed to read resource count")
			return 0, fmt.Errorf("failed to read resource count for type %s: %w", resourceType, err)
		}
	}

	log.Debug().Str("resource_type", resourceType).Int64("count", result.Count).Msg("Counted resources")
	return result.Count, nil
}

// GetAllResourcesByType retrieves all resources of a specific type
func (rm *ResourceModel) GetAllResourcesByType(ctx context.Context, resourceType string) ([]ResourceRow, error) {
	query := fmt.Sprintf("SELECT META(d).id AS id, d AS resource FROM `%s` AS d WHERE d.`resourceType` = $1", rm.conn.bucketName)
	rows, err := rm.conn.cluster.Query(query, &gocb.QueryOptions{
		PositionalParameters: []interface{}{resourceType},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query resources of type %s: %w", resourceType, err)
	}
	defer rows.Close()

	var resources []ResourceRow
	for rows.Next() {
		var row ResourceRow
		err = rows.Row(&row)
		if err != nil {
			log.Warn().Err(err).Str("resource_type", resourceType).Msg("Failed to read resource row")
			continue
		}
		resources = append(resources, row)
	}

	log.Debug().Str("resource_type", resourceType).Int("count", len(resources)).Msg("Retrieved resources")
	return resources, nil
}

// ResourceRow represents a row from a resource query
type ResourceRow struct {
	ID       string                 `json:"id"`
	Resource map[string]interface{} `json:"resource"`
}

package dal

import (
	"context"
	"fmt"
	"strings"
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

// createCollectionsAndIndexesIfNeeded creates collections and indexes if they don't exist
func (rm *ResourceModel) createCollectionsAndIndexesIfNeeded(ctx context.Context) error {
	indexMutex.Lock()
	defer indexMutex.Unlock()

	if indexesCreated {
		return nil
	}

	log.Info().Msg("Creating collections and indexes for FHIR resources...")

	// Create collections using N1QL with full bucket.scope.collection syntax
	bucketName := rm.conn.GetBucketName()
	collections := []string{
		fmt.Sprintf("CREATE COLLECTION `%s`.`_default`.`encounters`", bucketName),
		fmt.Sprintf("CREATE COLLECTION `%s`.`_default`.`patients`", bucketName),
		fmt.Sprintf("CREATE COLLECTION `%s`.`_default`.`practitioners`", bucketName),
	}

	for _, collectionQuery := range collections {
		_, err := rm.conn.GetCluster().Query(collectionQuery, &gocb.QueryOptions{Context: ctx})
		if err != nil {
			log.Warn().Err(err).Str("query", collectionQuery).Msg("Failed to create collection (may already exist)")
		} else {
			log.Debug().Str("query", collectionQuery).Msg("Collection created successfully")
		}
	}

	// Create indexes using N1QL queries for each collection
	indexes := []string{
		// Indexes for encounters collection
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_encounters_id ON `%s`.`_default`.`encounters`(id)", bucketName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_encounters_subjectPatientId ON `%s`.`_default`.`encounters`(subjectPatientId)", bucketName),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_encounters_practitionerIds ON `%s`.`_default`.`encounters`(practitionerIds)", bucketName),

		// Indexes for patients collection
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_patients_id ON `%s`.`_default`.`patients`(id)", bucketName),

		// Indexes for practitioners collection
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_practitioners_id ON `%s`.`_default`.`practitioners`(id)", bucketName),
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
	log.Info().Msg("Collections and indexes creation completed")
	return nil
}

// UpsertResource upserts a FHIR resource to Couchbase
func (rm *ResourceModel) UpsertResource(ctx context.Context, docID string, data map[string]interface{}) error {
	// Create collections and indexes on first upsert
	if err := rm.createCollectionsAndIndexesIfNeeded(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to create collections and indexes, continuing with upsert")
	}

	// Add reviewed field to all resources during FHIR ingestion
	data["reviewed"] = false

	// Get the appropriate collection based on resource type
	collection, err := rm.getCollectionForResource(docID)
	if err != nil {
		return fmt.Errorf("failed to get collection for resource %s: %w", docID, err)
	}

	start := time.Now()
	_, err = collection.Upsert(docID, data, nil)
	duration := time.Since(start)

	if err != nil {
		metrics.RecordCouchbaseOperation("upsert", "error")
		metrics.RecordCouchbaseOperationDuration("upsert", duration)
		return fmt.Errorf("failed to upsert resource %s: %w", docID, err)
	}

	// Add denormalized fields using merge for encounters
	if err := rm.mergeDenormalizedFields(collection, docID); err != nil {
		log.Warn().Err(err).Str("doc_id", docID).Msg("Failed to merge denormalized fields")
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

// getCollectionForResource returns the appropriate collection based on resource type
func (rm *ResourceModel) getCollectionForResource(docID string) (*gocb.Collection, error) {
	// Extract resource type from document ID (format: "ResourceType/ID")
	var collectionName string
	if len(docID) > 0 {
		for i, char := range docID {
			if char == '/' {
				collectionName = docID[:i]
				break
			}
		}
	}

	// Map resource types to collection names
	switch collectionName {
	case "Encounter":
		return rm.conn.bucket.Scope("_default").Collection("encounters"), nil
	case "Patient":
		return rm.conn.bucket.Scope("_default").Collection("patients"), nil
	case "Practitioner":
		return rm.conn.bucket.Scope("_default").Collection("practitioners"), nil
	default:
		return nil, fmt.Errorf("unknown resource type: %s", collectionName)
	}
}

// mergeDenormalizedFields merges denormalized fields into the document for better querying
func (rm *ResourceModel) mergeDenormalizedFields(collection *gocb.Collection, docID string) error {
	// Extract resource type from docID (format: "ResourceType/id")
	parts := strings.Split(docID, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid docID format: %s", docID)
	}

	resourceType := parts[0]

	// Add denormalized fields for encounters
	if resourceType == "Encounter" {
		// First, get the current document to extract references
		result, err := collection.Get(docID, nil)
		if err != nil {
			return fmt.Errorf("failed to get document for denormalization: %w", err)
		}

		var data map[string]interface{}
		if err := result.Content(&data); err != nil {
			return fmt.Errorf("failed to parse document content: %w", err)
		}

		// Prepare merge data
		mergeData := map[string]interface{}{
			"docId": docID,
		}

		// Extract and add subjectPatientId (keep full reference format)
		if subject, ok := data["subject"].(map[string]interface{}); ok {
			if reference, ok := subject["reference"].(string); ok {
				if strings.HasPrefix(reference, "Patient/") {
					mergeData["subjectPatientId"] = reference
				}
			}
		}

		// Extract and add practitionerIds array (keep full reference format)
		var practitionerIDs []string
		if participants, ok := data["participant"].([]interface{}); ok {
			for _, participant := range participants {
				if p, ok := participant.(map[string]interface{}); ok {
					if individual, ok := p["individual"].(map[string]interface{}); ok {
						if reference, ok := individual["reference"].(string); ok {
							if strings.HasPrefix(reference, "Practitioner/") {
								practitionerIDs = append(practitionerIDs, reference)
							}
						}
					}
				}
			}
		}
		mergeData["practitionerIds"] = practitionerIDs

		// Merge the denormalized fields
		_, err = collection.MutateIn(docID, []gocb.MutateInSpec{
			gocb.UpsertSpec("docId", mergeData["docId"], nil),
			gocb.UpsertSpec("subjectPatientId", mergeData["subjectPatientId"], nil),
			gocb.UpsertSpec("practitionerIds", mergeData["practitionerIds"], nil),
		}, nil)

		if err != nil {
			return fmt.Errorf("failed to merge denormalized fields: %w", err)
		}
	}

	return nil
}

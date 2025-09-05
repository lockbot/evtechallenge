package dal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

var (
	reviewIndexesCreated bool
	reviewIndexMutex     sync.Mutex
)

// ReviewDocument represents a tenant's review list
type ReviewDocument struct {
	TenantID      string                 `json:"tenantId"`
	Encounters    map[string]interface{} `json:"encounters"`    // key: "Encounter/ID", value: review metadata
	Patients      map[string]interface{} `json:"patients"`      // key: "Patient/ID", value: review metadata
	Practitioners map[string]interface{} `json:"practitioners"` // key: "Practitioner/ID", value: review metadata
	Updated       time.Time              `json:"updated"`
}

// ReviewInfo contains review status and metadata
type ReviewInfo struct {
	Reviewed   bool   `json:"reviewed"`
	ReviewTime string `json:"reviewTime,omitempty"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   string `json:"entityID,omitempty"`
}

// ReviewModel handles review-specific database operations
type ReviewModel struct {
	resourceModel *ResourceModel
}

// NewReviewModel creates a new review model instance
func NewReviewModel(resourceModel *ResourceModel) *ReviewModel {
	return &ReviewModel{resourceModel: resourceModel}
}

// GetReviewInfo checks if a resource is reviewed for a tenant and returns review metadata
func (rm *ReviewModel) GetReviewInfo(ctx context.Context, tenantID, resourceType, resourceID string) ReviewInfo {

	reviewKey := fmt.Sprintf("Review/%s", tenantID)
	log.Debug().
		Str("tenantID", tenantID).
		Str("resourceType", resourceType).
		Str("resourceID", resourceID).
		Str("reviewKey", reviewKey).
		Msg("Getting review info")

	var reviewDoc ReviewDocument
	res, err := rm.resourceModel.conn.GetBucket().DefaultCollection().Get(reviewKey, &gocb.GetOptions{Context: ctx})
	if err != nil {
		log.Debug().
			Err(err).
			Str("reviewKey", reviewKey).
			Msg("Review document not found")
		return ReviewInfo{Reviewed: false}
	}

	err = res.Content(&reviewDoc)
	if err != nil {
		log.Error().
			Err(err).
			Str("reviewKey", reviewKey).
			Msg("Failed to decode review document")
		return ReviewInfo{Reviewed: false}
	}

	// Get the appropriate map based on resource type
	var reviewData map[string]interface{}
	switch resourceType {
	case "Encounter":
		reviewData = reviewDoc.Encounters
	case "Patient":
		reviewData = reviewDoc.Patients
	case "Practitioner":
		reviewData = reviewDoc.Practitioners
	default:
		log.Warn().
			Str("resourceType", resourceType).
			Msg("Unsupported resource type for review")
		return ReviewInfo{Reviewed: false}
	}

	// If the map is nil, no reviews exist for this resource type
	if reviewData == nil {
		log.Debug().
			Str("resourceType", resourceType).
			Msg("Review data map is nil")
		return ReviewInfo{Reviewed: false}
	}

	entityKey := fmt.Sprintf("%s/%s", resourceType, resourceID)
	reviewDataItem, exists := reviewData[entityKey]
	if !exists {
		log.Debug().
			Str("entityKey", entityKey).
			Msg("Entity key not found in review data")
		return ReviewInfo{Reviewed: false}
	}

	reviewMap, ok := reviewDataItem.(map[string]interface{})
	if !ok {
		return ReviewInfo{Reviewed: true, EntityType: resourceType, EntityID: resourceID}
	}

	reviewTime := ""
	if rt, ok := reviewMap["reviewTime"].(string); ok {
		reviewTime = rt
	}

	log.Debug().
		Str("entityKey", entityKey).
		Bool("reviewed", true).
		Str("reviewTime", reviewTime).
		Msg("Review info found")

	return ReviewInfo{
		Reviewed:   true,
		ReviewTime: reviewTime,
		EntityType: resourceType,
		EntityID:   resourceID,
	}
}

// createReviewIndexesIfNeeded creates secondary indexes for review queries if they don't exist
func (rm *ReviewModel) createReviewIndexesIfNeeded(ctx context.Context) error {
	reviewIndexMutex.Lock()
	defer reviewIndexMutex.Unlock()

	if reviewIndexesCreated {
		return nil
	}

	log.Info().Msg("Creating secondary indexes for review queries...")

	// Create indexes using N1QL queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tenantId ON `" + rm.resourceModel.conn.GetBucketName() + "`(tenantId)",
		"CREATE INDEX IF NOT EXISTS idx_resourceType ON `" + rm.resourceModel.conn.GetBucketName() + "`(resourceType)",
		"CREATE INDEX IF NOT EXISTS idx_id ON `" + rm.resourceModel.conn.GetBucketName() + "`(id)",
		"CREATE INDEX IF NOT EXISTS idx_resourceType_id ON `" + rm.resourceModel.conn.GetBucketName() + "`(resourceType, id)",
	}

	for _, indexQuery := range indexes {
		_, err := rm.resourceModel.conn.GetCluster().Query(indexQuery, &gocb.QueryOptions{Context: ctx})
		if err != nil {
			log.Warn().Err(err).Str("query", indexQuery).Msg("Failed to create index (may already exist)")
		} else {
			log.Debug().Str("query", indexQuery).Msg("Index created successfully")
		}
	}

	reviewIndexesCreated = true
	log.Info().Msg("Review indexes creation completed")
	return nil
}

// CreateReviewRequest creates or updates a review for a resource
func (rm *ReviewModel) CreateReviewRequest(ctx context.Context, tenantID, resourceType, resourceID string) error {
	// Create indexes on first review creation
	if err := rm.createReviewIndexesIfNeeded(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to create review indexes, continuing with review creation")
	}
	// Verify the resource exists
	resourceKey := resourceType + "/" + resourceID
	exists, err := rm.resourceModel.ResourceExists(ctx, resourceKey)
	if err != nil {
		log.Error().
			Err(err).
			Str("resourceKey", resourceKey).
			Msg("Failed to check resource existence")
		return fmt.Errorf("failed to verify resource: %w", err)
	}
	if !exists {
		log.Warn().
			Str("resourceKey", resourceKey).
			Msg("Resource not found")
		return fmt.Errorf("resource not found")
	}

	// Get or create review document for this tenant
	reviewKey := fmt.Sprintf("Review/%s", tenantID)
	log.Debug().
		Str("tenantID", tenantID).
		Str("resourceType", resourceType).
		Str("resourceID", resourceID).
		Str("reviewKey", reviewKey).
		Msg("Creating review request")

	var reviewDoc ReviewDocument
	res, err := rm.resourceModel.conn.GetBucket().DefaultCollection().Get(reviewKey, &gocb.GetOptions{Context: ctx})
	if err != nil {
		// Create new review document
		reviewDoc = ReviewDocument{
			TenantID:      tenantID,
			Encounters:    make(map[string]interface{}),
			Patients:      make(map[string]interface{}),
			Practitioners: make(map[string]interface{}),
			Updated:       time.Now().UTC(),
		}
		log.Debug().
			Str("reviewKey", reviewKey).
			Msg("Creating new review document")
	} else {
		err = res.Content(&reviewDoc)
		if err != nil {
			log.Error().
				Err(err).
				Str("reviewKey", reviewKey).
				Msg("Failed to decode review document")
			return fmt.Errorf("failed to decode review document: %w", err)
		}

		// Initialize maps if they don't exist
		if reviewDoc.Encounters == nil {
			reviewDoc.Encounters = make(map[string]interface{})
		}
		if reviewDoc.Patients == nil {
			reviewDoc.Patients = make(map[string]interface{})
		}
		if reviewDoc.Practitioners == nil {
			reviewDoc.Practitioners = make(map[string]interface{})
		}
	}

	// Add review entry to the appropriate map
	entityKey := fmt.Sprintf("%s/%s", resourceType, resourceID)
	reviewEntry := map[string]interface{}{
		"reviewRequested": true,
		"reviewTime":      time.Now().UTC().Format(time.RFC3339),
		"entityType":      resourceType,
		"entityID":        resourceID,
	}

	switch resourceType {
	case "Encounter":
		reviewDoc.Encounters[entityKey] = reviewEntry
	case "Patient":
		reviewDoc.Patients[entityKey] = reviewEntry
	case "Practitioner":
		reviewDoc.Practitioners[entityKey] = reviewEntry
	default:
		log.Error().
			Str("resourceType", resourceType).
			Msg("Unsupported resource type for review")
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	reviewDoc.Updated = time.Now().UTC()

	// Upsert the review document
	reviewDocMap := map[string]interface{}{
		"tenantID":      reviewDoc.TenantID,
		"encounters":    reviewDoc.Encounters,
		"patients":      reviewDoc.Patients,
		"practitioners": reviewDoc.Practitioners,
		"updated":       reviewDoc.Updated,
	}
	err = rm.resourceModel.UpsertResource(ctx, reviewKey, reviewDocMap)
	if err != nil {
		log.Error().
			Err(err).
			Str("reviewKey", reviewKey).
			Msg("Failed to save review document")
		return fmt.Errorf("failed to save review: %w", err)
	}

	log.Info().
		Str("tenantID", tenantID).
		Str("entityKey", entityKey).
		Msg("Review request created successfully")

	return nil
}

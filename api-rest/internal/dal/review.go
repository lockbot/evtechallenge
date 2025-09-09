package dal

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// ReviewInfo contains review status and metadata embedded in resource documents
type ReviewInfo struct {
	Reviewed   bool   `json:"reviewed"`
	ReviewTime string `json:"reviewTime,omitempty"`
}

// ReviewModel handles review-specific database operations using embedded fields
type ReviewModel struct {
	resourceModel *ResourceModel
}

// NewReviewModel creates a new review model instance
func NewReviewModel(resourceModel *ResourceModel) *ReviewModel {
	return &ReviewModel{resourceModel: resourceModel}
}

// NewReviewModelWithTenant creates a new review model instance for a specific tenant
func NewReviewModelWithTenant(conn *Connection, tenantScope string) *ReviewModel {
	resourceModel := NewResourceModelWithTenant(conn, tenantScope)
	return &ReviewModel{resourceModel: resourceModel}
}

// GetReviewInfo checks if a resource is reviewed and returns review metadata from embedded fields
func (rm *ReviewModel) GetReviewInfo(ctx context.Context, tenantID, resourceType, resourceID string) ReviewInfo {
	docID := fmt.Sprintf("%s/%s", resourceType, resourceID)

	log.Debug().
		Str("tenantID", tenantID).
		Str("resourceType", resourceType).
		Str("resourceID", resourceID).
		Str("docID", docID).
		Msg("Getting review info from embedded fields")

	// Get the resource document
	resourceData, err := rm.resourceModel.GetResource(ctx, docID)
	if err != nil {
		log.Debug().
			Err(err).
			Str("docID", docID).
			Msg("Resource not found")
		return ReviewInfo{Reviewed: false}
	}

	// Check for embedded review fields
	reviewed, ok := resourceData["reviewed"].(bool)
	if !ok {
		log.Debug().
			Str("docID", docID).
			Msg("No review field found, resource not reviewed")
		return ReviewInfo{Reviewed: false}
	}

	reviewTime := ""
	if rt, ok := resourceData["reviewTime"].(string); ok {
		reviewTime = rt
	}

	log.Debug().
		Str("docID", docID).
		Bool("reviewed", reviewed).
		Str("reviewTime", reviewTime).
		Msg("Review info found in embedded fields")

	return ReviewInfo{
		Reviewed:   reviewed,
		ReviewTime: reviewTime,
	}
}

// CreateReviewRequest creates or updates a review for a resource by embedding review fields
func (rm *ReviewModel) CreateReviewRequest(ctx context.Context, tenantID, resourceType, resourceID string) error {
	docID := fmt.Sprintf("%s/%s", resourceType, resourceID)

	log.Debug().
		Str("tenantID", tenantID).
		Str("resourceType", resourceType).
		Str("resourceID", resourceID).
		Str("docID", docID).
		Msg("Creating review request with embedded fields")

	// Verify the resource exists
	exists, err := rm.resourceModel.ResourceExists(ctx, docID)
	if err != nil {
		log.Error().
			Err(err).
			Str("docID", docID).
			Msg("Failed to check resource existence")
		return fmt.Errorf("failed to verify resource: %w", err)
	}
	if !exists {
		log.Warn().
			Str("docID", docID).
			Msg("Resource not found")
		return fmt.Errorf("resource not found")
	}

	// Get the current resource document
	resourceData, err := rm.resourceModel.GetResource(ctx, docID)
	if err != nil {
		log.Error().
			Err(err).
			Str("docID", docID).
			Msg("Failed to get resource for review update")
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Add embedded review fields
	resourceData["reviewed"] = true
	resourceData["reviewTime"] = time.Now().UTC().Format(time.RFC3339)

	// Update the resource document with embedded review fields
	err = rm.resourceModel.UpsertResource(ctx, docID, resourceData)
	if err != nil {
		log.Error().
			Err(err).
			Str("docID", docID).
			Msg("Failed to update resource with review fields")
		return fmt.Errorf("failed to update resource with review: %w", err)
	}

	log.Info().
		Str("tenantID", tenantID).
		Str("docID", docID).
		Msg("Review request created successfully with embedded fields")

	return nil
}

package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GetTenantFromRequest extracts tenant ID from request headers
func GetTenantFromRequest(r *http.Request) (string, error) {
	tenant := r.Header.Get(TenantHeaderKey)
	if tenant == "" {
		return "", fmt.Errorf("missing required header: %s", TenantHeaderKey)
	}
	trimmedTenant := strings.TrimSpace(tenant)
	if trimmedTenant == "" {
		return "", fmt.Errorf("tenant ID cannot be empty")
	}
	return trimmedTenant, nil
}

// GetReviewInfo checks if a resource is reviewed for a tenant and returns review metadata
func GetReviewInfo(tenantID, resourceType, resourceID string) ReviewInfo {
	bucket := GetBucket()
	if bucket == nil {
		return ReviewInfo{Reviewed: false}
	}
	reviewKey := fmt.Sprintf("Review/%s", tenantID)
	var reviewDoc ReviewDocument
	res, err := bucket.DefaultCollection().Get(reviewKey, nil)
	if err != nil {
		return ReviewInfo{Reviewed: false}
	}
	err = res.Content(&reviewDoc)
	if err != nil {
		return ReviewInfo{Reviewed: false}
	}
	entityKey := fmt.Sprintf("%s/%s", resourceType, resourceID)
	reviewData, exists := reviewDoc.Reviews[entityKey]
	if !exists {
		return ReviewInfo{Reviewed: false}
	}

	reviewMap, ok := reviewData.(map[string]interface{})
	if !ok {
		return ReviewInfo{Reviewed: true, EntityType: resourceType, EntityID: resourceID}
	}

	reviewTime := ""
	if rt, ok := reviewMap["reviewTime"].(string); ok {
		reviewTime = rt
	}

	return ReviewInfo{
		Reviewed:   true,
		ReviewTime: reviewTime,
		EntityType: resourceType,
		EntityID:   resourceID,
	}
}

// CreateReviewRequest creates or updates a review for a resource
func CreateReviewRequest(tenantID, resourceType, resourceID string) error {
	bucket := GetBucket()
	if bucket == nil {
		return fmt.Errorf("database not initialized")
	}

	// Verify the resource exists
	resourceKey := resourceType + "/" + resourceID
	_, err := bucket.DefaultCollection().Get(resourceKey, nil)
	if err != nil {
		return fmt.Errorf("resource not found: %w", err)
	}

	// Get or create review document for this tenant
	reviewKey := fmt.Sprintf("Review/%s", tenantID)
	var reviewDoc ReviewDocument
	res, err := bucket.DefaultCollection().Get(reviewKey, nil)
	if err != nil {
		// Create new review document
		reviewDoc = ReviewDocument{
			TenantID: tenantID,
			Reviews:  make(map[string]interface{}),
			Updated:  time.Now().UTC(),
		}
	} else {
		err = res.Content(&reviewDoc)
		if err != nil {
			return fmt.Errorf("failed to decode review document: %w", err)
		}
	}

	// Add review entry
	entityKey := fmt.Sprintf("%s/%s", resourceType, resourceID)
	reviewDoc.Reviews[entityKey] = map[string]interface{}{
		"reviewRequested": true,
		"reviewTime":      time.Now().UTC().Format(time.RFC3339),
		"entityType":      resourceType,
		"entityID":        resourceID,
	}
	reviewDoc.Updated = time.Now().UTC()

	// Upsert the review document
	_, err = bucket.DefaultCollection().Upsert(reviewKey, reviewDoc, nil)
	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}

	return nil
}

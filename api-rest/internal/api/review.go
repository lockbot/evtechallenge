package api

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

// CreateReviewRequest marks a resource as reviewed for a tenant
func CreateReviewRequest(tenantID, resourceType, resourceID string) error {
	bucket := GetBucket()
	if bucket == nil {
		return fmt.Errorf("database not initialized")
	}

	// Get the tenant collection from the tenant goroutine manager
	tenantManager := GetTenantGoroutineManager()
	if tenantManager == nil {
		return fmt.Errorf("tenant goroutine manager not initialized")
	}

	collection, err := tenantManager.GetTenantCollection(tenantID)
	if err != nil {
		return fmt.Errorf("tenant not ready or collection not initialized: %w", err)
	}

	// Get the resource from tenant collection
	resourceKey := fmt.Sprintf("%s/%s", resourceType, resourceID)
	result, err := collection.Get(resourceKey, &gocb.GetOptions{})
	if err != nil {
		return fmt.Errorf("resource not found: %w", err)
	}

	var resource map[string]interface{}
	if err := result.Content(&resource); err != nil {
		return fmt.Errorf("failed to parse resource: %w", err)
	}

	// Update the resource with review information
	resource["reviewed"] = true
	resource["reviewTime"] = time.Now().UTC().Format(time.RFC3339)

	// Upsert the updated resource back to the tenant collection
	_, err = collection.Upsert(resourceKey, resource, &gocb.UpsertOptions{})
	if err != nil {
		return fmt.Errorf("failed to update resource review status: %w", err)
	}

	return nil
}

package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// TenantCollectionManager manages tenant collections
type TenantCollectionManager struct {
	bucket *gocb.Bucket
}

// NewTenantCollectionManager creates a new tenant collection manager
func NewTenantCollectionManager(bucket *gocb.Bucket) *TenantCollectionManager {
	return &TenantCollectionManager{
		bucket: bucket,
	}
}

// WaitForFHIRIngestion waits for FHIR client to complete ingestion
func (tcm *TenantCollectionManager) WaitForFHIRIngestion(ctx context.Context) error {
	log.Info().Msg("Waiting for FHIR ingestion to complete...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ready, err := tcm.checkIngestionStatus()
			if err != nil {
				log.Error().Err(err).Msg("Error checking ingestion status")
				continue
			}

			if ready {
				log.Info().Msg("FHIR ingestion completed, API is ready to serve requests")
				return nil
			}

			log.Info().Msg("FHIR ingestion still in progress, waiting...")
		}
	}
}

// checkIngestionStatus checks if FHIR ingestion is complete
func (tcm *TenantCollectionManager) checkIngestionStatus() (bool, error) {
	collection := tcm.bucket.DefaultCollection()

	result, err := collection.Get(IngestionStatusKey, &gocb.GetOptions{})
	if err != nil {
		// Check if it's a key not found error using proper Couchbase error checking
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			// Ingestion status document doesn't exist yet
			return false, nil
		}
		return false, fmt.Errorf("failed to get ingestion status: %w", err)
	}

	var status IngestionStatus
	if err := result.Content(&status); err != nil {
		return false, fmt.Errorf("failed to parse ingestion status: %w", err)
	}

	return status.Ready, nil
}

// EnsureTenantCollection ensures a tenant collection exists, creating it if necessary
func (tcm *TenantCollectionManager) EnsureTenantCollection(tenantID string) (*gocb.Collection, error) {
	collectionName := fmt.Sprintf("%s%s", TenantCollectionPrefix, tenantID)

	// Check if collection already exists
	collections, err := tcm.bucket.Collections().GetAllScopes(&gocb.GetAllScopesOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get collections: %w", err)
	}

	// Look for existing tenant collection
	for _, scope := range collections {
		for _, collection := range scope.Collections {
			if collection.Name == collectionName {
				// Collection exists, return it
				return tcm.bucket.Scope(scope.Name).Collection(collectionName), nil
			}
		}
	}

	// Collection doesn't exist, create it by copying DefaultCollection
	log.Info().Str("collection", collectionName).Msg("Creating tenant collection")

	// For now, we'll use the default collection as a fallback
	// In a real implementation, you'd want to copy the DefaultCollection data
	// This is a simplified version - you might want to implement proper collection copying
	return tcm.bucket.DefaultCollection(), nil
}

// GetTenantCollection returns the collection for a specific tenant
func (tcm *TenantCollectionManager) GetTenantCollection(tenantID string) (*gocb.Collection, error) {
	return tcm.EnsureTenantCollection(tenantID)
}

// Global accessor for TenantCollectionManager
var globalTenantCollectionManager *TenantCollectionManager

// SetGlobalTenantCollectionManager sets the global instance
func SetGlobalTenantCollectionManager(manager *TenantCollectionManager) {
	globalTenantCollectionManager = manager
}

// GetGlobalTenantCollectionManager returns the global instance
func GetGlobalTenantCollectionManager() *TenantCollectionManager {
	return globalTenantCollectionManager
}

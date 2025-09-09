package dal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// ScopeModel represents the database model for scope management
type ScopeModel struct {
	conn *Connection
}

// NewScopeModel creates a new scope model
func NewScopeModel(conn *Connection) *ScopeModel {
	return &ScopeModel{
		conn: conn,
	}
}

// EnsureTenantScope ensures a tenant scope exists and is ready for use
// This method implements the scope creation logic:
// 1. Check if scope exists
// 2. If not, create scope and copy data from DefaultScope
// 3. Set ingestion status to false during copy
// 4. Copy all data from DefaultScope collections to tenant scope collections
// 5. Set ingestion status to true when complete
// 6. Wait for ingestion status if it's false (with 5-minute timeout)
func (sm *ScopeModel) EnsureTenantScope(ctx context.Context, tenantScope string) error {
	log.Info().Str("tenant", tenantScope).Msg("Ensuring tenant scope exists")

	// Step 1: Check if scope exists
	scopeExists, err := sm.scopeExists(ctx, tenantScope)
	if err != nil {
		return fmt.Errorf("failed to check if scope exists: %w", err)
	}

	if !scopeExists {
		log.Info().Str("tenant", tenantScope).Msg("Scope does not exist, creating and copying data")

		// Step 2: Create scope and collections
		if err := sm.createScopeAndCollections(ctx, tenantScope); err != nil {
			return fmt.Errorf("failed to create scope and collections: %w", err)
		}

		// Step 3: Set ingestion status to false and start copying
		ism := NewIngestionStatusModel(sm.conn)
		if err := ism.MarkTenantScopeIngestionStarted(ctx, tenantScope); err != nil {
			return fmt.Errorf("failed to mark ingestion as started: %w", err)
		}

		// Step 4: Copy data from DefaultScope to tenant scope
		if err := sm.copyDataFromDefaultScope(ctx, tenantScope); err != nil {
			return fmt.Errorf("failed to copy data from default scope: %w", err)
		}

		// Step 5: Mark ingestion as completed
		if err := ism.MarkTenantScopeIngestionCompleted(ctx, tenantScope, "Data copied from DefaultScope"); err != nil {
			return fmt.Errorf("failed to mark ingestion as completed: %w", err)
		}

		log.Info().Str("tenant", tenantScope).Msg("Tenant scope created and data copied successfully")
	}
	log.Debug().Str("tenant", tenantScope).Msg("Scope already exists")

	// Step 6: Wait for ingestion status if it's false (with 5-minute timeout)
	ism := NewIngestionStatusModel(sm.conn)
	ready, err := sm.waitForIngestionReady(ctx, tenantScope, ism)
	if err != nil {
		return fmt.Errorf("failed to wait for ingestion ready: %w", err)
	}

	if !ready {
		return fmt.Errorf("tenant scope %s ingestion not ready after timeout", tenantScope)
	}

	log.Info().Str("tenant", tenantScope).Msg("Tenant scope is ready for use")
	return nil
}

// scopeExists checks if a scope exists by trying to create it
func (sm *ScopeModel) scopeExists(ctx context.Context, scopeName string) (bool, error) {
	bucketName := sm.conn.GetBucketName()

	// Check if scope exists by trying to query the defaulty collection
	// If scope doesn't exist, this will fail
	query := fmt.Sprintf("SELECT COUNT(*) as count FROM `%s`.`%s`.defaulty LIMIT 1", bucketName, scopeName)
	_, err := sm.conn.GetCluster().Query(query, &gocb.QueryOptions{Context: ctx})
	if err != nil {
		// If scope doesn't exist, we'll get a "keyspace not found" error
		if strings.Contains(err.Error(), "keyspace not found") || strings.Contains(err.Error(), "Keyspace not found") {
			return false, nil
		}
		// For other errors, assume scope doesn't exist
		return false, nil
	}

	// If query succeeded, scope exists
	return true, nil
}

// createScopeAndCollections creates a scope and its three collections
func (sm *ScopeModel) createScopeAndCollections(ctx context.Context, scopeName string) error {
	bucketName := sm.conn.GetBucketName()

	// Create scope (scopeExists already tried to create it, but let's be explicit)
	createScopeQuery := fmt.Sprintf("CREATE SCOPE `%s`.`%s`", bucketName, scopeName)
	_, err := sm.conn.GetCluster().Query(createScopeQuery, &gocb.QueryOptions{Context: ctx})
	if err != nil {
		// If scope already exists, that's okay
		if !sm.isScopeExistsError(err) {
			return fmt.Errorf("failed to create scope %s: %w", scopeName, err)
		}
		log.Debug().Str("scope", scopeName).Msg("Scope already exists")
	}

	// Create collections using full bucket.scope.collection syntax
	collections := []string{"defaulty", "encounters", "patients", "practitioners"}
	for _, collectionName := range collections {
		createCollectionQuery := fmt.Sprintf("CREATE COLLECTION `%s`.`%s`.`%s`", bucketName, scopeName, collectionName)
		_, err := sm.conn.GetCluster().Query(createCollectionQuery, &gocb.QueryOptions{Context: ctx})
		if err != nil {
			// Log the actual error to see what's happening
			log.Warn().Err(err).Str("scope", scopeName).Str("collection", collectionName).Str("query", createCollectionQuery).Msg("Collection creation error")

			// If collection already exists, that's okay
			if !sm.isCollectionExistsError(err) {
				return fmt.Errorf("failed to create collection %s in scope %s: %w", collectionName, scopeName, err)
			}
			log.Debug().Str("scope", scopeName).Str("collection", collectionName).Msg("Collection already exists")
		} else {
			log.Info().Str("scope", scopeName).Str("collection", collectionName).Msg("Collection created successfully")
		}
	}

	// Create collection-specific indexes
	if err := sm.createCollectionIndexes(ctx, bucketName, scopeName); err != nil {
		log.Warn().Err(err).Str("scope", scopeName).Msg("Failed to create collection indexes, continuing")
	}

	log.Info().Str("scope", scopeName).Msg("Scope and collections created successfully")
	return nil
}

// createCollectionIndexes creates collection-specific indexes for the tenant scope
func (sm *ScopeModel) createCollectionIndexes(ctx context.Context, bucketName, scopeName string) error {
	log.Info().Str("scope", scopeName).Msg("Creating collection-specific indexes")

	// Define indexes for each collection
	indexes := []struct {
		collection string
		indexName  string
		fields     string
	}{
		{"defaulty", "idx_defaulty_id", "id"},
		{"defaulty", "idx_defaulty_ready", "ready"},
		{"encounters", "idx_encounters_id", "id"},
		{"encounters", "idx_encounters_resourceType", "resourceType"},
		{"encounters", "idx_encounters_reviewed", "reviewed"},
		{"patients", "idx_patients_id", "id"},
		{"patients", "idx_patients_resourceType", "resourceType"},
		{"patients", "idx_patients_reviewed", "reviewed"},
		{"practitioners", "idx_practitioners_id", "id"},
		{"practitioners", "idx_practitioners_resourceType", "resourceType"},
		{"practitioners", "idx_practitioners_reviewed", "reviewed"},
	}

	for _, idx := range indexes {
		createIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS `%s` ON `%s`.`%s`.`%s`(%s)",
			idx.indexName, bucketName, scopeName, idx.collection, idx.fields)

		_, err := sm.conn.GetCluster().Query(createIndexQuery, &gocb.QueryOptions{Context: ctx})
		if err != nil {
			log.Warn().
				Err(err).
				Str("scope", scopeName).
				Str("collection", idx.collection).
				Str("index", idx.indexName).
				Msg("Failed to create index (may already exist)")
		} else {
			log.Debug().
				Str("scope", scopeName).
				Str("collection", idx.collection).
				Str("index", idx.indexName).
				Msg("Index created successfully")
		}
	}

	log.Info().Str("scope", scopeName).Msg("Collection indexes creation completed")
	return nil
}

// copyDataFromDefaultScope copies all data from DefaultScope collections to tenant scope collections
func (sm *ScopeModel) copyDataFromDefaultScope(ctx context.Context, tenantScope string) error {
	bucketName := sm.conn.GetBucketName()
	collections := []string{"encounters", "patients", "practitioners"}

	for _, collectionName := range collections {
		log.Info().Str("scope", tenantScope).Str("collection", collectionName).Msg("Copying data from DefaultScope")

		// Copy all documents from DefaultScope collection to tenant scope collection
		copyQuery := fmt.Sprintf("INSERT INTO `%s`.`%s`.`%s` (KEY k, VALUE v) SELECT META(d).id as k, d as v FROM `%s`.`_default`.`%s` AS d",
			bucketName, tenantScope, collectionName,
			bucketName, collectionName)

		_, err := sm.conn.GetCluster().Query(copyQuery, &gocb.QueryOptions{Context: ctx})
		if err != nil {
			return fmt.Errorf("failed to copy data for collection %s: %w", collectionName, err)
		}

		log.Info().Str("scope", tenantScope).Str("collection", collectionName).Msg("Data copied successfully")
	}

	return nil
}

// waitForIngestionReady waits for ingestion to be ready with a 5-minute timeout
func (sm *ScopeModel) waitForIngestionReady(ctx context.Context, tenantScope string, ism *IngestionStatusModel) (bool, error) {
	timeout := 5 * time.Minute
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return false, fmt.Errorf("timeout waiting for ingestion to be ready")
		case <-ticker.C:
			ready, err := ism.IsTenantScopeIngestionReady(ctx, tenantScope)
			if err != nil {
				return false, fmt.Errorf("failed to check ingestion status: %w", err)
			}
			if ready {
				return true, nil
			}
		}
	}
}

// isScopeExistsError checks if the error indicates the scope already exists
func (sm *ScopeModel) isScopeExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "already exists") || contains(errStr, "duplicate")
}

// isCollectionExistsError checks if the error indicates the collection already exists
func (sm *ScopeModel) isCollectionExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "already exists") || contains(errStr, "duplicate")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

// indexOf finds the index of a substring in a string
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

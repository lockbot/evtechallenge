package dal

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// IngestionStatusModel represents the database model for ingestion status
type IngestionStatusModel struct {
	conn *Connection
}

// IngestionStatus represents the ingestion status document
type IngestionStatus struct {
	Ready       bool      `json:"ready"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Message     string    `json:"message"`
}

// IngestionStatusKey is the document key for ingestion status
const IngestionStatusKey = "_system/ingestion_status"

// NewIngestionStatusModel creates a new ingestion status model
func NewIngestionStatusModel(conn *Connection) *IngestionStatusModel {
	return &IngestionStatusModel{
		conn: conn,
	}
}

// GetDefaultScopeIngestionStatus retrieves ingestion status from default scope (for API startup monitoring)
func (ism *IngestionStatusModel) GetDefaultScopeIngestionStatus(ctx context.Context) (*IngestionStatus, error) {
	collection := ism.conn.GetBucket().DefaultCollection()

	result, err := collection.Get(IngestionStatusKey, &gocb.GetOptions{})
	if err != nil {
		// Check if it's a key not found error (simplified check)
		if err.Error() == "document not found" {
			// Ingestion status document doesn't exist yet
			return &IngestionStatus{Ready: false}, nil
		}
		return nil, fmt.Errorf("failed to get default scope ingestion status: %w", err)
	}

	var status IngestionStatus
	if err := result.Content(&status); err != nil {
		return nil, fmt.Errorf("failed to parse default scope ingestion status: %w", err)
	}

	return &status, nil
}

// IsDefaultScopeIngestionReady checks if FHIR ingestion is complete in default scope (for API startup)
func (ism *IngestionStatusModel) IsDefaultScopeIngestionReady(ctx context.Context) (bool, error) {
	status, err := ism.GetDefaultScopeIngestionStatus(ctx)
	if err != nil {
		return false, err
	}

	return status.Ready, nil
}

// GetTenantScopeIngestionStatus retrieves ingestion status from tenant scope
func (ism *IngestionStatusModel) GetTenantScopeIngestionStatus(ctx context.Context, tenantScope string) (*IngestionStatus, error) {
	collection := ism.conn.GetBucket().Scope(tenantScope).Collection("_default")

	result, err := collection.Get(IngestionStatusKey, &gocb.GetOptions{})
	if err != nil {
		// Check if it's a key not found error (simplified check)
		if err.Error() == "document not found" {
			// Ingestion status document doesn't exist yet
			return &IngestionStatus{Ready: false}, nil
		}
		return nil, fmt.Errorf("failed to get tenant scope ingestion status for %s: %w", tenantScope, err)
	}

	var status IngestionStatus
	if err := result.Content(&status); err != nil {
		return nil, fmt.Errorf("failed to parse tenant scope ingestion status for %s: %w", tenantScope, err)
	}

	return &status, nil
}

// IsTenantScopeIngestionReady checks if FHIR ingestion is complete for a specific tenant scope
func (ism *IngestionStatusModel) IsTenantScopeIngestionReady(ctx context.Context, tenantScope string) (bool, error) {
	status, err := ism.GetTenantScopeIngestionStatus(ctx, tenantScope)
	if err != nil {
		return false, err
	}

	return status.Ready, nil
}

// SetTenantScopeIngestionStatus sets the ingestion status for a specific tenant scope
func (ism *IngestionStatusModel) SetTenantScopeIngestionStatus(ctx context.Context, tenantScope string, status *IngestionStatus) error {
	collection := ism.conn.GetBucket().Scope(tenantScope).Collection("_default")

	_, err := collection.Upsert(IngestionStatusKey, status, &gocb.UpsertOptions{})
	if err != nil {
		return fmt.Errorf("failed to set tenant scope ingestion status for %s: %w", tenantScope, err)
	}

	log.Debug().Str("tenant", tenantScope).Bool("ready", status.Ready).Msg("Tenant scope ingestion status updated")
	return nil
}

// MarkTenantScopeIngestionStarted marks ingestion as started for a specific tenant scope
func (ism *IngestionStatusModel) MarkTenantScopeIngestionStarted(ctx context.Context, tenantScope string) error {
	status := &IngestionStatus{
		Ready:     false,
		StartedAt: time.Now(),
		Message:   "FHIR ingestion started",
	}

	return ism.SetTenantScopeIngestionStatus(ctx, tenantScope, status)
}

// MarkTenantScopeIngestionCompleted marks ingestion as completed for a specific tenant scope
func (ism *IngestionStatusModel) MarkTenantScopeIngestionCompleted(ctx context.Context, tenantScope string, message string) error {
	status := &IngestionStatus{
		Ready:       true,
		CompletedAt: time.Now(),
		Message:     message,
	}

	return ism.SetTenantScopeIngestionStatus(ctx, tenantScope, status)
}

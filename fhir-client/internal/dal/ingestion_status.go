package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// IngestionStatus represents the ingestion status document
type IngestionStatus struct {
	Ready       bool      `json:"ready"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Message     string    `json:"message"`
}

// IngestionStatusKey is the document key for ingestion status
const IngestionStatusKey = "template/ingestion_status"

// IngestionStatusModel represents the database model for ingestion status
type IngestionStatusModel struct {
	conn *Connection
}

// NewIngestionStatusModel creates a new ingestion status model
func NewIngestionStatusModel(conn *Connection) *IngestionStatusModel {
	return &IngestionStatusModel{
		conn: conn,
	}
}

// GetIngestionStatus retrieves the current ingestion status
func (ism *IngestionStatusModel) GetIngestionStatus(ctx context.Context) (*IngestionStatus, error) {
	collection := ism.conn.GetBucket().DefaultCollection()

	result, err := collection.Get(IngestionStatusKey, &gocb.GetOptions{})
	if err != nil {
		// Check if document doesn't exist (multiple ways to check this)
		if errors.Is(err, gocb.ErrDocumentNotFound) ||
			strings.Contains(err.Error(), "document not found") ||
			strings.Contains(err.Error(), "Not Found") ||
			strings.Contains(err.Error(), "KEY_ENOENT") {
			// No status document exists yet
			return &IngestionStatus{Ready: false}, nil
		}
		return nil, fmt.Errorf("failed to get ingestion status: %w", err)
	}

	var status IngestionStatus
	if err := result.Content(&status); err != nil {
		return nil, fmt.Errorf("failed to parse ingestion status: %w", err)
	}

	return &status, nil
}

// SetIngestionStatus sets the ingestion status
func (ism *IngestionStatusModel) SetIngestionStatus(ctx context.Context, ready bool, message string) error {
	collection := ism.conn.GetBucket().DefaultCollection()

	status := IngestionStatus{
		Ready:     ready,
		StartedAt: time.Now().UTC(),
		Message:   message,
	}

	if ready {
		status.CompletedAt = time.Now().UTC()
	}

	_, err := collection.Upsert(IngestionStatusKey, status, &gocb.UpsertOptions{})
	if err != nil {
		return fmt.Errorf("failed to set ingestion status: %w", err)
	}

	if ready {
		log.Info().Msg("✅ FHIR ingestion completed successfully")
	} else {
		log.Info().Msg("📝 Ingestion status set to 'not ready'")
	}

	return nil
}

// IsIngestionReady checks if FHIR ingestion is complete
func (ism *IngestionStatusModel) IsIngestionReady(ctx context.Context) (bool, error) {
	status, err := ism.GetIngestionStatus(ctx)
	if err != nil {
		return false, err
	}

	return status.Ready, nil
}

package fhir

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// IngestionStatus represents the FHIR client ingestion status
type IngestionStatus struct {
	Ready       bool      `json:"ready"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Message     string    `json:"message"`
}

const (
	// System document keys
	IngestionStatusKey = "_system/ingestion_status"
)

// CheckAndSetIngestionStatus checks if ingestion is already complete and sets initial status
func (c *Client) CheckAndSetIngestionStatus(ctx context.Context) error {
	log.Info().Msg("Checking ingestion status...")

	// Check if ingestion status document exists
	collection := c.bucket.DefaultCollection()
	result, err := collection.Get(IngestionStatusKey, &gocb.GetOptions{})
	if err != nil {
		if errors.Is(err, gocb.ErrDocumentNotFound) {
			// No status document exists, this is a fresh start
			log.Info().Msg("No ingestion status found, starting fresh ingestion")
			return c.setIngestionStatus(ctx, false, "FHIR ingestion started")
		}
		return fmt.Errorf("failed to check ingestion status: %w", err)
	}

	// Parse existing status
	var status IngestionStatus
	if err := result.Content(&status); err != nil {
		return fmt.Errorf("failed to parse ingestion status: %w", err)
	}

	if status.Ready {
		log.Info().Msg("FHIR ingestion already completed, exiting gracefully")
		return fmt.Errorf("ingestion already completed at %s", status.CompletedAt.Format(time.RFC3339))
	}

	log.Info().Msg("Previous ingestion was incomplete, continuing...")
	return c.setIngestionStatus(ctx, false, "FHIR ingestion resumed")
}

// setIngestionStatus sets the ingestion status in Couchbase
func (c *Client) setIngestionStatus(ctx context.Context, ready bool, message string) error {
	collection := c.bucket.DefaultCollection()

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

// SetIngestionComplete marks the ingestion as complete
func (c *Client) SetIngestionComplete(ctx context.Context) error {
	return c.setIngestionStatus(ctx, true, "FHIR ingestion completed successfully")
}

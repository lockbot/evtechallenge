package fhir

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir-client/internal/dal"
)

// CheckAndSetIngestionStatus checks if ingestion is already complete and sets initial status
func (c *Client) CheckAndSetIngestionStatus(ctx context.Context) error {
	log.Info().Msg("Checking ingestion status...")

	// Create ingestion status model
	ism := dal.NewIngestionStatusModel(c.dal)

	// Check if ingestion status document exists
	status, err := ism.GetIngestionStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to check ingestion status: %w", err)
	}

	if status.Ready {
		log.Info().
			Str("completed_at", status.CompletedAt.Format(time.RFC3339)).
			Msg("FHIR ingestion already completed, exiting gracefully")
		return fmt.Errorf("ingestion already completed at %s", status.CompletedAt.Format(time.RFC3339)) // Success - ingestion is already done!
	}

	// No status document exists or ingestion was incomplete, start fresh
	log.Info().Msg("No ingestion status found or previous ingestion was incomplete, starting fresh ingestion")
	return ism.SetIngestionStatus(ctx, false, "FHIR ingestion started")
}

// SetIngestionComplete marks the ingestion as complete
func (c *Client) SetIngestionComplete(ctx context.Context) error {
	ism := dal.NewIngestionStatusModel(c.dal)
	return ism.SetIngestionStatus(ctx, true, "FHIR ingestion completed successfully")
}

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/dal"
)

// WaitForFHIRIngestion waits for FHIR client to complete ingestion
func WaitForFHIRIngestion(ctx context.Context) error {
	log.Info().Msg("Waiting for FHIR ingestion to complete...")

	conn, err := dal.GetConnOrGenConn()
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}
	defer dal.ReturnConnection(conn) // Return connection to pool

	ingestionModel := dal.NewIngestionStatusModel(conn)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ready, err := ingestionModel.IsDefaultScopeIngestionReady(ctx)
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

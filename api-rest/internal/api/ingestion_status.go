package api

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
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

	bucket := conn.GetBucket()
	collection := bucket.DefaultCollection()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ready, err := checkIngestionStatus(collection)
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
func checkIngestionStatus(collection *gocb.Collection) (bool, error) {
	result, err := collection.Get(IngestionStatusKey, &gocb.GetOptions{})
	if err != nil {
		// Check if it's a key not found error (simplified check)
		if err.Error() == "document not found" {
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

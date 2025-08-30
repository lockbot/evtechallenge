package main

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/evtechallenge/internal/couchbase"
	"stealthcompany.com/evtechallenge/internal/fhir"
	"stealthcompany.com/evtechallenge/internal/zerolog_config"
)

func main() {
	// Initialize zerolog
	zerolog_config.StartupWithEnv("http://elasticsearch:9200", "logs")

	log.Info().Msg("Starting evtechallenge-ingest service")

	// Initialize Couchbase connection
	dbClient, err := couchbase.NewClient(
		os.Getenv("COUCHBASE_URL"),
		os.Getenv("COUCHBASE_USERNAME"),
		os.Getenv("COUCHBASE_PASSWORD"),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Couchbase")
	}
	defer dbClient.Close()

	// Initialize FHIR client
	fhirClient := fhir.NewClient("https://hapi.fhir.org/baseR4", 30*time.Second, dbClient)

	// Create database locker
	locker := dbClient.GetLocker()

	// Lock the database before ingestion
	log.Info().Msg("Locking database for ingestion")
	if err := locker.Lock(); err != nil {
		log.Fatal().Err(err).Msg("Failed to lock database")
	}

	// Ensure unlock happens even if ingestion fails
	defer func() {
		log.Info().Msg("Unlocking database after ingestion")
		if err := locker.Unlock(); err != nil {
			log.Error().Err(err).Msg("Failed to unlock database")
		}
	}()

	// Ingest FHIR data
	if err := fhirClient.IngestAllResources(); err != nil {
		log.Fatal().Err(err).Msg("Failed to ingest FHIR data")
	}

	log.Info().Msg("FHIR data ingestion completed successfully")
}

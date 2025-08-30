package main

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/evtechallenge/internal/metrics"
	"stealthcompany.com/evtechallenge/internal/orchestrator"
	"stealthcompany.com/evtechallenge/internal/zerolog_config"
)

func main() {
	// Initialize zerolog with Elasticsearch (only if enabled)
	zerolog_config.StartupWithEnv(
		os.Getenv("ELASTICSEARCH_URL"),
		os.Getenv("ELASTICSEARCH_INDEX"),
	)

	log.Info().Msg("Starting evtechallenge-orch orchestrator")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start system metrics only if enabled
	metrics.StartSystemMetrics(15 * time.Second)

	// Initialize orchestrator components
	serviceManager := orchestrator.NewServiceManager()
	signalHandler := orchestrator.NewSignalHandler()

	// Handle graceful shutdown signals
	signalHandler.HandleSignals(ctx, cancel)

	// Get binary extension from env
	binExt := os.Getenv("BINARY_EXTENSION")

	// Start ingest service first
	if err := serviceManager.StartIngestService(ctx, binExt); err != nil {
		log.Fatal().Err(err).Msg("Failed to start ingest service")
	}

	// Start API service
	if err := serviceManager.StartAPIService(ctx, binExt); err != nil {
		log.Fatal().Err(err).Msg("Failed to start API service")
	}

	// Wait for services to complete or shutdown
	serviceManager.WaitForServices(ctx)

	log.Info().Msg("Orchestrator shutdown complete")
}

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir-client/internal/fhir"
	"stealthcompany.com/fhir-client/internal/metrics"
	"stealthcompany.com/pkg/zerolog_config"
)

func main() {
	// Load .env file from parent directory
	err := godotenv.Load("../.env")
	if err != nil {
		log.Info().Msg("Not found .env file in parent directory, trying current directory")
		err = godotenv.Load(".env")
		if err != nil {
			log.Info().Msg("Not found .env file in current directory, assuming environment variables are set")
		}
	}

	// Get configuration from environment
	elasticsearchURL := getEnvOrDefault("ELASTICSEARCH_URL", "http://elasticsearch:9200")
	fhirPort := getEnvOrDefault("FHIR_PORT", "8081")

	// Initialize zerolog with Elasticsearch
	zerolog_config.StartupWithEnv(elasticsearchURL, "fhir-")

	log.Info().Msg("Starting evtechallenge-fhir service")

	// Start system metrics collection
	metrics.StartSystemMetricsCollection("fhir-client")

	// Start metrics HTTP server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:    ":" + fhirPort,
			Handler: mux,
		}

		log.Info().
			Str("port", fhirPort).
			Msg("Starting metrics server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().
				Err(err).
				Msg("Metrics server failed")
		}
	}()

	// Create FHIR client
	fhirClient, err := fhir.NewClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create FHIR client")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	// Run FHIR data ingestion
	err = fhirClient.IngestData(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to ingest FHIR data")
	}

	log.Info().Msg("FHIR data ingestion completed successfully")
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

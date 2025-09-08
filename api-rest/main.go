package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/api-rest/internal/api"
	"stealthcompany.com/api-rest/internal/dal"
	"stealthcompany.com/api-rest/internal/metrics"
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
	apiPort := getEnvOrDefault("API_PORT", "8080")
	apiLogLevel := getEnvOrDefault("API_LOG_LEVEL", "info")

	// Set app prefix
	zerolog_config.SetAppPrefix("api-rest")

	// Initialize zerolog with Elasticsearch
	zerolog_config.StartupWithEnv(elasticsearchURL, "logs", apiLogLevel)

	log.Info().Msg("Starting evtechallenge-api service")

	// Start system metrics collection
	metrics.StartSystemMetricsCollection("api-rest")

	// Wait for FHIR ingestion to complete before starting API
	log.Info().Msg("Waiting for FHIR ingestion to complete...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = api.WaitForFHIRIngestion(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to wait for FHIR ingestion")
	}

	// Setup routes
	router := api.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + apiPort,
		Handler: router,
	}

	// Setup graceful shutdown

	// Listen for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Info().
			Str("port", apiPort).
			Msg("Server starting")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().
				Err(err).
				Msg("Failed to start server")
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info().Msg("Received shutdown signal, shutting down gracefully...")

	// Shutdown server with timeout
	shutdownTimeout := 30 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	}

	// Cleanup tenant channels
	log.Info().Msg("Cleaning up tenant channels...")
	api.CleanupAllChannels()

	// Close database connection
	log.Info().Msg("Closing database connection...")
	dalConn, err := dal.GetConnOrGenConn()
	if err == nil {
		dalConn.Close()
		log.Info().Msg("Database connection closed")
	} else {
		log.Warn().Err(err).Msg("Failed to get connection for cleanup")
	}

	log.Info().Msg("API service shutdown complete")
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

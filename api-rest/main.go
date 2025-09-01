package main

import (
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/api/internal/api"            // Update with your actual module name
	"stealthcompany.com/api/internal/metrics"        // Update with your actual module name
	"stealthcompany.com/api/internal/zerolog_config" // Update with your actual module name
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
	elasticsearchIndex := getEnvOrDefault("ELASTICSEARCH_INDEX", "logs")
	apiPort := getEnvOrDefault("API_PORT", "8080")

	// Initialize zerolog with Elasticsearch
	zerolog_config.StartupWithEnv(elasticsearchURL, elasticsearchIndex)

	log.Info().Msg("Starting evtechallenge-api service")

	// Start system metrics collection
	metrics.StartSystemMetricsCollection("api-rest")

	// Setup routes
	router := api.SetupRoutes()

	log.Info().
		Str("port", apiPort).
		Msg("Server starting")

	err = http.ListenAndServe(":"+apiPort, router)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start server")
	}
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

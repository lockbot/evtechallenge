package main

import (
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/evtechallenge/internal/api_rest"
	"stealthcompany.com/evtechallenge/internal/zerolog_config" // Update with your actual module name
)

func main() {
	// Initialize zerolog with Elasticsearch
	zerolog_config.StartupWithEnv("http://elasticsearch:9200", "logs")

	log.Info().Msg("Starting evtechallenge-api service")

	// Setup routes
	router := api_rest.SetupRoutes()

	log.Info().
		Str("port", os.Getenv("API_PORT")).
		Msg("API Server starting")

	if err := http.ListenAndServe(":"+os.Getenv("API_PORT"), router); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start API server")
	}
}

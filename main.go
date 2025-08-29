package main

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/evtechallenge/internal/http_rest"      // Update with your actual module name
	"stealthcompany.com/evtechallenge/internal/zerolog_config" // Update with your actual module name
)

func main() {
	// Initialize zerolog with Elasticsearch
	zerolog_config.StartupWithEnv("http://elasticsearch:9200", "logs")

	log.Info().Msg("Starting evtechallenge-orch service")

	// Setup routes
	router := http_rest.SetupRoutes()

	log.Info().
		Str("port", "8080").
		Msg("Server starting")

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to start server")
	}
}

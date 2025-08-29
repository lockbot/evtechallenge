package main

import (
	"github.com/rs/zerolog/log"
	"stealthcompany.com/evtechallenge/internal/zerolog_config"
)

func main() {
	zerolog_config.StartupWithEnv("http://localhost:9200", "logs")
	log.Info().Msg("Testing logging")
}

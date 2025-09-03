package zerolog_config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.elastic.co/ecszerolog"
)

// ElasticsearchWriter sends logs directly to Elasticsearch
type ElasticsearchWriter struct {
	URL string
}

func (ew ElasticsearchWriter) Write(p []byte) (n int, err error) {
	// Send JSON log to Elasticsearch
	resp, err := http.Post(
		ew.URL+"/_doc",
		"application/json",
		bytes.NewBuffer(p),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("elasticsearch returned %d", resp.StatusCode)
	}

	return len(p), nil
}

// ConsoleLevelWriter for CLI output with pretty formatting
type ConsoleLevelWriter struct {
	Writer io.Writer
}

func (clw ConsoleLevelWriter) Write(p []byte) (n int, err error) {
	return clw.Writer.Write(p)
}

func StartupWithEnv(elasticsearchURL string, subAddress string) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if elasticsearchURL == "" {
		// Fallback to console only
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
		return
	}

	// ECS format for Elasticsearch with semantic endpoint
	ecsLogger := ecszerolog.New(&ElasticsearchWriter{
		URL: elasticsearchURL + "/" + subAddress,
	})

	// Pretty console output
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	// MultiLevelWriter: ECS to Elasticsearch + Pretty to Console
	multi := zerolog.MultiLevelWriter(
		ecsLogger,
		consoleWriter,
	)

	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
}

package zerolog_config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.elastic.co/ecszerolog"
)

var appPrefix string
var setAppPrefixOnce *sync.Once = &sync.Once{}
var startupLoggerOnce *sync.Once = &sync.Once{}

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

func startupLoggerWithEnv(elasticsearchURL string, subAddress string) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if elasticsearchURL == "" {
		// Fallback to console only
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Str("app", appPrefix).
			Timestamp().Logger()
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

	log.Logger = zerolog.New(multi).With().Str("app", appPrefix).
		Timestamp().Logger()
}

// SetAppPrefix sets the app prefix
func SetAppPrefix(subAddress string) {
	setAppPrefixOnce.Do(func() {
		appPrefix = subAddress
	})
}

// StartupWithEnv sets up the logger with the given Elasticsearch URL and subAddress.
// It returns an error if the subAddress is empty.
// Run SetAppPrefix before StartupWithEnv.
func StartupWithEnv(elasticsearchURL string, subAddress string) error {
	if subAddress == "" {
		return fmt.Errorf("subAddress is required")
	}
	startupLoggerOnce.Do(func() {
		startupLoggerWithEnv(elasticsearchURL, subAddress)
	})
	return nil
}

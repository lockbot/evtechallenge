package fhir

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
)

// Client represents the FHIR client for data ingestion
type Client struct {
	httpClient  *http.Client
	couchbase   *gocb.Cluster
	bucket      *gocb.Bucket
	fhirBaseURL string
	timeout     time.Duration
}

// NewClient creates a new FHIR client
func NewClient() (*Client, error) {
	var err error

	// Get configuration from environment
	fhirBaseURL := getEnvOrDefault("FHIR_BASE_URL", "https://hapi.fhir.org/baseR4")
	timeoutStr := getEnvOrDefault("FHIR_TIMEOUT", "30s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 30 * time.Second
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Connect to Couchbase
	couchbaseURL := getEnvOrDefault("COUCHBASE_URL", "couchbase://evtechallenge-db")
	username := getEnvOrDefault("COUCHBASE_USERNAME", "evtechallenge_user")
	password := getEnvOrDefault("COUCHBASE_PASSWORD", "password")

	cluster, err := gocb.Connect(couchbaseURL, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
		TimeoutsConfig: gocb.TimeoutsConfig{
			ConnectTimeout:    60 * time.Second,
			KVTimeout:         5 * time.Second,
			QueryTimeout:      30 * time.Second,
			ManagementTimeout: 30 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Couchbase: %w", err)
	}

	bucket := cluster.Bucket("evtechallenge")

	// Wait for the bucket to be ready for KV and Query operations
	err = bucket.WaitUntilReady(90*time.Second, &gocb.WaitUntilReadyOptions{
		Context:      context.Background(),
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery},
	})
	if err != nil {
		return nil, fmt.Errorf("couchbase bucket not ready: %w", err)
	}

	// Ensure primary index exists for N1QL queries
	_, err = cluster.Query("CREATE PRIMARY INDEX IF NOT EXISTS ON `evtechallenge`", &gocb.QueryOptions{})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to ensure primary index on evtechallenge bucket")
	}

	log.Info().
		Str("fhir_base_url", fhirBaseURL).
		Str("couchbase_url", couchbaseURL).
		Str("bucket", "evtechallenge").
		Msg("FHIR client initialized successfully")

	return &Client{
		httpClient:  httpClient,
		couchbase:   cluster,
		bucket:      bucket,
		fhirBaseURL: fhirBaseURL,
		timeout:     timeout,
	}, nil
}

// Close closes the FHIR client and connections
func (c *Client) Close() error {
	if c.couchbase != nil {
		return c.couchbase.Close(nil)
	}
	return nil
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

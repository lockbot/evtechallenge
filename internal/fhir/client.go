package fhir

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/evtechallenge/internal/couchbase"
	"stealthcompany.com/evtechallenge/internal/metrics"
)

// Client handles FHIR operations including data fetching and ingestion
type Client struct {
	httpClient *http.Client
	baseURL    string
	dbClient   *couchbase.Client
}

// Bundle represents a FHIR bundle response
type Bundle struct {
	ResourceType string `json:"resourceType"`
	Type         string `json:"type"`
	Entry        []struct {
		Resource json.RawMessage `json:"resource"`
	} `json:"entry"`
}

// EndpointConfig defines a FHIR endpoint to ingest
type EndpointConfig struct {
	Name         string
	ResourceType string
	Count        int
	Collection   string
}

// NewClient creates a new FHIR client
func NewClient(baseURL string, timeout time.Duration, dbClient *couchbase.Client) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:  baseURL,
		dbClient: dbClient,
	}
}

// IngestAllResources ingests all configured FHIR resources
func (fc *Client) IngestAllResources() error {
	endpoints := []EndpointConfig{
		{Name: "Practitioners", ResourceType: "Practitioner", Count: 502, Collection: "practitioners"},
		{Name: "Patients", ResourceType: "Patient", Count: 500, Collection: "patients"},
		{Name: "Encounters", ResourceType: "Encounter", Count: 500, Collection: "encounters"},
	}

	for _, endpoint := range endpoints {
		log.Info().Str("endpoint", endpoint.Name).Msg("Starting ingestion")

		if err := fc.ingestEndpoint(endpoint); err != nil {
			return fmt.Errorf("failed to ingest %s: %w", endpoint.Name, err)
		}

		log.Info().Str("endpoint", endpoint.Name).Msg("Completed ingestion")
	}

	return nil
}

// ingestEndpoint ingests a single FHIR endpoint
func (fc *Client) ingestEndpoint(endpoint EndpointConfig) error {
	startTime := time.Now()

	// Fetch FHIR data
	bundle, err := fc.fetchResource(endpoint.ResourceType, endpoint.Count)
	if err != nil {
		metrics.RecordIngestionMetrics(endpoint.Name, endpoint.Collection, startTime, "failed", 0, 0, 0)
		return fmt.Errorf("failed to fetch %s: %w", endpoint.Name, err)
	}

	resourceCount := len(bundle.Entry)
	log.Info().
		Str("endpoint", endpoint.Name).
		Int("count", resourceCount).
		Msg("Parsed FHIR bundle")

	// Store each resource in Couchbase
	storedCount := 0
	failedCount := 0

	for i, entry := range bundle.Entry {
		docID := fmt.Sprintf("%s_%d", endpoint.Collection, i+1)

		if err := fc.dbClient.UpsertDocument(endpoint.Collection, docID, entry.Resource); err != nil {
			log.Error().
				Err(err).
				Str("endpoint", endpoint.Name).
				Str("docID", docID).
				Msg("Failed to store document")
			failedCount++
			continue
		}

		storedCount++

		if (i+1)%100 == 0 {
			log.Info().
				Str("endpoint", endpoint.Name).
				Int("processed", i+1).
				Int("total", resourceCount).
				Msg("Progress update")
		}
	}

	// Record metrics
	metrics.RecordIngestionMetrics(endpoint.Name, endpoint.Collection, startTime, "success", resourceCount, storedCount, failedCount)

	log.Info().
		Str("endpoint", endpoint.Name).
		Int("total", resourceCount).
		Int("stored", storedCount).
		Int("failed", failedCount).
		Msg("Completed ingestion")

	return nil
}

// fetchResource fetches a specific FHIR resource type with count limit
func (fc *Client) fetchResource(resourceType string, count int) (*Bundle, error) {
	startTime := time.Now()
	url := fmt.Sprintf("%s/%s?_count=%d", fc.baseURL, resourceType, count)

	resp, err := fc.httpClient.Get(url)
	if err != nil {
		metrics.RecordHTTPMetrics(resourceType, startTime, 0)
		return nil, fmt.Errorf("failed to fetch %s: %w", resourceType, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("Failed to close response body")
		}
	}()

	// Record HTTP metrics
	metrics.RecordHTTPMetrics(resourceType, startTime, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR server returned status %d for %s", resp.StatusCode, resourceType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for %s: %w", resourceType, err)
	}

	var bundle Bundle
	if err := json.Unmarshal(body, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse FHIR bundle for %s: %w", resourceType, err)
	}

	return &bundle, nil
}

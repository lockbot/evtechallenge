package fhir

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir/internal/metrics"
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
	if err := bucket.WaitUntilReady(90*time.Second, &gocb.WaitUntilReadyOptions{
		Context:      context.Background(),
		ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery},
	}); err != nil {
		return nil, fmt.Errorf("couchbase bucket not ready: %w", err)
	}

	// Ensure primary index exists for N1QL queries
	if _, err := cluster.Query("CREATE PRIMARY INDEX IF NOT EXISTS ON `evtechallenge`", &gocb.QueryOptions{}); err != nil {
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

// IngestData performs the complete FHIR data ingestion process
func (c *Client) IngestData(ctx context.Context) error {
	log.Info().Msg("Starting FHIR data ingestion process")

	// Step 1: Check if database is empty and sync existing data
	if err := c.syncExistingData(ctx); err != nil {
		return fmt.Errorf("failed to sync existing data: %w", err)
	}

	// Step 2: Fetch and ingest new encounters
	if err := c.ingestEncounters(ctx); err != nil {
		return fmt.Errorf("failed to ingest encounters: %w", err)
	}

	// Step 3: Fetch and ingest new practitioners
	if err := c.ingestPractitioners(ctx); err != nil {
		return fmt.Errorf("failed to ingest practitioners: %w", err)
	}

	// Step 4: Fetch and ingest new patients
	if err := c.ingestPatients(ctx); err != nil {
		return fmt.Errorf("failed to ingest patients: %w", err)
	}

	log.Info().Msg("FHIR data ingestion completed successfully")
	return nil
}

// syncExistingData checks existing data and syncs with FHIR API
func (c *Client) syncExistingData(ctx context.Context) error {
	log.Info().Msg("Checking existing data and syncing with FHIR API")

	// Check if encounters collection is empty
	query := "SELECT COUNT(*) as count FROM `evtechallenge` WHERE `resourceType` = 'Encounter'"
	rows, err := c.couchbase.Query(query, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check existing encounters, proceeding with ingestion")
		return nil
	}
	defer rows.Close()

	var result struct {
		Count int64 `json:"count"`
	}
	if rows.Next() {
		if err := rows.Row(&result); err != nil {
			log.Warn().Err(err).Msg("Failed to read encounter count")
		}
	}

	if result.Count == 0 {
		log.Info().Msg("No existing encounters found, skipping sync")
		return nil
	}

	log.Info().Int64("existing_encounters", result.Count).Msg("Found existing encounters, syncing data")

	// Sync existing encounters
	if err := c.syncExistingEncounters(ctx); err != nil {
		return fmt.Errorf("failed to sync existing encounters: %w", err)
	}

	return nil
}

// syncExistingEncounters syncs existing encounters with FHIR API
func (c *Client) syncExistingEncounters(ctx context.Context) error {
	query := "SELECT META(d).id AS id, d AS resource FROM `evtechallenge` AS d WHERE d.`resourceType` = 'Encounter'"
	rows, err := c.couchbase.Query(query, nil)
	if err != nil {
		return fmt.Errorf("failed to query existing encounters: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var row struct {
			ID       string                 `json:"id"`
			Resource map[string]interface{} `json:"resource"`
		}
		if err := rows.Row(&row); err != nil {
			log.Warn().Err(err).Msg("Failed to read encounter row")
			continue
		}

		if err := c.syncEncounter(ctx, row.ID, row.Resource); err != nil {
			log.Warn().Err(err).Str("encounter_id", row.ID).Msg("Failed to sync encounter")
			continue
		}

		count++
		if count%100 == 0 {
			log.Info().Int("processed", count).Msg("Synced encounters progress")
		}
	}

	log.Info().Int("total_synced", count).Msg("Completed syncing existing encounters")
	return nil
}

// syncEncounter syncs a single encounter with FHIR API
func (c *Client) syncEncounter(ctx context.Context, id string, resource map[string]interface{}) error {
	// Extract patient and practitioner references
	patientRefs := c.extractPatientReferences(resource)
	practitionerRefs := c.extractPractitionerReferences(resource)

	// Sync patient references
	for _, patientRef := range patientRefs {
		if err := c.syncPatient(ctx, patientRef); err != nil {
			log.Warn().Err(err).Str("patient_ref", patientRef).Msg("Failed to sync patient")
		}
	}

	// Sync practitioner references
	for _, practitionerRef := range practitionerRefs {
		if err := c.syncPractitioner(ctx, practitionerRef); err != nil {
			log.Warn().Err(err).Str("practitioner_ref", practitionerRef).Msg("Failed to sync practitioner")
		}
	}

	return nil
}

// ingestEncounters fetches and ingests new encounters from FHIR API
func (c *Client) ingestEncounters(ctx context.Context) error {
	log.Info().Msg("Fetching encounters from FHIR API")

	url := fmt.Sprintf("%s/Encounter?_count=500", c.fhirBaseURL)
	encounters, err := c.fetchFHIRBundle(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch encounters: %w", err)
	}

	log.Info().Int("total_encounters", len(encounters)).Msg("Fetched encounters from FHIR API")

	var ingested, skipped int
	for _, encounter := range encounters {
		if err := c.ingestEncounter(ctx, encounter); err != nil {
			log.Warn().Err(err).Str("encounter_id", encounter.ID).Msg("Failed to ingest encounter")
			skipped++
			continue
		}
		ingested++
	}

	log.Info().
		Int("ingested", ingested).
		Int("skipped", skipped).
		Msg("Completed ingesting encounters")

	metrics.RecordFHIRIngestion("encounters", ingested, skipped)
	return nil
}

// ingestPractitioners fetches and ingests new practitioners from FHIR API
func (c *Client) ingestPractitioners(ctx context.Context) error {
	log.Info().Msg("Fetching practitioners from FHIR API")

	url := fmt.Sprintf("%s/Practitioner?_count=500", c.fhirBaseURL)
	practitioners, err := c.fetchFHIRBundle(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch practitioners: %w", err)
	}

	log.Info().Int("total_practitioners", len(practitioners)).Msg("Fetched practitioners from FHIR API")

	var ingested, skipped int
	for _, practitioner := range practitioners {
		if err := c.ingestPractitioner(ctx, practitioner); err != nil {
			log.Warn().Err(err).Str("practitioner_id", practitioner.ID).Msg("Failed to ingest practitioner")
			skipped++
			continue
		}
		ingested++
	}

	log.Info().
		Int("ingested", ingested).
		Int("skipped", skipped).
		Msg("Completed ingesting practitioners")

	metrics.RecordFHIRIngestion("practitioners", ingested, skipped)
	return nil
}

// ingestPatients fetches and ingests new patients from FHIR API
func (c *Client) ingestPatients(ctx context.Context) error {
	log.Info().Msg("Fetching patients from FHIR API")

	url := fmt.Sprintf("%s/Patient?_count=500", c.fhirBaseURL)
	patients, err := c.fetchFHIRBundle(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch patients: %w", err)
	}

	log.Info().Int("total_patients", len(patients)).Msg("Fetched patients from FHIR API")

	var ingested, skipped int
	for _, patient := range patients {
		if err := c.ingestPatient(ctx, patient); err != nil {
			log.Warn().Err(err).Str("patient_id", patient.ID).Msg("Failed to ingest patient")
			skipped++
			continue
		}
		ingested++
	}

	log.Info().
		Int("ingested", ingested).
		Int("skipped", skipped).
		Msg("Completed ingesting patients")

	metrics.RecordFHIRIngestion("patients", ingested, skipped)
	return nil
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

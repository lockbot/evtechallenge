package fhir

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir-client/internal/dal"
)

// Client represents the FHIR client for data ingestion
type Client struct {
	httpClient        *http.Client
	dal               *dal.Connection
	encounterModel    *dal.EncounterModel
	patientModel      *dal.PatientModel
	practitionerModel *dal.PractitionerModel
	fhirBaseURL       string
	timeout           time.Duration
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

	// Connect to Couchbase via DAL
	dalConn, err := dal.GetConnOrGenConn()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DAL connection: %w", err)
	}

	// Initialize models
	resourceModel := dal.NewResourceModel(dalConn)
	encounterModel := dal.NewEncounterModel(resourceModel)
	patientModel := dal.NewPatientModel(resourceModel)
	practitionerModel := dal.NewPractitionerModel(resourceModel)

	log.Info().
		Str("fhir_base_url", fhirBaseURL).
		Msg("FHIR client initialized successfully")

	return &Client{
		httpClient:        httpClient,
		dal:               dalConn,
		encounterModel:    encounterModel,
		patientModel:      patientModel,
		practitionerModel: practitionerModel,
		fhirBaseURL:       fhirBaseURL,
		timeout:           timeout,
	}, nil
}

// Close closes the FHIR client and returns connection to pool
func (c *Client) Close() error {
	if c.dal != nil {
		dal.ReturnConnection(c.dal)
		c.dal = nil
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

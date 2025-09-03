package fhir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir-client/internal/metrics"
)

// syncExistingData checks existing data and syncs with FHIR API
func (c *Client) syncExistingData(ctx context.Context) error {
	var err error

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
		err = rows.Row(&result)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to read encounter count")
		}
	}

	if result.Count == 0 {
		log.Info().Msg("No existing encounters found, skipping sync")
		return nil
	}

	log.Info().Int64("existing_encounters", result.Count).Msg("Found existing encounters, syncing data")

	// Sync existing encounters
	err = c.syncExistingEncounters(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync existing encounters: %w", err)
	}

	return nil
}

// syncExistingEncounters syncs existing encounters with FHIR API
func (c *Client) syncExistingEncounters(ctx context.Context) error {
	var err error

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
		err = rows.Row(&row)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to read encounter row")
			continue
		}

		err = c.syncEncounter(ctx, row.ID, row.Resource)
		if err != nil {
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
	var err error

	// Extract patient and practitioner references
	patientRefs := c.extractPatientReferences(resource)
	practitionerRefs := c.extractPractitionerReferences(resource)

	// Sync patient references
	for _, patientRef := range patientRefs {
		err = c.syncPatient(ctx, patientRef)
		if err != nil {
			log.Warn().Err(err).Str("patient_ref", patientRef).Msg("Failed to sync patient")
		}
	}

	// Sync practitioner references
	for _, practitionerRef := range practitionerRefs {
		err = c.syncPractitioner(ctx, practitionerRef)
		if err != nil {
			log.Warn().Err(err).Str("practitioner_ref", practitionerRef).Msg("Failed to sync practitioner")
		}
	}

	return nil
}

// syncPatient syncs a patient reference with FHIR API
func (c *Client) syncPatient(ctx context.Context, patientRef string) error {
	var err error

	// Check if patient already exists in Couchbase
	docID := fmt.Sprintf("Patient/%s", patientRef)

	// Try to get existing patient
	start := time.Now()
	_, err = c.bucket.DefaultCollection().Get(docID, nil)
	duration := time.Since(start)

	if err == nil {
		// Patient already exists, no need to sync
		metrics.RecordCouchbaseOperation("get", "success")
		metrics.RecordCouchbaseOperationDuration("get", duration)
		return nil
	}

	metrics.RecordCouchbaseOperation("get", "miss")
	metrics.RecordCouchbaseOperationDuration("get", duration)

	// Patient doesn't exist, fetch from FHIR API
	url := fmt.Sprintf("%s/Patient/%s", c.fhirBaseURL, patientRef)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create patient request: %w", err)
	}

	fetchStart := time.Now()
	resp, err := c.httpClient.Do(req)
	fetchDuration := time.Since(fetchStart)

	if err != nil {
		metrics.RecordFHIRAPICall("Patient", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Patient", "individual", fetchDuration)
		return fmt.Errorf("failed to fetch patient: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metrics.RecordFHIRAPICall("Patient", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Patient", "individual", fetchDuration)
		return fmt.Errorf("FHIR API returned status %d for patient", resp.StatusCode)
	}

	metrics.RecordFHIRAPICall("Patient", "success")
	metrics.RecordHTTPFetch("resource_fetch", "success")
	metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
	metrics.RecordFHIRAPICallDuration("Patient", "individual", fetchDuration)

	var patientData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&patientData)
	if err != nil {
		return fmt.Errorf("failed to decode patient data: %w", err)
	}

	// Upsert the patient (denormalize fields)
	patientData["docId"] = docID
	patientData["resourceType"] = "Patient"

	start = time.Now()
	_, err = c.bucket.DefaultCollection().Upsert(docID, patientData, nil)
	duration = time.Since(start)

	if err != nil {
		metrics.RecordCouchbaseOperation("upsert", "error")
		metrics.RecordCouchbaseOperationDuration("upsert", duration)
		return fmt.Errorf("failed to upsert patient: %w", err)
	}

	metrics.RecordCouchbaseOperation("upsert", "success")
	metrics.RecordCouchbaseOperationDuration("upsert", duration)

	return nil
}

// syncPractitioner syncs a practitioner reference with FHIR API
func (c *Client) syncPractitioner(ctx context.Context, practitionerRef string) error {
	var err error

	// Check if practitioner already exists in Couchbase
	docID := fmt.Sprintf("Practitioner/%s", practitionerRef)

	// Try to get existing practitioner
	start := time.Now()
	_, err = c.bucket.DefaultCollection().Get(docID, nil)
	duration := time.Since(start)

	if err == nil {
		// Practitioner already exists, no need to sync
		metrics.RecordCouchbaseOperation("get", "success")
		metrics.RecordCouchbaseOperationDuration("get", duration)
		return nil
	}

	metrics.RecordCouchbaseOperation("get", "miss")
	metrics.RecordCouchbaseOperationDuration("get", duration)

	// Practitioner doesn't exist, fetch from FHIR API
	url := fmt.Sprintf("%s/Practitioner/%s", c.fhirBaseURL, practitionerRef)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create practitioner request: %w", err)
	}

	fetchStart := time.Now()
	resp, err := c.httpClient.Do(req)
	fetchDuration := time.Since(fetchStart)

	if err != nil {
		metrics.RecordFHIRAPICall("Practitioner", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Practitioner", "individual", fetchDuration)
		return fmt.Errorf("failed to fetch practitioner: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metrics.RecordFHIRAPICall("Practitioner", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Practitioner", "individual", fetchDuration)
		return fmt.Errorf("FHIR API returned status %d for practitioner", resp.StatusCode)
	}

	metrics.RecordFHIRAPICall("Practitioner", "success")
	metrics.RecordHTTPFetch("resource_fetch", "success")
	metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
	metrics.RecordFHIRAPICallDuration("Practitioner", "individual", fetchDuration)

	var practitionerData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&practitionerData)
	if err != nil {
		return fmt.Errorf("failed to decode practitioner data: %w", err)
	}

	// Upsert the practitioner (denormalize fields)
	practitionerData["docId"] = docID
	practitionerData["resourceType"] = "Practitioner"

	start = time.Now()
	_, err = c.bucket.DefaultCollection().Upsert(docID, practitionerData, nil)
	duration = time.Since(start)

	if err != nil {
		metrics.RecordCouchbaseOperation("upsert", "error")
		metrics.RecordCouchbaseOperationDuration("upsert", duration)
		return fmt.Errorf("failed to upsert practitioner: %w", err)
	}

	metrics.RecordCouchbaseOperation("upsert", "success")
	metrics.RecordCouchbaseOperationDuration("upsert", duration)

	return nil
}

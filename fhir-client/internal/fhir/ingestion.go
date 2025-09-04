package fhir

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"stealthcompany.com/fhir-client/internal/metrics"
)

// IngestData performs the complete FHIR data ingestion process
func (c *Client) IngestData(ctx context.Context) error {
	var err error

	log.Info().Msg("Starting FHIR data ingestion process")

	// Step 1: Check if database is empty and sync existing data
	err = c.syncExistingData(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync existing data: %w", err)
	}

	// Step 2: Fetch and ingest new encounters
	err = c.ingestEncounters(ctx)
	if err != nil {
		return fmt.Errorf("failed to ingest encounters: %w", err)
	}

	// Step 3: Fetch and ingest new practitioners
	err = c.ingestPractitioners(ctx)
	if err != nil {
		return fmt.Errorf("failed to ingest practitioners: %w", err)
	}

	// Step 4: Fetch and ingest new patients
	err = c.ingestPatients(ctx)
	if err != nil {
		return fmt.Errorf("failed to ingest patients: %w", err)
	}

	log.Info().Msg("FHIR data ingestion completed successfully")
	return nil
}

// ingestEncounters fetches and ingests new encounters from FHIR API
func (c *Client) ingestEncounters(ctx context.Context) error {
	var err error

	log.Info().Msg("Fetching encounters from FHIR API")

	url := fmt.Sprintf("%s/Encounter?_count=500", c.fhirBaseURL)
	encounters, err := c.fetchFHIRBundle(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch encounters: %w", err)
	}

	log.Info().Int("total_encounters", len(encounters)).Msg("Fetched encounters from FHIR API")

	var ingested, skipped int
	for _, encounter := range encounters {
		err = c.ingestEncounter(ctx, encounter)
		if err != nil {
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
	var err error

	log.Info().Msg("Fetching practitioners from FHIR API")

	url := fmt.Sprintf("%s/Practitioner?_count=500", c.fhirBaseURL)
	practitioners, err := c.fetchFHIRBundle(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch practitioners: %w", err)
	}

	log.Info().Int("total_practitioners", len(practitioners)).Msg("Fetched practitioners from FHIR API")

	var ingested, skipped int
	for _, practitioner := range practitioners {
		err = c.ingestPractitioner(ctx, practitioner)
		if err != nil {
			log.Debug().Err(err).Str("practitioner_id", practitioner.ID).Msg("Failed to ingest practitioner")
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
	var err error

	log.Info().Msg("Fetching patients from FHIR API")

	url := fmt.Sprintf("%s/Patient?_count=500", c.fhirBaseURL)
	patients, err := c.fetchFHIRBundle(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to fetch patients: %w", err)
	}

	log.Info().Int("total_patients", len(patients)).Msg("Fetched patients from FHIR API")

	var ingested, skipped int
	for _, patient := range patients {
		err = c.ingestPatient(ctx, patient)
		if err != nil {
			log.Debug().Err(err).Str("patient_id", patient.ID).Msg("Failed to ingest patient")
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

// ingestEncounter ingests a single encounter resource
func (c *Client) ingestEncounter(ctx context.Context, resource FHIRResource) error {
	err := c.encounterModel.UpsertEncounter(ctx, resource.ID, resource.Data)
	if err != nil {
		return fmt.Errorf("failed to upsert encounter: %w", err)
	}

	// Extract and sync related resources
	patientRefs := c.extractPatientReferences(resource.Data)
	practitionerRefs := c.extractPractitionerReferences(resource.Data)

	// Sync patient references
	for _, patientRef := range patientRefs {
		err = c.syncPatient(ctx, patientRef)
		if err != nil {
			log.Debug().Err(err).Str("patient_ref", patientRef).Msg("Failed to sync patient")
		}
	}

	// Sync practitioner references
	for _, practitionerRef := range practitionerRefs {
		err = c.syncPractitioner(ctx, practitionerRef)
		if err != nil {
			log.Debug().Err(err).Str("practitioner_ref", practitionerRef).Msg("Failed to sync practitioner")
		}
	}

	return nil
}

// ingestPractitioner ingests a single practitioner resource
func (c *Client) ingestPractitioner(ctx context.Context, resource FHIRResource) error {
	err := c.practitionerModel.UpsertPractitioner(ctx, resource.ID, resource.Data)
	if err != nil {
		return fmt.Errorf("failed to upsert practitioner: %w", err)
	}
	return nil
}

// ingestPatient ingests a single patient resource
func (c *Client) ingestPatient(ctx context.Context, resource FHIRResource) error {
	err := c.patientModel.UpsertPatient(ctx, resource.ID, resource.Data)
	if err != nil {
		return fmt.Errorf("failed to upsert patient: %w", err)
	}
	return nil
}

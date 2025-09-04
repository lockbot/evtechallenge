package fhir

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// syncExistingData checks existing data and syncs with FHIR API
func (c *Client) syncExistingData(ctx context.Context) error {
	log.Info().Msg("Checking existing data and syncing with FHIR API")

	// Check if encounters collection is empty
	count, err := c.encounterModel.CountEncounters(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check existing encounters, proceeding with ingestion")
		return nil
	}

	if count == 0 {
		log.Info().Msg("No existing encounters found, skipping sync")
		return nil
	}

	log.Info().Int64("existing_encounters", count).Msg("Found existing encounters, syncing data")

	// Sync existing encounters
	err = c.syncExistingEncounters(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync existing encounters: %w", err)
	}

	return nil
}

// syncExistingEncounters syncs existing encounters with FHIR API
func (c *Client) syncExistingEncounters(ctx context.Context) error {
	encounters, err := c.encounterModel.GetAllEncounters(ctx)
	if err != nil {
		return fmt.Errorf("failed to query existing encounters: %w", err)
	}

	var count int
	for _, row := range encounters {
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
	// Extract patient and practitioner references
	patientRefs := c.extractPatientReferences(resource)
	practitionerRefs := c.extractPractitionerReferences(resource)

	// Sync patient references
	for _, patientRef := range patientRefs {
		err := c.syncPatient(ctx, patientRef)
		if err != nil {
			log.Debug().Err(err).Str("patient_ref", patientRef).Msg("Failed to sync patient")
		}
	}

	// Sync practitioner references
	for _, practitionerRef := range practitionerRefs {
		err := c.syncPractitioner(ctx, practitionerRef)
		if err != nil {
			log.Debug().Err(err).Str("practitioner_ref", practitionerRef).Msg("Failed to sync practitioner")
		}
	}

	return nil
}

// syncPatient syncs a patient reference with FHIR API
func (c *Client) syncPatient(ctx context.Context, patientRef string) error {
	// Check if patient already exists in Couchbase
	exists, err := c.patientModel.PatientExists(ctx, patientRef)
	if err != nil {
		return fmt.Errorf("failed to check patient existence: %w", err)
	}

	if exists {
		log.Debug().Str("patient_id", patientRef).Msg("Patient already exists, skipping sync")
		return nil
	}

	// Patient doesn't exist, fetch from FHIR API
	patientData, err := c.fetchPatientFromAPI(ctx, patientRef)
	if err != nil {
		return fmt.Errorf("failed to fetch patient from API: %w", err)
	}

	// Upsert the patient
	err = c.patientModel.UpsertPatient(ctx, patientRef, patientData)
	if err != nil {
		return fmt.Errorf("failed to upsert patient: %w", err)
	}

	log.Debug().Str("patient_id", patientRef).Msg("Successfully synced patient")
	return nil
}

// syncPractitioner syncs a practitioner reference with FHIR API
func (c *Client) syncPractitioner(ctx context.Context, practitionerRef string) error {
	// Check if practitioner already exists in Couchbase
	exists, err := c.practitionerModel.PractitionerExists(ctx, practitionerRef)
	if err != nil {
		return fmt.Errorf("failed to check practitioner existence: %w", err)
	}

	if exists {
		log.Debug().Str("practitioner_id", practitionerRef).Msg("Practitioner already exists, skipping sync")
		return nil
	}

	// Practitioner doesn't exist, fetch from FHIR API
	practitionerData, err := c.fetchPractitionerFromAPI(ctx, practitionerRef)
	if err != nil {
		return fmt.Errorf("failed to fetch practitioner from API: %w", err)
	}

	// Upsert the practitioner
	err = c.practitionerModel.UpsertPractitioner(ctx, practitionerRef, practitionerData)
	if err != nil {
		return fmt.Errorf("failed to upsert practitioner: %w", err)
	}

	log.Debug().Str("practitioner_id", practitionerRef).Msg("Successfully synced practitioner")
	return nil
}

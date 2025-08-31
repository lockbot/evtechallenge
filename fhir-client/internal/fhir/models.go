package fhir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// FHIRBundle represents a FHIR bundle response
type FHIRBundle struct {
	ResourceType string        `json:"resourceType"`
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Entry        []BundleEntry `json:"entry"`
}

// BundleEntry represents an entry in a FHIR bundle
type BundleEntry struct {
	FullURL  string                 `json:"fullUrl"`
	Resource map[string]interface{} `json:"resource"`
	Search   map[string]interface{} `json:"search,omitempty"`
}

// FHIRResource represents a generic FHIR resource
type FHIRResource struct {
	ID           string                 `json:"id"`
	ResourceType string                 `json:"resourceType"`
	Meta         map[string]interface{} `json:"meta,omitempty"`
	Data         map[string]interface{} `json:"-"`
}

// extractPatientReferences extracts patient references from an encounter resource
func (c *Client) extractPatientReferences(resource map[string]interface{}) []string {
	var refs []string

	// Check subject field
	if subject, ok := resource["subject"].(map[string]interface{}); ok {
		if reference, ok := subject["reference"].(string); ok {
			if ref := c.extractReferenceID(reference, "Patient"); ref != "" {
				refs = append(refs, ref)
			}
		}
	}

	return refs
}

// extractPractitionerReferences extracts practitioner references from an encounter resource
func (c *Client) extractPractitionerReferences(resource map[string]interface{}) []string {
	var refs []string

	// Check participant field
	if participants, ok := resource["participant"].([]interface{}); ok {
		for _, participant := range participants {
			if p, ok := participant.(map[string]interface{}); ok {
				if individual, ok := p["individual"].(map[string]interface{}); ok {
					if reference, ok := individual["reference"].(string); ok {
						if ref := c.extractReferenceID(reference, "Practitioner"); ref != "" {
							refs = append(refs, ref)
						}
					}
				}
			}
		}
	}

	return refs
}

// extractReferenceID extracts the ID from a FHIR reference
func (c *Client) extractReferenceID(reference, resourceType string) string {
	// Handle different reference formats:
	// "Patient/123" -> "123"
	// "urn:uuid:abc-123" -> "abc-123"
	// "Group/456" -> skip if not matching resourceType

	if strings.Contains(reference, "/") {
		parts := strings.Split(reference, "/")
		if len(parts) == 2 {
			refType := parts[0]
			refID := parts[1]

			// Check if this is the resource type we're looking for
			if refType == resourceType {
				return refID
			}
		}
	} else if strings.HasPrefix(reference, "urn:uuid:") {
		// Inline bundle reference (not resolvable via FHIR API): skip external sync
		return ""
	}

	return ""
}

// fetchFHIRBundle fetches a FHIR bundle from the given URL
func (c *Client) fetchFHIRBundle(ctx context.Context, url string) ([]FHIRResource, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch FHIR bundle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR API returned status %d", resp.StatusCode)
	}

	var bundle FHIRBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode FHIR bundle: %w", err)
	}

	var resources []FHIRResource
	for _, entry := range bundle.Entry {
		if entry.Resource != nil {
			resource := FHIRResource{
				Data: entry.Resource,
			}

			// Extract ID and resource type
			if id, ok := entry.Resource["id"].(string); ok {
				resource.ID = id
			}
			if rt, ok := entry.Resource["resourceType"].(string); ok {
				resource.ResourceType = rt
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// ingestEncounter ingests a single encounter resource
func (c *Client) ingestEncounter(ctx context.Context, resource FHIRResource) error {
	// Build canonical doc key and denormalize lookups
	docID := fmt.Sprintf("Encounter/%s", resource.ID)
	patientRefs := c.extractPatientReferences(resource.Data)
	practitionerRefs := c.extractPractitionerReferences(resource.Data)
	resource.Data["docId"] = docID
	resource.Data["resourceType"] = "Encounter"
	if len(patientRefs) > 0 {
		resource.Data["subjectPatientId"] = patientRefs[0]
	}
	if len(practitionerRefs) > 0 {
		resource.Data["practitionerIds"] = practitionerRefs
	}

	// Upsert the encounter
	_, err := c.bucket.DefaultCollection().Upsert(docID, resource.Data, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert encounter: %w", err)
	}

	// Extract and sync related resources
	// use already computed refs

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

// ingestPractitioner ingests a single practitioner resource
func (c *Client) ingestPractitioner(ctx context.Context, resource FHIRResource) error {
	docID := fmt.Sprintf("Practitioner/%s", resource.ID)
	resource.Data["docId"] = docID
	resource.Data["resourceType"] = "Practitioner"

	// Upsert the practitioner
	_, err := c.bucket.DefaultCollection().Upsert(docID, resource.Data, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert practitioner: %w", err)
	}

	return nil
}

// ingestPatient ingests a single patient resource
func (c *Client) ingestPatient(ctx context.Context, resource FHIRResource) error {
	docID := fmt.Sprintf("Patient/%s", resource.ID)
	resource.Data["docId"] = docID
	resource.Data["resourceType"] = "Patient"

	// Upsert the patient
	_, err := c.bucket.DefaultCollection().Upsert(docID, resource.Data, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert patient: %w", err)
	}

	return nil
}

// syncPatient syncs a patient reference with FHIR API
func (c *Client) syncPatient(ctx context.Context, patientRef string) error {
	// Check if patient already exists in Couchbase
	docID := fmt.Sprintf("Patient/%s", patientRef)

	// Try to get existing patient
	_, err := c.bucket.DefaultCollection().Get(docID, nil)
	if err == nil {
		// Patient already exists, no need to sync
		return nil
	}

	// Patient doesn't exist, fetch from FHIR API
	url := fmt.Sprintf("%s/Patient/%s", c.fhirBaseURL, patientRef)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create patient request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch patient: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FHIR API returned status %d for patient", resp.StatusCode)
	}

	var patientData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&patientData); err != nil {
		return fmt.Errorf("failed to decode patient data: %w", err)
	}

	// Upsert the patient (denormalize fields)
	patientData["docId"] = docID
	patientData["resourceType"] = "Patient"
	_, err = c.bucket.DefaultCollection().Upsert(docID, patientData, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert patient: %w", err)
	}

	return nil
}

// syncPractitioner syncs a practitioner reference with FHIR API
func (c *Client) syncPractitioner(ctx context.Context, practitionerRef string) error {
	// Check if practitioner already exists in Couchbase
	docID := fmt.Sprintf("Practitioner/%s", practitionerRef)

	// Try to get existing practitioner
	_, err := c.bucket.DefaultCollection().Get(docID, nil)
	if err == nil {
		// Practitioner already exists, no need to sync
		return nil
	}

	// Practitioner doesn't exist, fetch from FHIR API
	url := fmt.Sprintf("%s/Practitioner/%s", c.fhirBaseURL, practitionerRef)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create practitioner request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch practitioner: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FHIR API returned status %d for practitioner", resp.StatusCode)
	}

	var practitionerData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&practitionerData); err != nil {
		return fmt.Errorf("failed to decode practitioner data: %w", err)
	}

	// Upsert the practitioner (denormalize fields)
	practitionerData["docId"] = docID
	practitionerData["resourceType"] = "Practitioner"
	_, err = c.bucket.DefaultCollection().Upsert(docID, practitionerData, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert practitioner: %w", err)
	}

	return nil
}

package dal

import (
	"context"
	"fmt"
	"strings"
)

// EncounterModel handles encounter-specific database operations
type EncounterModel struct {
	resourceModel *ResourceModel
}

// NewEncounterModel creates a new encounter model
func NewEncounterModel(resourceModel *ResourceModel) *EncounterModel {
	return &EncounterModel{
		resourceModel: resourceModel,
	}
}

// UpsertEncounter upserts an encounter resource
func (em *EncounterModel) UpsertEncounter(ctx context.Context, encounterID string, data map[string]interface{}) error {
	docID := fmt.Sprintf("Encounter/%s", encounterID)

	// Denormalize fields for better querying
	data["docId"] = docID
	data["resourceType"] = "Encounter"

	// Extract and add patient reference
	patientRefs := em.extractPatientReferences(data)
	if len(patientRefs) > 0 {
		data["subjectPatientId"] = patientRefs[0]
	}

	// Extract and add practitioner references
	practitionerRefs := em.extractPractitionerReferences(data)
	if len(practitionerRefs) > 0 {
		data["practitionerIds"] = practitionerRefs
	}

	return em.resourceModel.UpsertResource(ctx, docID, data)
}

// GetEncounter retrieves an encounter by ID
func (em *EncounterModel) GetEncounter(ctx context.Context, encounterID string) (map[string]interface{}, error) {
	docID := fmt.Sprintf("Encounter/%s", encounterID)
	return em.resourceModel.GetResource(ctx, docID)
}

// EncounterExists checks if an encounter exists
func (em *EncounterModel) EncounterExists(ctx context.Context, encounterID string) (bool, error) {
	docID := fmt.Sprintf("Encounter/%s", encounterID)
	return em.resourceModel.ResourceExists(ctx, docID)
}

// CountEncounters counts all encounters
func (em *EncounterModel) CountEncounters(ctx context.Context) (int64, error) {
	return em.resourceModel.CountResourcesByType(ctx, "Encounter")
}

// GetAllEncounters retrieves all encounters
func (em *EncounterModel) GetAllEncounters(ctx context.Context) ([]ResourceRow, error) {
	return em.resourceModel.GetAllResourcesByType(ctx, "Encounter")
}

// extractPatientReferences extracts patient references from an encounter resource
func (em *EncounterModel) extractPatientReferences(resource map[string]interface{}) []string {
	var refs []string

	subject, ok := resource["subject"].(map[string]interface{})
	if !ok {
		return refs
	}

	reference, ok := subject["reference"].(string)
	if !ok {
		return refs
	}

	if ref := em.extractReferenceID(reference, "Patient"); ref != "" {
		refs = append(refs, ref)
	}

	return refs
}

// extractPractitionerReferences extracts practitioner references from an encounter resource
func (em *EncounterModel) extractPractitionerReferences(resource map[string]interface{}) []string {
	var refs []string

	participants, ok := resource["participant"].([]interface{})
	if !ok {
		return refs
	}

	for _, participant := range participants {
		p, ok := participant.(map[string]interface{})
		if !ok {
			continue
		}

		individual, ok := p["individual"].(map[string]interface{})
		if !ok {
			continue
		}

		reference, ok := individual["reference"].(string)
		if !ok {
			continue
		}

		if ref := em.extractReferenceID(reference, "Practitioner"); ref != "" {
			refs = append(refs, ref)
		}
	}

	return refs
}

// extractReferenceID extracts the ID from a FHIR reference
func (em *EncounterModel) extractReferenceID(reference, resourceType string) string {
	if strings.Contains(reference, "/") {
		parts := strings.Split(reference, "/")
		if len(parts) == 2 {
			refType := parts[0]
			refID := parts[1]

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

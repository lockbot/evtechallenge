package fhir

import (
	"strings"
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
	subject, ok := resource["subject"].(map[string]interface{})
	if !ok {
		return refs
	}

	reference, ok := subject["reference"].(string)
	if !ok {
		return refs
	}

	if ref := c.extractReferenceID(reference, "Patient"); ref != "" {
		refs = append(refs, ref)
	}

	return refs
}

// extractPractitionerReferences extracts practitioner references from an encounter resource
func (c *Client) extractPractitionerReferences(resource map[string]interface{}) []string {
	var refs []string

	// Check participant field
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

		if ref := c.extractReferenceID(reference, "Practitioner"); ref != "" {
			refs = append(refs, ref)
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

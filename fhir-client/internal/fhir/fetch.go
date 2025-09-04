package fhir

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"stealthcompany.com/fhir-client/internal/metrics"
)

// fetchFHIRBundle fetches a FHIR bundle from the given URL
func (c *Client) fetchFHIRBundle(ctx context.Context, url string) ([]FHIRResource, error) {
	var err error

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	fetchStart := time.Now()
	resp, err := c.httpClient.Do(req)
	fetchDuration := time.Since(fetchStart)

	if err != nil {
		metrics.RecordHTTPFetch("bundle_fetch", "error")
		metrics.RecordHTTPFetchDuration("bundle_fetch", fetchDuration)
		return nil, fmt.Errorf("failed to fetch FHIR bundle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metrics.RecordHTTPFetch("bundle_fetch", "error")
		metrics.RecordHTTPFetchDuration("bundle_fetch", fetchDuration)
		return nil, fmt.Errorf("FHIR API returned status %d", resp.StatusCode)
	}

	metrics.RecordHTTPFetch("bundle_fetch", "success")
	metrics.RecordHTTPFetchDuration("bundle_fetch", fetchDuration)

	var bundle FHIRBundle
	err = json.NewDecoder(resp.Body).Decode(&bundle)
	if err != nil {
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

// fetchPatientFromAPI fetches a single patient from FHIR API
func (c *Client) fetchPatientFromAPI(ctx context.Context, patientID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/Patient/%s", c.fhirBaseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create patient request: %w", err)
	}

	fetchStart := time.Now()
	resp, err := c.httpClient.Do(req)
	fetchDuration := time.Since(fetchStart)

	if err != nil {
		metrics.RecordFHIRAPICall("Patient", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Patient", "individual", fetchDuration)
		return nil, fmt.Errorf("failed to fetch patient: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metrics.RecordFHIRAPICall("Patient", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Patient", "individual", fetchDuration)
		return nil, fmt.Errorf("FHIR API returned status %d for patient", resp.StatusCode)
	}

	metrics.RecordFHIRAPICall("Patient", "success")
	metrics.RecordHTTPFetch("resource_fetch", "success")
	metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
	metrics.RecordFHIRAPICallDuration("Patient", "individual", fetchDuration)

	var patientData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&patientData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode patient data: %w", err)
	}

	return patientData, nil
}

// fetchPractitionerFromAPI fetches a single practitioner from FHIR API
func (c *Client) fetchPractitionerFromAPI(ctx context.Context, practitionerID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/Practitioner/%s", c.fhirBaseURL, practitionerID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create practitioner request: %w", err)
	}

	fetchStart := time.Now()
	resp, err := c.httpClient.Do(req)
	fetchDuration := time.Since(fetchStart)

	if err != nil {
		metrics.RecordFHIRAPICall("Practitioner", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Practitioner", "individual", fetchDuration)
		return nil, fmt.Errorf("failed to fetch practitioner: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metrics.RecordFHIRAPICall("Practitioner", "error")
		metrics.RecordHTTPFetch("resource_fetch", "error")
		metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
		metrics.RecordFHIRAPICallDuration("Practitioner", "individual", fetchDuration)
		return nil, fmt.Errorf("FHIR API returned status %d for practitioner", resp.StatusCode)
	}

	metrics.RecordFHIRAPICall("Practitioner", "success")
	metrics.RecordHTTPFetch("resource_fetch", "success")
	metrics.RecordHTTPFetchDuration("resource_fetch", fetchDuration)
	metrics.RecordFHIRAPICallDuration("Practitioner", "individual", fetchDuration)

	var practitionerData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&practitionerData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode practitioner data: %w", err)
	}

	return practitionerData, nil
}

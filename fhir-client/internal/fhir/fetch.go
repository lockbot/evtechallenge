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

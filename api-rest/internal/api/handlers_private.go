package api

import (
	"context"
	"fmt"
	"strings"

	"stealthcompany.com/api-rest/internal/dal"
)

// getResourceByID retrieves a single resource by ID (private function for channel processing)
func getResourceByID(ctx context.Context, tenantID, resourceType, id string) (map[string]interface{}, error) {
	// Get connection
	conn, err := dal.GetConnectionWithRetry()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer dal.ReturnConnection(conn)

	// Create resource model
	resourceModel := dal.NewResourceModel(conn)

	// Get the resource
	doc, err := resourceModel.GetResource(ctx, resourceType+"/"+id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resource: %w", err)
	}

	// Review fields are already embedded in the document from fhir-client ingestion
	return map[string]interface{}{
		"data": doc,
	}, nil
}

// listResources retrieves a list of resources (private function for channel processing)
func listResources(ctx context.Context, tenantID, resourceType string, page, count int) (map[string]interface{}, error) {
	// Get connection
	conn, err := dal.GetConnectionWithRetry()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer dal.ReturnConnection(conn)

	// Create resource model
	resourceModel := dal.NewResourceModel(conn)

	// Get resources using the appropriate model
	var paginatedResponse *dal.PaginatedResponse
	var listErr error

	switch resourceType {
	case "Encounter":
		encounterModel := dal.NewEncounterModel(resourceModel)
		paginatedResponse, listErr = encounterModel.List(ctx, page, count)
	case "Patient":
		patientModel := dal.NewPatientModel(resourceModel)
		paginatedResponse, listErr = patientModel.List(ctx, page, count)
	case "Practitioner":
		practitionerModel := dal.NewPractitionerModel(resourceModel)
		paginatedResponse, listErr = practitionerModel.List(ctx, page, count)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	if listErr != nil {
		return nil, fmt.Errorf("failed to list resources: %w", listErr)
	}

	// Review fields are already embedded in the documents from fhir-client ingestion
	// No need to fetch review info separately

	return map[string]interface{}{
		"data":       paginatedResponse.Data,
		"pagination": paginatedResponse.Pagination,
	}, nil
}

// processReviewRequest processes a review request (private function for channel processing)
func processReviewRequest(ctx context.Context, tenantID, resourceType, entityID string) (map[string]interface{}, error) {
	// Get connection
	conn, err := dal.GetConnectionWithRetry()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer dal.ReturnConnection(conn)

	// Create resource model
	resourceModel := dal.NewResourceModel(conn)
	reviewModel := dal.NewReviewModel(resourceModel)

	// entityID is in format "ResourceType/ID", extract just the ID part
	resourceID := entityID
	if strings.Contains(entityID, "/") {
		parts := strings.Split(entityID, "/")
		resourceID = parts[len(parts)-1] // Get the last part (the actual ID)
	}

	err = reviewModel.CreateReviewRequest(ctx, tenantID, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create review request: %w", err)
	}

	// Create response
	response := map[string]string{
		"status":   "review requested",
		"tenant":   tenantID,
		"entity":   entityID,
		"reviewed": "true",
	}

	return map[string]interface{}{
		"status":   response["status"],
		"tenant":   response["tenant"],
		"entity":   response["entity"],
		"reviewed": response["reviewed"],
	}, nil
}

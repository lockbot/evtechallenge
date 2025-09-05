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

	// Check review status for this tenant
	reviewModel := dal.NewReviewModel(resourceModel)
	reviewInfo := reviewModel.GetReviewInfo(ctx, tenantID, resourceType, id)

	response := ResponseWithReview{
		Reviewed:   reviewInfo.Reviewed,
		ReviewTime: reviewInfo.ReviewTime,
		Data:       doc,
	}

	return map[string]interface{}{
		"reviewed":   response.Reviewed,
		"reviewTime": response.ReviewTime,
		"data":       response.Data,
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

	// Add review information to each resource
	reviewModel := dal.NewReviewModel(resourceModel)
	for i := range paginatedResponse.Data {
		// Extract just the ID part from the document ID (remove resource type prefix)
		documentID := paginatedResponse.Data[i].ID
		resourceID := documentID
		if strings.Contains(documentID, "/") {
			parts := strings.Split(documentID, "/")
			resourceID = parts[len(parts)-1] // Get the last part (the actual ID)
		}

		reviewInfo := reviewModel.GetReviewInfo(ctx, tenantID, resourceType, resourceID)
		// Add review info to the Resource field so it appears in the JSON response
		paginatedResponse.Data[i].Resource["reviewed"] = reviewInfo.Reviewed
		if reviewInfo.Reviewed {
			paginatedResponse.Data[i].Resource["reviewTime"] = reviewInfo.ReviewTime
			paginatedResponse.Data[i].Resource["entityType"] = reviewInfo.EntityType
			paginatedResponse.Data[i].Resource["entityID"] = reviewInfo.EntityID
		}
	}

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

package dal

import (
	"context"
	"fmt"
)

// PractitionerModel handles practitioner-specific database operations
type PractitionerModel struct {
	resourceModel *ResourceModel
}

// NewPractitionerModel creates a new practitioner model
func NewPractitionerModel(resourceModel *ResourceModel) *PractitionerModel {
	return &PractitionerModel{
		resourceModel: resourceModel,
	}
}

// UpsertPractitioner upserts a practitioner resource
func (pm *PractitionerModel) UpsertPractitioner(ctx context.Context, practitionerID string, data map[string]interface{}) error {
	docID := fmt.Sprintf("Practitioner/%s", practitionerID)

	// Denormalize fields for better querying
	data["docId"] = docID
	data["resourceType"] = "Practitioner"

	return pm.resourceModel.UpsertResource(ctx, docID, data)
}

// GetPractitioner retrieves a practitioner by ID
func (pm *PractitionerModel) GetPractitioner(ctx context.Context, practitionerID string) (map[string]interface{}, error) {
	docID := fmt.Sprintf("Practitioner/%s", practitionerID)
	return pm.resourceModel.GetResource(ctx, docID)
}

// PractitionerExists checks if a practitioner exists
func (pm *PractitionerModel) PractitionerExists(ctx context.Context, practitionerID string) (bool, error) {
	docID := fmt.Sprintf("Practitioner/%s", practitionerID)
	return pm.resourceModel.ResourceExists(ctx, docID)
}

// CountPractitioners counts all practitioners
func (pm *PractitionerModel) CountPractitioners(ctx context.Context) (int64, error) {
	return pm.resourceModel.CountResourcesByType(ctx, "Practitioner")
}

// GetAllPractitioners retrieves all practitioners
func (pm *PractitionerModel) GetAllPractitioners(ctx context.Context) ([]ResourceRow, error) {
	return pm.resourceModel.GetAllResourcesByType(ctx, "Practitioner")
}

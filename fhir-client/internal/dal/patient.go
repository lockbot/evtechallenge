package dal

import (
	"context"
	"fmt"
)

// PatientModel handles patient-specific database operations
type PatientModel struct {
	resourceModel *ResourceModel
}

// NewPatientModel creates a new patient model
func NewPatientModel(resourceModel *ResourceModel) *PatientModel {
	return &PatientModel{
		resourceModel: resourceModel,
	}
}

// UpsertPatient upserts a patient resource
func (pm *PatientModel) UpsertPatient(ctx context.Context, patientID string, data map[string]interface{}) error {
	docID := fmt.Sprintf("Patient/%s", patientID)

	// Denormalize fields for better querying
	data["docId"] = docID
	data["resourceType"] = "Patient"

	return pm.resourceModel.UpsertResource(ctx, docID, data)
}

// GetPatient retrieves a patient by ID
func (pm *PatientModel) GetPatient(ctx context.Context, patientID string) (map[string]interface{}, error) {
	docID := fmt.Sprintf("Patient/%s", patientID)
	return pm.resourceModel.GetResource(ctx, docID)
}

// PatientExists checks if a patient exists
func (pm *PatientModel) PatientExists(ctx context.Context, patientID string) (bool, error) {
	docID := fmt.Sprintf("Patient/%s", patientID)
	return pm.resourceModel.ResourceExists(ctx, docID)
}

// CountPatients counts all patients
func (pm *PatientModel) CountPatients(ctx context.Context) (int64, error) {
	return pm.resourceModel.CountResourcesByType(ctx, "Patient")
}

// GetAllPatients retrieves all patients
func (pm *PatientModel) GetAllPatients(ctx context.Context) ([]ResourceRow, error) {
	return pm.resourceModel.GetAllResourcesByType(ctx, "Patient")
}

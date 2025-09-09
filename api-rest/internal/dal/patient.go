package dal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

// PatientModel handles patient-specific database operations
type PatientModel struct {
	resourceModel *ResourceModel
}

// NewPatientModel creates a new patient model instance
func NewPatientModel(resourceModel *ResourceModel) *PatientModel {
	return &PatientModel{resourceModel: resourceModel}
}

// NewPatientModelWithTenant creates a new patient model instance for a specific tenant
func NewPatientModelWithTenant(conn *Connection, tenantScope string) *PatientModel {
	resourceModel := NewResourceModelWithTenant(conn, tenantScope)
	return &PatientModel{resourceModel: resourceModel}
}

// GetByID retrieves a patient by ID
func (pm *PatientModel) GetByID(ctx context.Context, id string) (map[string]interface{}, error) {
	log.Debug().
		Str("id", id).
		Msg("Getting patient by ID")

	docID := fmt.Sprintf("Patient/%s", id)
	return pm.resourceModel.GetResource(ctx, docID)
}

// List retrieves a paginated list of patients
func (pm *PatientModel) List(ctx context.Context, page, count int) (*PaginatedResponse, error) {
	log.Debug().
		Int("page", page).
		Int("count", count).
		Msg("Listing patients")

	params := PaginationParams{
		Page:  page,
		Count: count,
	}
	return pm.resourceModel.ListResources(ctx, "Patient", params)
}

// ValidatePaginationParams validates and normalizes pagination parameters
func (pm *PatientModel) ValidatePaginationParams(pageStr, countStr string) (int, int, error) {
	page := 1
	count := 100

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 10000 {
			count = c
		}
	}

	log.Debug().
		Int("page", page).
		Int("count", count).
		Msg("Validated pagination parameters")

	return page, count, nil
}

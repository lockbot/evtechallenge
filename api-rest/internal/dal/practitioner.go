package dal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

// PractitionerModel handles practitioner-specific database operations
type PractitionerModel struct {
	resourceModel *ResourceModel
}

// NewPractitionerModel creates a new practitioner model instance
func NewPractitionerModel(resourceModel *ResourceModel) *PractitionerModel {
	return &PractitionerModel{resourceModel: resourceModel}
}

// GetByID retrieves a practitioner by ID
func (prm *PractitionerModel) GetByID(ctx context.Context, id string) (map[string]interface{}, error) {
	log.Debug().
		Str("id", id).
		Msg("Getting practitioner by ID")

	docID := fmt.Sprintf("Practitioner/%s", id)
	return prm.resourceModel.GetResource(ctx, docID)
}

// List retrieves a paginated list of practitioners
func (prm *PractitionerModel) List(ctx context.Context, page, count int) (*PaginatedResponse, error) {
	log.Debug().
		Int("page", page).
		Int("count", count).
		Msg("Listing practitioners")

	params := PaginationParams{
		Page:  page,
		Count: count,
	}
	return prm.resourceModel.ListResources(ctx, "Practitioner", params)
}

// ValidatePaginationParams validates and normalizes pagination parameters
func (prm *PractitionerModel) ValidatePaginationParams(pageStr, countStr string) (int, int, error) {
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

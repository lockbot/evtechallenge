package dal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

// EncounterModel handles encounter-specific database operations
type EncounterModel struct {
	resourceModel *ResourceModel
}

// NewEncounterModel creates a new encounter model instance
func NewEncounterModel(resourceModel *ResourceModel) *EncounterModel {
	return &EncounterModel{resourceModel: resourceModel}
}

// GetByID retrieves an encounter by ID
func (em *EncounterModel) GetByID(ctx context.Context, id string) (map[string]interface{}, error) {
	log.Debug().
		Str("id", id).
		Msg("Getting encounter by ID")

	docID := fmt.Sprintf("Encounter/%s", id)
	return em.resourceModel.GetResource(ctx, docID)
}

// List retrieves a paginated list of encounters
func (em *EncounterModel) List(ctx context.Context, page, count int) (*PaginatedResponse, error) {
	log.Debug().
		Int("page", page).
		Int("count", count).
		Msg("Listing encounters")

	params := PaginationParams{
		Page:  page,
		Count: count,
	}
	return em.resourceModel.ListResources(ctx, "Encounter", params)
}

// ValidatePaginationParams validates and normalizes pagination parameters
func (em *EncounterModel) ValidatePaginationParams(pageStr, countStr string) (int, int, error) {
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

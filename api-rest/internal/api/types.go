package api

import "time"

// Request Types
type AllGoodRequest struct {
	Yes bool `json:"yes"`
}

type ReviewRequest struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
}

// Response Types
type ResponseWithReview struct {
	Reviewed   bool                   `json:"reviewed"`
	ReviewTime string                 `json:"reviewTime,omitempty"`
	Data       map[string]interface{} `json:"data"`
}

// System Types
type IngestionStatus struct {
	Ready       bool      `json:"ready"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Message     string    `json:"message"`
}

// Constants
const (
	// Tenant Management
	DefaultTenant = "default"

	// System Document Keys
	IngestionStatusKey = "_system/ingestion_status"
)

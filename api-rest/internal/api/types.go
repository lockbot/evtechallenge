package api

import "time"

// AllGoodRequest represents the expected JSON payload
type AllGoodRequest struct {
	Yes bool `json:"yes"`
}

// IngestionStatus represents the FHIR client ingestion status
type IngestionStatus struct {
	Ready       bool      `json:"ready"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Message     string    `json:"message"`
}

// ReviewRequest represents the expected JSON payload for review requests
type ReviewRequest struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
}

// ResponseWithReview wraps API responses with review status
type ResponseWithReview struct {
	Reviewed   bool                   `json:"reviewed"`
	ReviewTime string                 `json:"reviewTime,omitempty"`
	Data       map[string]interface{} `json:"data"`
}

// QueryRow represents a row from N1QL query results
type QueryRow struct {
	ID       string                 `json:"id"`
	Resource map[string]interface{} `json:"resource"`
}

const (
	TenantHeaderKey = "X-Tenant-ID"
	DefaultTenant   = "default"

	// System document keys
	IngestionStatusKey = "_system/ingestion_status"

	// Collection naming
	DefaultCollectionName  = "_default"
	TenantCollectionPrefix = "tenant_"
)

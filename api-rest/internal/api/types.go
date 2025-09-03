package api

import "time"

// AllGoodRequest represents the expected JSON payload
type AllGoodRequest struct {
	Yes bool `json:"yes"`
}

// ReviewRequest represents the expected JSON payload for review requests
type ReviewRequest struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
}

// ReviewDocument represents a tenant's review list
type ReviewDocument struct {
	TenantID      string                 `json:"tenantId"`
	Encounters    map[string]interface{} `json:"encounters"`    // key: "Encounter/ID", value: review metadata
	Patients      map[string]interface{} `json:"patients"`      // key: "Patient/ID", value: review metadata
	Practitioners map[string]interface{} `json:"practitioners"` // key: "Practitioner/ID", value: review metadata
	Updated       time.Time              `json:"updated"`
}

// ReviewInfo contains review status and metadata
type ReviewInfo struct {
	Reviewed   bool   `json:"reviewed"`
	ReviewTime string `json:"reviewTime,omitempty"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   string `json:"entityID,omitempty"`
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
)

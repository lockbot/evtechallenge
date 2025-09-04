package api

// AllGoodRequest represents the expected JSON payload
type AllGoodRequest struct {
	Yes bool `json:"yes"`
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

const (
	TenantHeaderKey = "X-Tenant-ID"
	DefaultTenant   = "default"
)

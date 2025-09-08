package api

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

// Constants
const (
	// Tenant Management
	DefaultTenant = "default"
)

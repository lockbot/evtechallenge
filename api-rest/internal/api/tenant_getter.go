package api

import (
	"net/http"
)

// GetTenantFromRequest extracts tenant ID from request context (set by auth middleware)
func GetTenantFromRequest(r *http.Request) (string, error) {
	return GetTenantFromContext(r.Context())
}

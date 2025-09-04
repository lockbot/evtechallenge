package api

import (
	"fmt"
	"net/http"
	"strings"
)

// GetTenantFromRequest extracts tenant ID from request headers
func GetTenantFromRequest(r *http.Request) (string, error) {
	tenant := r.Header.Get(TenantHeaderKey)
	if tenant == "" {
		return "", fmt.Errorf("missing required header: %s", TenantHeaderKey)
	}
	trimmedTenant := strings.TrimSpace(tenant)
	if trimmedTenant == "" {
		return "", fmt.Errorf("tenant ID cannot be empty")
	}
	return trimmedTenant, nil
}

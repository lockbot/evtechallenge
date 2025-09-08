package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with auth middleware
	authHandler := AuthMiddleware(handler)

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Health endpoint should skip auth",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Metrics endpoint should skip auth",
			path:           "/metrics",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API endpoint without auth should fail",
			path:           "/api/tenant1/patients",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "API endpoint with invalid auth should fail",
			path:           "/api/tenant1/patients",
			authHeader:     "Invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "API endpoint with Bearer but no token should fail",
			path:           "/api/tenant1/patients",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			authHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestExtractTenantFromURL(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid tenant URL",
			path:        "/api/tenant1/patients",
			expected:    "tenant1",
			expectError: false,
		},
		{
			name:        "Valid tenant URL with ID",
			path:        "/api/tenant2/encounters/123",
			expected:    "tenant2",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			path:        "/patients",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty tenant",
			path:        "/api//patients",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTenantFromURL(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

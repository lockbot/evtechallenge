package api

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Context key types to avoid collisions (Go best practice)
type contextKey string

const (
	TenantIDKey   contextKey = "tenantID"
	UserIDKey     contextKey = "userID"
	UsernameKey   contextKey = "username"
	UserGroupsKey contextKey = "userGroups"
)

// HTTP header constants
const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
)

// HTTP path constants
const (
	HealthPath  = "/health"
	MetricsPath = "/metrics"
)

// Error message constants
const (
	ErrAuthHeaderRequired  = "Authorization header required"
	ErrInvalidAuthHeader   = "Invalid authorization header format"
	ErrInvalidToken        = "Invalid token"
	ErrInvalidTenantConfig = "Invalid tenant configuration"
	ErrNoGroupsAssigned    = "user has no groups assigned"

	ErrTenantIDNotFound      = "tenant ID not found in context"
	ErrUserIDNotFound        = "user ID not found in context"
	ErrUsernameNotFound      = "username not found in context"
	ErrUserGroupsNotFound    = "user groups not found in context"
	ErrMissingRequiredHeader = "missing required header: %s"
	ErrTenantIDEmpty         = "tenant ID cannot be empty"
	ErrInvalidTokenClaims    = "invalid token claims"
	ErrTokenExpired          = "token expired"
	ErrTokenIssuedInFuture   = "token issued in the future"
	ErrTokenParseFailed      = "failed to parse token: %w"
)

// Log message constants
const (
	LogJWTValidationFailed    = "JWT token validation failed"
	LogTenantExtractionFailed = "Failed to extract tenant from user groups"
)

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	Sub               string                 `json:"sub"`
	Iat               int64                  `json:"iat"`
	Exp               int64                  `json:"exp"`
	Iss               string                 `json:"iss"`
	Aud               jwt.ClaimStrings       `json:"aud"`
	Typ               string                 `json:"typ"`
	Azp               string                 `json:"azp"`
	Nonce             string                 `json:"nonce"`
	SessionState      string                 `json:"session_state"`
	Acr               string                 `json:"acr"`
	AllowedOrigins    interface{}            `json:"allowed-origins"`
	RealmAccess       map[string]interface{} `json:"realm_access"`
	ResourceAccess    map[string]interface{} `json:"resource_access"`
	Scope             string                 `json:"scope"`
	Sid               string                 `json:"sid"`
	EmailVerified     bool                   `json:"email_verified"`
	Name              string                 `json:"name"`
	PreferredUsername string                 `json:"preferred_username"`
	GivenName         string                 `json:"given_name"`
	FamilyName        string                 `json:"family_name"`
	Email             string                 `json:"email"`
	Groups            []string               `json:"groups"`
}

// GetAudience implements jwt.Claims interface
func (c *JWTClaims) GetAudience() (jwt.ClaimStrings, error) {
	return c.Aud, nil
}

// GetExpirationTime implements jwt.Claims interface
func (c *JWTClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.Exp == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.Exp, 0)), nil
}

// GetIssuedAt implements jwt.Claims interface
func (c *JWTClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	if c.Iat == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.Iat, 0)), nil
}

// GetIssuer implements jwt.Claims interface
func (c *JWTClaims) GetIssuer() (string, error) {
	return c.Iss, nil
}

// GetNotBefore implements jwt.Claims interface
func (c *JWTClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

// GetSubject implements jwt.Claims interface
func (c *JWTClaims) GetSubject() (string, error) {
	return c.Sub, nil
}

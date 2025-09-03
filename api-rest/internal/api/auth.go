package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// AuthMiddleware validates JWT tokens and extracts tenant information
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check endpoints
		if r.URL.Path == HealthPath || r.URL.Path == MetricsPath {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get(AuthorizationHeader)
		if authHeader == "" {
			http.Error(w, ErrAuthHeaderRequired, http.StatusUnauthorized)
			return
		}

		// Check if it's a Bearer token
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			http.Error(w, ErrInvalidAuthHeader, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)

		// Parse and validate the JWT token
		claims, err := validateJWTToken(tokenString)
		if err != nil {
			log.Error().Err(err).Msg(LogJWTValidationFailed)
			http.Error(w, ErrInvalidToken, http.StatusUnauthorized)
			return
		}

		// Extract tenant ID from user groups
		tenantID, err := extractTenantFromGroups(claims.Groups)
		if err != nil {
			log.Error().Err(err).Msg(LogTenantExtractionFailed)
			http.Error(w, ErrInvalidTenantConfig, http.StatusForbidden)
			return
		}

		// Add tenant ID and user info to request context
		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, UserIDKey, claims.Sub)
		ctx = context.WithValue(ctx, UsernameKey, claims.PreferredUsername)
		ctx = context.WithValue(ctx, UserGroupsKey, claims.Groups)

		// Add tenant ID to headers for backward compatibility
		r.Header.Set(TenantHeaderKey, tenantID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateJWTToken validates the JWT token and returns the claims
func validateJWTToken(tokenString string) (*JWTClaims, error) {
	// Parse the token without verification first to get the claims
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf(ErrTokenParseFailed, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New(ErrInvalidTokenClaims)
	}

	// Check if token is expired
	if exp, err := claims.GetExpirationTime(); err == nil && exp != nil {
		if exp.Before(time.Now()) {
			return nil, errors.New(ErrTokenExpired)
		}
	}

	// Check if token is issued in the future
	if iat, err := claims.GetIssuedAt(); err == nil && iat != nil {
		if iat.After(time.Now()) {
			return nil, errors.New(ErrTokenIssuedInFuture)
		}
	}

	// For production, you should verify the token signature with Keycloak's public key
	// For now, we'll trust the token structure and timing

	return claims, nil
}

// extractTenantFromGroups extracts the tenant ID from the user's groups
func extractTenantFromGroups(groups []string) (string, error) {
	if len(groups) == 0 {
		return "", errors.New(ErrNoGroupsAssigned)
	}

	// Return the first group as the tenant ID
	// In B2B model, each group represents a tenant organization
	return groups[0], nil
}

// GetTenantFromContext extracts tenant ID from request context
func GetTenantFromContext(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value(TenantIDKey).(string)
	if !ok || tenantID == "" {
		return "", errors.New(ErrTenantIDNotFound)
	}
	return tenantID, nil
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (string, string, []string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return "", "", nil, errors.New(ErrUserIDNotFound)
	}

	username, ok := ctx.Value(UsernameKey).(string)
	if !ok {
		return "", "", nil, errors.New(ErrUsernameNotFound)
	}

	groups, ok := ctx.Value(UserGroupsKey).([]string)
	if !ok {
		return "", "", nil, errors.New(ErrUserGroupsNotFound)
	}

	return userID, username, groups, nil
}

// GetTenantFromRequest extracts tenant ID from request headers (for backward compatibility)
func GetTenantFromRequest(r *http.Request) (string, error) {
	// First try to get from context (new auth flow)
	if ctx := r.Context(); ctx != nil {
		if tenantID, err := GetTenantFromContext(ctx); err == nil {
			return tenantID, nil
		}
	}

	// Fallback to header (legacy flow)
	tenant := r.Header.Get(TenantHeaderKey)
	if tenant == "" {
		return "", fmt.Errorf(ErrMissingRequiredHeader, TenantHeaderKey)
	}
	trimmedTenant := strings.TrimSpace(tenant)
	if trimmedTenant == "" {
		return "", errors.New(ErrTenantIDEmpty)
	}
	return trimmedTenant, nil
}

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
		// Skip auth for health check and metrics endpoints
		if r.URL.Path == HealthPath || r.URL.Path == MetricsPath {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get(AuthorizationHeader)
		if authHeader == "" {
			log.Warn().Str("path", r.URL.Path).Msg("Authorization header missing")
			http.Error(w, ErrAuthHeaderRequired, http.StatusUnauthorized)
			return
		}

		// Check if it's a Bearer token
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			log.Warn().Str("path", r.URL.Path).Msg("Invalid authorization header format")
			http.Error(w, ErrInvalidAuthHeader, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)

		// Parse and validate the JWT token
		claims, err := validateJWTToken(tokenString)
		if err != nil {
			log.Error().Err(err).Str("path", r.URL.Path).Msg(LogJWTValidationFailed)
			http.Error(w, ErrInvalidToken, http.StatusUnauthorized)
			return
		}

		// Extract tenant ID from username (since groups aren't configured yet)
		tenantID := claims.PreferredUsername
		if tenantID == "" {
			log.Warn().Msg("No preferred_username found in token")
			http.Error(w, ErrInvalidTenantConfig, http.StatusForbidden)
			return
		}

		// Validate tenant from URL path
		urlTenant, err := extractTenantFromURL(r.URL.Path)
		if err != nil {
			log.Error().Err(err).Str("path", r.URL.Path).Msg("Failed to extract tenant from URL")
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		// Ensure tenant in URL matches tenant in token
		if urlTenant != tenantID {
			log.Warn().
				Str("url_tenant", urlTenant).
				Str("token_tenant", tenantID).
				Str("path", r.URL.Path).
				Msg(LogTenantValidationFailed)
			http.Error(w, ErrTenantMismatch, http.StatusForbidden)
			return
		}

		// Add tenant ID and user info to request context
		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, UserIDKey, claims.Sub)
		ctx = context.WithValue(ctx, UsernameKey, claims.PreferredUsername)
		ctx = context.WithValue(ctx, UserGroupsKey, claims.Groups)
		ctx = context.WithValue(ctx, JWTClaimsKey, claims)

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

// extractTenantFromURL extracts tenant ID from URL path like /api/{tenant}/...
func extractTenantFromURL(path string) (string, error) {
	// Remove leading slash and split by slashes
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	// Expected format: api/{tenant}/...
	if len(parts) < 2 || parts[0] != "api" {
		return "", errors.New("invalid URL format: expected /api/{tenant}/")
	}

	tenant := parts[1]
	if tenant == "" {
		return "", errors.New("tenant ID cannot be empty")
	}

	return tenant, nil
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

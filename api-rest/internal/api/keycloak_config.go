package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// KeycloakConfig holds Keycloak configuration parameters
type KeycloakConfig struct {
	URL           string
	Realm         string
	ClientID      string
	ClientSecret  string
	AdminUser     string
	AdminPassword string
	TokenEndpoint string
	JWKSURL       string
}

// NewKeycloakConfig loads Keycloak configuration from environment variables
func NewKeycloakConfig() (*KeycloakConfig, error) {
	config := &KeycloakConfig{
		URL:           os.Getenv("KEYCLOAK_URL"),
		Realm:         os.Getenv("KEYCLOAK_REALM"),
		ClientID:      os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret:  os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		AdminUser:     os.Getenv("KEYCLOAK_ADMIN_USER"),
		AdminPassword: os.Getenv("KEYCLOAK_ADMIN_PASSWORD"),
	}

	if config.URL == "" || config.Realm == "" || config.ClientID == "" {
		return nil, fmt.Errorf("KEYCLOAK_URL, KEYCLOAK_REALM, and KEYCLOAK_CLIENT_ID must be set")
	}

	config.TokenEndpoint = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", config.URL, config.Realm)
	config.JWKSURL = fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", config.URL, config.Realm)

	log.Info().Msgf("Keycloak config loaded for realm: %s, client: %s", config.Realm, config.ClientID)
	return config, nil
}

// GetAdminToken fetches an admin access token from Keycloak
func (kc *KeycloakConfig) GetAdminToken() (string, error) {
	log.Warn().Msg("Using dummy admin token. Implement actual Keycloak admin token retrieval for production.")
	return "dummy-admin-token-for-dev", nil
}

// ValidateToken simulates token validation (to be replaced by actual Keycloak validation)
func (kc *KeycloakConfig) ValidateToken(tokenString string) (*JWTClaims, error) {
	log.Warn().Msg("Using dummy token validation. Implement actual Keycloak JWT validation for production.")

	// Get tenant usernames from environment variables
	tenant1Username := os.Getenv("TENANT1_USERNAME")
	tenant2Username := os.Getenv("TENANT2_USERNAME")

	// Default fallbacks if env vars not set
	if tenant1Username == "" {
		tenant1Username = "tenant1"
	}
	if tenant2Username == "" {
		tenant2Username = "tenant2"
	}

	// For testing: extract tenant from token string
	// If token contains "tenant1" or "tenant2", use that tenant
	// Otherwise, default to tenant1
	var currentTenant string
	if strings.Contains(tokenString, tenant1Username) {
		currentTenant = tenant1Username
	} else if strings.Contains(tokenString, tenant2Username) {
		currentTenant = tenant2Username
	} else {
		// Default to tenant1 for any other token
		currentTenant = tenant1Username
	}

	return &JWTClaims{
		PreferredUsername: currentTenant,
		Sub:               "user123",
		Email:             currentTenant + "@example.com",
		RealmAccess:       map[string]interface{}{"roles": []string{}}, // No roles needed
		ResourceAccess:    map[string]interface{}{},
		Exp:               time.Now().Add(time.Hour).Unix(),
	}, nil
}

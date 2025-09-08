package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// AuthHandlers handles authentication-related HTTP endpoints
type AuthHandlers struct {
	config *KeycloakConfig
}

// NewAuthHandlers creates a new AuthHandlers instance
func NewAuthHandlers(config *KeycloakConfig) *AuthHandlers {
	return &AuthHandlers{
		config: config,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response body
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserInfoResponse represents user information response
type UserInfoResponse struct {
	Sub               string                 `json:"sub"`
	PreferredUsername string                 `json:"preferred_username"`
	Email             string                 `json:"email"`
	EmailVerified     bool                   `json:"email_verified"`
	Name              string                 `json:"name"`
	GivenName         string                 `json:"given_name"`
	FamilyName        string                 `json:"family_name"`
	Roles             []string               `json:"roles"`
	TenantID          string                 `json:"tenant_id"`
	IsAdmin           bool                   `json:"is_admin"`
	Exp               int64                  `json:"exp"`
	Iat               int64                  `json:"iat"`
	AdditionalClaims  map[string]interface{} `json:"additional_claims,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}

// LoginHandler handles user login and token retrieval
func (ah *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Warn().Msgf("Simulating login for user: %s. Implement actual Keycloak login for production.", req.Username)

	resp := LoginResponse{
		AccessToken:  "dummy-access-token-" + req.Username,
		RefreshToken: "dummy-refresh-token-" + req.Username,
		ExpiresIn:    3600, // 1 hour
		TokenType:    "Bearer",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RefreshTokenHandler handles token refresh
func (ah *AuthHandlers) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	log.Warn().Msg("Simulating token refresh. Implement actual Keycloak token refresh for production.")

	resp := LoginResponse{
		AccessToken:  "new-dummy-access-token",
		RefreshToken: "new-dummy-refresh-token",
		ExpiresIn:    3600, // 1 hour
		TokenType:    "Bearer",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UserInfoHandler returns authenticated user information
func (ah *AuthHandlers) UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get JWT claims from context
	claims, ok := ctx.Value(JWTClaimsKey).(*JWTClaims)
	if !ok {
		log.Error().Msg("JWT claims not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tenant ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := UserInfoResponse{
		Sub:               claims.Sub,
		PreferredUsername: claims.PreferredUsername,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		Name:              claims.Name,
		GivenName:         claims.GivenName,
		FamilyName:        claims.FamilyName,
		Roles:             []string{}, // No roles in simplified version
		TenantID:          tenantID,
		IsAdmin:           false, // No admin concept in simplified version
		Exp:               claims.Exp,
		Iat:               claims.Iat,
		AdditionalClaims:  make(map[string]interface{}),
	}

	// Add any additional claims
	if claims.Aud != nil {
		response.AdditionalClaims["aud"] = claims.Aud
	}
	if claims.Iss != "" {
		response.AdditionalClaims["iss"] = claims.Iss
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthHandler provides a health check endpoint
func (ah *AuthHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]string)

	keycloakStatus := "healthy"
	if ah.config == nil || ah.config.URL == "" {
		keycloakStatus = "unhealthy: Keycloak config not loaded"
	}
	services["keycloak"] = keycloakStatus

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // Replace with actual version
		Services:  services,
	}

	if keycloakStatus != "healthy" {
		response.Status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConfigureAuthRoutes sets up authentication-related routes
func ConfigureAuthRoutes(r *mux.Router, config *KeycloakConfig) {
	authHandlers := NewAuthHandlers(config)

	authRouter := r.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/login", authHandlers.LoginHandler).Methods("POST")
	authRouter.HandleFunc("/refresh", authHandlers.RefreshTokenHandler).Methods("POST")
	authRouter.HandleFunc("/userinfo", authHandlers.UserInfoHandler).Methods("GET")

	r.HandleFunc("/health", authHandlers.HealthHandler).Methods("GET")

	log.Info().Msg("Authentication routes configured")
}

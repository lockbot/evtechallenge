#!/bin/sh
set -e

echo "Starting Keycloak setup..."

# Wait for Keycloak to be fully ready using external HTTP health check
# Following Keycloak documentation: "external HTTP requests should be used for health checks"
echo "Waiting for Keycloak to be ready (using external health endpoint)..."
attempts=0; max=30
until curl -sf "http://keycloak:9000/health/ready" > /dev/null; do
  attempts=$((attempts+1))
  if [ $attempts -ge $max ]; then echo "ERROR: Keycloak health check failed after $max attempts"; exit 1; fi
  echo "Waiting for Keycloak health endpoint... ($attempts/$max)"
  sleep 3
done

# Check if Keycloak is already initialized
if [ -f "/tmp/keycloak-init-flag/initialized" ]; then
  echo "Keycloak already initialized, skipping setup..."
  exit 0
fi

echo "Keycloak is ready and healthy!"

# Get admin token
echo "Getting admin token..."
ADMIN_TOKEN=$(curl -s -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=$KEYCLOAK_ADMIN_USER" \
  -d "password=$KEYCLOAK_ADMIN_PASSWORD" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | \
  grep -o '"access_token":"[^"]*' | \
  cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
  echo "ERROR: Failed to get admin token"
  exit 1
fi
echo "Admin token obtained successfully"

# Create realm
echo "Creating realm '$KEYCLOAK_REALM'..."
curl -s -X POST "$KEYCLOAK_URL/admin/realms" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "realm": "'"$KEYCLOAK_REALM"'",
    "enabled": true,
    "displayName": "EVTech Challenge",
    "displayNameHtml": "<div class=\"kc-logo-text\"><span>EVTech Challenge</span></div>",
    "attributes": {
      "frontendUrl": "http://localhost:8080"
    },
    "accessTokenLifespan": 300,
    "accessTokenLifespanForImplicitFlow": 900,
    "ssoSessionIdleTimeout": 1800,
    "ssoSessionMaxLifespan": 36000,
    "offlineSessionIdleTimeout": 2592000,
    "accessCodeLifespan": 60,
    "accessCodeLifespanUserAction": 300,
    "accessCodeLifespanLogin": 1800,
    "actionTokenGeneratedByAdminLifespan": 43200,
    "actionTokenGeneratedByUserLifespan": 300,
    "oauth2DeviceCodeLifespan": 600,
    "oauth2DevicePollingInterval": 5,
    "revokeRefreshToken": true,
    "refreshTokenMaxReuse": 0,
    "loginWithEmailAllowed": true,
    "duplicateEmailsAllowed": false,
    "resetPasswordAllowed": true,
    "editUsernameAllowed": false,
    "bruteForceProtected": true,
    "permanentLockout": false,
    "maxFailureWaitSeconds": 900,
    "minimumQuickLoginWaitSeconds": 60,
    "waitIncrementSeconds": 60,
    "quickLoginCheckMilliSeconds": 1000,
    "maxDeltaTimeSeconds": 43200,
    "failureFactor": 30,
    "defaultSignatureAlgorithm": "RS256"
  }' > /dev/null

echo "Realm created successfully"

# Create or get existing client
echo "Creating client '$KEYCLOAK_CLIENT_ID'..."
CLIENT_CREATE_RESPONSE=$(curl -s -X POST "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "'"$KEYCLOAK_CLIENT_ID"'",
    "name": "API Client",
    "description": "Client for API REST authentication",
    "rootUrl": "http://localhost:8080",
    "adminUrl": "http://localhost:8080",
    "baseUrl": "http://localhost:8080",
    "surrogateAuthRequired": false,
    "enabled": true,
    "clientAuthenticatorType": "client-secret",
    "defaultRoles": ["offline_access", "uma_authorization"],
    "redirectUris": ["http://localhost:8080/*"],
    "webOrigins": ["http://localhost:8080"],
    "notBefore": 0,
    "bearerOnly": false,
    "consentRequired": false,
    "standardFlowEnabled": false,
    "implicitFlowEnabled": false,
    "directAccessGrantsEnabled": true,
    "serviceAccountsEnabled": false,
    "publicClient": false,
    "frontchannelLogout": false,
    "protocol": "openid-connect",
    "attributes": {
      "access.token.lifespan": "300"
    },
    "authenticationFlowBindingOverrides": {},
    "fullScopeAllowed": true,
    "nodeReRegistrationTimeout": -1,
    "defaultClientScopes": ["web-origins", "profile", "roles", "email"],
    "optionalClientScopes": ["address", "phone", "offline_access", "microprofile-jwt"],
    "access": {
      "view": true,
      "configure": true,
      "manage": true
    }
  }')

# Get client ID (whether newly created or existing)
echo "Getting client ID..."
CLIENT_ID=$(curl -s -X GET "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | \
  grep -o '"id":"[^"]*","clientId":"'"$KEYCLOAK_CLIENT_ID"'"' | \
  grep -o '"id":"[^"]*' | \
  cut -d'"' -f4 | \
  head -1)

# Get client secret
if [ -n "$CLIENT_ID" ]; then
  echo "Getting client secret..."
  CLIENT_SECRET=$(curl -s -X GET "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients/$CLIENT_ID/client-secret" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | \
    grep -o '"value":"[^"]*' | \
    cut -d'"' -f4)
else
  echo "ERROR: Could not find client ID"
  CLIENT_SECRET=""
fi

if [ -z "$CLIENT_SECRET" ]; then
  echo "ERROR: Failed to create client or get client secret"
  exit 1
fi
echo "Client created successfully with secret: $CLIENT_SECRET"

# Create tenant1 user
echo "Creating user '$TENANT1_USERNAME'..."
curl -s -X POST "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "'"$TENANT1_USERNAME"'",
    "email": "'"$TENANT1_USERNAME"'@example.com",
    "firstName": "Tenant",
    "lastName": "One",
    "enabled": true,
    "emailVerified": true,
    "credentials": [{
      "type": "password",
      "value": "'"$TENANT1_PASSWORD"'",
      "temporary": false
    }]
  }' > /dev/null

echo "User '$TENANT1_USERNAME' created successfully"

# Create tenant2 user
echo "Creating user '$TENANT2_USERNAME'..."
curl -s -X POST "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "'"$TENANT2_USERNAME"'",
    "email": "'"$TENANT2_USERNAME"'@example.com",
    "firstName": "Tenant",
    "lastName": "Two",
    "enabled": true,
    "emailVerified": true,
    "credentials": [{
      "type": "password",
      "value": "'"$TENANT2_PASSWORD"'",
      "temporary": false
    }]
  }' > /dev/null

echo "User '$TENANT2_USERNAME' created successfully"

echo "Users created successfully - no roles needed for simple tenant-based access"

# Mark as initialized (skip if no write permissions)
mkdir -p /tmp/keycloak-init-flag 2>/dev/null || true
touch /tmp/keycloak-init-flag/initialized 2>/dev/null || true

echo "Keycloak setup completed successfully!"
echo "Realm: $KEYCLOAK_REALM"
echo "Client ID: $KEYCLOAK_CLIENT_ID"
echo "Client Secret: $CLIENT_SECRET"
echo "Tenant users created: $TENANT1_USERNAME, $TENANT2_USERNAME"
echo "Note: Simplified setup - no roles, just tenant-based access"
echo "Access Keycloak admin console at: $KEYCLOAK_URL"
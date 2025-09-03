#!/bin/bash

# Simple Keycloak setup script
# Creates realm, client, and default user

set -e

echo "🔑 Setting up Keycloak..."

# Wait for Keycloak to be ready
echo "⏳ Waiting for Keycloak to be ready..."
until curl -s -f http://localhost:8080/health > /dev/null; do
    echo "   Waiting for Keycloak health endpoint..."
    sleep 5
done

# Get admin token
echo "🔐 Getting admin token..."
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8080/realms/master/protocol/openid-connect/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "username=$KEYCLOAK_ADMIN&password=$KEYCLOAK_ADMIN_PASSWORD&grant_type=password&client_id=admin-cli" | jq -r '.access_token')

if [ "$ADMIN_TOKEN" = "null" ] || [ -z "$ADMIN_TOKEN" ]; then
    echo "❌ Failed to get admin token"
    exit 1
fi

echo "✅ Admin token obtained"

# Create realm
echo "🏛️ Creating realm 'evtechallenge'..."
curl -s -X POST http://localhost:8080/admin/realms \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "realm": "evtechallenge",
    "enabled": true,
    "displayName": "EVT Challenge",
    "loginWithEmailAllowed": false,
    "duplicateEmailsAllowed": false
  }'

echo "✅ Realm created"

# Create client
echo "🔧 Creating client 'evtechallenge-api'..."
curl -s -X POST http://localhost:8080/admin/realms/evtechallenge/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "evtechallenge-api",
    "enabled": true,
    "publicClient": false,
    "redirectUris": ["http://localhost:8080/*"],
    "webOrigins": ["http://localhost:8080"],
    "standardFlowEnabled": true,
    "directAccessGrantsEnabled": true,
    "serviceAccountsEnabled": true
  }'

echo "✅ Client created"

# Create default tenant group
echo "🏢 Creating default tenant group..."
curl -s -X POST http://localhost:8080/admin/realms/evtechallenge/groups \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DefaultTenant"
  }'

echo "✅ Default tenant group created"



echo ""
echo "🎉 Keycloak setup complete!"
echo ""
echo "📋 Setup Information:"
echo "   URL: http://localhost:8082"
echo "   Realm: evtechallenge"
echo "   Client: evtechallenge-api"
echo "   Default tenant group: DefaultTenant"
echo ""
echo "💡 Next steps:"
echo "   1. Create tenant groups in Keycloak admin interface"
echo "   2. Create users and assign them to tenant groups"
echo "   3. Each tenant gets their own Couchbase collection"

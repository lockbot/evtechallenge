# Multi-Tenancy Setup - Simplified

This is a simplified multi-tenant setup that focuses on the essentials.

## What We Have

1. **Keycloak Authentication** - JWT tokens with tenant isolation
2. **Per-Tenant Collections** - Each tenant gets their own Couchbase collection
3. **Backward Compatibility** - Existing `X-Tenant-ID` header usage still works

## What We DON'T Have (By Design)

- Complex tenant management API endpoints
- Automatic tenant creation through API
- Data replication between tenants
- Over-engineered setup scripts

## Setup

### 1. Environment Variables

Copy `env.example` to `.env` and adjust as needed:

```bash
cp env.example .env
```

**Note**: The default values in `.env` will work for development. Keycloak will use `admin/admin` credentials.

### 2. Start Services

```bash
docker-compose up -d
```

### 3. Wait for Setup

```bash
# Wait for Couchbase setup
docker-compose logs -f evtechallenge-db-setup

# Wait for Keycloak setup
docker-compose logs -f keycloak-setup
```

### 4. Automated Keycloak Setup

The setup script automatically configures Keycloak with:
- ✅ Realm `evtechallenge` 
- ✅ Client `evtechallenge-api`
- ✅ Default tenant group `DefaultTenant`


**No manual configuration needed!** 🎉

## How It Works

### Authentication Flow

1. User authenticates with Keycloak
2. Keycloak returns JWT token with user groups
3. API extracts tenant ID from user groups
4. Tenant ID is automatically injected into requests

### Tenant Management

**For regular users:**
- Go to http://localhost:8082
- Click "Register" to create a new account
- Users are automatically assigned to the `DefaultTenant` group

**For admins (optional):**
- Login with `admin/admin` 
- Go to Groups to create new tenant groups
- Assign users to different tenant groups

### Data Isolation

- Each tenant gets a separate Couchbase collection
- Default tenant uses `_default` collection
- Other tenants use `tenant_<tenant_id>` collections
- Complete data separation between tenants

## API Usage

### With JWT Token (Recommended)

```bash
# Get token from Keycloak
curl -X POST http://localhost:8082/realms/evtechallenge/protocol/openid-connect/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'username=your_user&password=your_password&grant_type=password&client_id=evtechallenge-api'

# Use token in API calls
curl -H "Authorization: Bearer <your_jwt_token>" \
     http://localhost:8080/hello
```

### With X-Tenant-ID Header (Legacy)

```bash
curl -H "X-Tenant-ID: default" http://localhost:8080/hello
```

## Key Points

1. **User-Friendly** - Regular users can register at http://localhost:8082
2. **Automated Setup** - No manual Keycloak configuration needed
3. **Security First** - JWT tokens with proper tenant isolation
4. **Backward Compatible** - Existing code continues to work
5. **No Over-Engineering** - Focus on what's actually needed

## Troubleshooting

- Check logs: `docker-compose logs -f`
- Verify Keycloak: http://localhost:8082
- Verify API: http://localhost:8080/health
- Verify Couchbase: http://localhost:8091

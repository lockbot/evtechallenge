# 🔐 Keycloak Setup & Multi-Tenancy Testing Guide

This guide walks you through setting up Keycloak authentication and testing the multi-tenancy system step by step.

## 🚀 Phase 1: Start the Services

### 1.1 Start Docker Compose
```bash
docker-compose up -d
```

### 1.2 Wait for Services to Be Ready
- **Couchbase**: Wait for "Ready" status
- **Keycloak**: Wait for "Ready" status  
- **API REST**: Wait for "Ready" status

Check status with:
```bash
docker-compose ps
```

## 🏗️ Phase 2: Set Up Keycloak (via UI)

### 2.1 Access Keycloak Admin Console
- **URL**: http://localhost:8082
- **Username**: `admin`
- **Password**: `admin`

### 2.2 Create a New Realm
1. Click "Add realm" button
2. **Realm name**: `evtechallenge` (or your preferred name)
3. Click "Create"

### 2.3 Create a New Client
1. Go to "Clients" in left sidebar
2. Click "Create" button
3. **Client ID**: `api-client`
4. **Client Protocol**: `openid-connect`
5. **Root URL**: `http://localhost:8080`
6. Click "Save"

### 2.4 Configure Client Settings
**For your Keycloak version:**
1. **Client authentication**: Turn ON ✅
2. **Direct access grants only**: Turn ON ✅
3. **Standard flow**: Turn OFF ❌
4. **Valid redirect URIs**: `http://localhost:8080/*`
5. **Web origins**: `http://localhost:8080`
6. Click "Save"

### 2.5 Get Client Secret
1. Go to "Credentials" tab
2. Copy the "Secret" value
3. **Save it for Postman use** - you'll need this!

### 2.6 Create a User (Tenant)
1. Go to "Users" in left sidebar
2. Click "Add user" button
3. **Username**: `tenant1` (or your preferred name)
4. **Email**: `tenant1@example.com`
5. **First Name**: `Tenant`
6. **Last Name**: `One`
7. Click "Save"

### 2.7 Set User Password
1. Go to "Credentials" tab
2. **Set Password**: `password123` (or your preferred password)
3. **Temporary**: Turn OFF ❌
4. Click "Set Password"

## 🧪 Phase 3: Test with Postman

### 3.1 Get Access Token
**Request**: `POST` `http://localhost:8082/realms/evtechallenge/protocol/openid-connect/token`

**Headers**:
```
Content-Type: application/x-www-form-urlencoded
```

**Body** (x-www-form-urlencoded):
```
grant_type: password
client_id: api-client
client_secret: [YOUR_CLIENT_SECRET_FROM_STEP_2.5]
username: tenant1
password: password123
```

**Expected Response**:
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 300,
  "refresh_expires_in": 1800,
  "refresh_token": "...",
  "token_type": "Bearer",
  "scope": "openid profile email"
}
```

**Copy the `access_token` value!**

### 3.2 Warm Up Tenant
**Request**: `POST` `http://localhost:8080/warm-up-tenant`

**Headers**:
```
Authorization: Bearer [YOUR_ACCESS_TOKEN]
X-Tenant-ID: tenant1
Content-Type: application/json
```

**Body**: `{}`

**Expected Response**:
```json
{
  "message": "Tenant warming up",
  "tenant": "tenant1"
}
```

### 3.3 Test Hello Endpoint
**Request**: `GET` `http://localhost:8080/hello`

**Headers**:
```
Authorization: Bearer [YOUR_ACCESS_TOKEN]
X-Tenant-ID: tenant1
```

**Expected Response**:
```json
{
  "message": "Hello, World!",
  "status": "success"
}
```

### 3.4 Test FHIR Endpoints
**Request**: `GET` `http://localhost:8080/patients`

**Headers**:
```
Authorization: Bearer [YOUR_ACCESS_TOKEN]
X-Tenant-ID: tenant1
```

**Expected Response**: Either data or empty array (depending on FHIR ingestion status)

## 📊 Phase 4: Verify Tenant Status

### 4.1 Check Tenant Status
**Request**: `GET` `http://localhost:8080/tenant-status/tenant1`

**Headers**:
```
Authorization: Bearer [YOUR_ACCESS_TOKEN]
X-Tenant-ID: tenant1
```

**Expected Response**:
```json
{
  "tenantId": "tenant1",
  "ready": true,
  "warmedAt": "2024-01-15T10:30:00Z",
  "lastRequest": "2024-01-15T10:35:00Z",
  "message": "Tenant ready"
}
```

## 🔍 Troubleshooting Tips

### Common Issues & Solutions

#### Issue 1: "Tenant not ready or collection not initialized"
**Solution**: Make sure you called `/warm-up-tenant` first
**Check**: Tenant status endpoint

#### Issue 2: "Unauthorized" or "Invalid token"
**Solution**: Token expired - get a new one from Keycloak
**Check**: Token expiration time

#### Issue 3: "Bucket not initialized"
**Solution**: Wait for Couchbase to be ready
**Check**: Docker logs for Couchbase

#### Issue 4: "Tenant goroutine manager not initialized"
**Solution**: Wait for API REST to fully start
**Check**: API REST logs

#### Issue 5: "Failed to extract tenant from user groups"
**Solution**: This is expected - the system uses `preferred_username` as tenant ID
**Note**: No action needed, this is working as designed

## 📋 Postman Collection Setup

### Environment Variables
Create a Postman environment with:
```
base_url: http://localhost:8080
keycloak_url: http://localhost:8082
realm: evtechallenge
client_id: api-client
client_secret: [YOUR_CLIENT_SECRET]
access_token: [WILL_BE_SET_AUTOMATICALLY]
```

### Pre-request Script for Token
```javascript
// Debug: Log current environment state
console.log("Current access_token:", pm.environment.get("access_token"));
console.log("Current token_expires_at:", pm.environment.get("token_expires_at"));

// Get new token if current one is expired or missing
if (!pm.environment.get("access_token") || Date.now() > pm.environment.get("token_expires_at")) {
    console.log("Token expired or missing, requesting new one...");
    
    pm.sendRequest({
        url: pm.environment.get("keycloak_url") + "/realms/" + pm.environment.get("realm") + "/protocol/openid-connect/token",
        method: "POST",
        header: {
            "Content-Type": "application/x-www-form-urlencoded"
        },
        body: {
            mode: "urlencoded",
            urlencoded: [
                { key: "grant_type", value: "password" },
                { key: "client_id", value: pm.environment.get("client_id") },
                { key: "client_secret", value: pm.environment.get("client_secret") },
                { key: "username", value: "tenant1" },
                { key: "password", value: "password123" }
            ]
        }
    }, function (err, response) {
        if (err) {
            console.error("Token request failed:", err);
            return;
        }
        
        console.log("Token response received:", response.text());
        
        try {
            const tokenData = response.json();
            console.log("Parsed token data:", tokenData);
            
            pm.environment.set("access_token", tokenData.access_token);
            pm.environment.set("token_expires_at", Date.now() + (tokenData.expires_in * 1000));
            
            console.log("Token saved to environment");
        } catch (parseError) {
            console.error("Failed to parse token response:", parseError);
        }
    });
}

// Wait a bit for the async token request to complete (if needed)
if (!pm.environment.get("access_token")) {
    console.error("No access token available!");
    return;
}

// Set Authorization header
pm.request.headers.add({
    key: "Authorization",
    value: "Bearer " + pm.environment.get("access_token")
});

// Set X-Tenant-ID header
pm.request.headers.add({
    key: "X-Tenant-ID",
    value: "tenant1"
});

console.log("Headers set - Authorization:", pm.environment.get("access_token").substring(0, 20) + "...");
console.log("Headers set - X-Tenant-ID: tenant1");
```

## 🚀 Next Steps After Testing

1. **Test other endpoints**: `/encounters`, `/practitioners`, `/review-request`
2. **Test tenant cool-down**: Wait 30 minutes, then try a request
3. **Test multiple tenants**: Create `tenant2`, `tenant3`, etc.
4. **Test error scenarios**: Invalid tokens, missing headers, etc.

## 🔧 Keycloak Version Notes

**Your Keycloak version uses:**
- ✅ **Client authentication** (instead of "Access Type: confidential")
- ✅ **Direct access grants only** (instead of standard flow)
- ❌ **Standard flow** should be OFF

This configuration allows direct token requests without browser redirects, which is perfect for API testing.

## 📝 Quick Reference Commands

### Check Service Status
```bash
docker-compose ps
```

### View Logs
```bash
# API logs
docker-compose logs -f api-rest

# Keycloak logs  
docker-compose logs -f keycloak

# Couchbase logs
docker-compose logs -f couchbase
```

### Restart Services
```bash
docker-compose restart api-rest
docker-compose restart keycloak
```

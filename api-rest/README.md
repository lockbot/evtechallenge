# API REST Service

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.pt-br.md)

This service exposes a multi-tenant REST API for clinical data access and review management. It integrates with Couchbase for data persistence and provides comprehensive observability through structured logging and metrics.

## Architecture Overview

The API service implements a **multi-tenant architecture** with complete logical isolation through Couchbase scopes and collections. Each tenant has their own dedicated scope with separate collections, ensuring complete data isolation and automatic scalability.

### Multi-Tenant Design
- **Tenant Scopes**: Each tenant gets their own Couchbase scope (e.g., `tenant1`, `tenant2`)
- **Automatic Creation**: Scopes and collections are created automatically on first tenant access
- **Data Isolation**: Complete physical separation of tenant data
- **Review Integration**: Review fields (`reviewed`, `reviewTime`) are embedded directly in FHIR documents
- **Performance**: Direct queries without tenant filters, leveraging native Couchbase indexes

## Quick Start

1) Start required services
```bash
docker-compose up -d evtechallenge-db evtechallenge-db-setup
```

2) Start the API (if enabled in compose) or run locally
```bash
# via compose (uncomment in docker-compose.yml)
docker-compose up -d api

# or run locally
go run ./api-rest
```

## Configuration

Environment variables:
- `COUCHBASE_URL=couchbase://evt-db`
- `COUCHBASE_USERNAME=evtechallenge_user`
- `COUCHBASE_PASSWORD=password`
- `COUCHBASE_BUCKET=EvTeChallenge`
- `API_PORT=8080`
- `API_LOG_LEVEL=info`
- `ELASTICSEARCH_URL=http://elasticsearch:9200`


## API Endpoints

### Health & Status
- `GET /` - Health check with tenant validation
- `GET /hello` - Simple hello endpoint (requires tenant header)
- `POST /all-good` - Business logic validation endpoint (requires tenant header)
- `GET /metrics` - Prometheus metrics endpoint

### FHIR Resource Endpoints

All endpoints use tenant-based routing (`/api/{tenant}/...`) and return FHIR resources with embedded review status.

#### Encounters
- `GET /api/{tenant}/encounters` - List all encounters with embedded review status
- `GET /api/{tenant}/encounters/{id}` - Get specific encounter with embedded review status

#### Patients  
- `GET /api/{tenant}/patients` - List all patients with embedded review status
- `GET /api/{tenant}/patients/{id}` - Get specific patient with embedded review status

#### Practitioners
- `GET /api/{tenant}/practitioners` - List all practitioners with embedded review status
- `GET /api/{tenant}/practitioners/{id}` - Get specific practitioner with embedded review status

### Pagination

All list endpoints support pagination using query parameters:

- `?count=<number>` - Number of items per page (default: 100, max: 10000)
- `?page=<number>` - Page number (default: 1)

**Example:**
```bash
# Get first 50 encounters for tenant1
GET /api/tenant1/encounters?count=50&page=1

# Get second page of 25 patients for tenant1
GET /api/tenant1/patients?count=25&page=2

# Get first 100 practitioners for tenant1 (default)
GET /api/tenant1/practitioners
```

**Paginated Response Format:**
```json
{
  "data": [
    {
      "id": "encounter-123",
      "resource": { /* FHIR resource data */ }
    }
  ],
  "pagination": {
    "page": 1,
    "count": 50,
    "offset": 0,
    "totalItems": 50,
    "hasNext": true
  }
}
```

**Note:** Couchbase has a default limit of 100 documents per query. Use pagination to access larger datasets efficiently.

### Review Management
- `POST /api/{tenant}/review-request` - Mark a resource for review

## Multi-Tenant Architecture

### Tenant Scope Structure
Each tenant has their own Couchbase scope with dedicated collections:

**DefaultScope** (Template Data):
- `encounters`: Original FHIR encounter data
- `patients`: Original FHIR patient data  
- `practitioners`: Original FHIR practitioner data
- `_default`: System ingestion status (`template/ingestion_status`)

**Tenant Scopes** (e.g., `tenant1`, `tenant2`):
- `encounters`: Tenant-specific encounter data with embedded review fields
- `patients`: Tenant-specific patient data with embedded review fields
- `practitioners`: Tenant-specific practitioner data with embedded review fields
- `defaulty`: Tenant ingestion status (`tenant/ingestion_status`)

### Review Integration
Review fields are embedded directly in FHIR documents:

```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "reviewed": true,
  "reviewTime": "2024-01-15T10:30:00Z",
  "subject": { "reference": "Patient/patient-456" },
  "participant": [...]
}
```

### Review Request Endpoint
```bash
POST /api/tenant1/review-request
Headers: Authorization: Bearer <jwt-token>
Body: {
  "entity": "encounter",
  "id": "encounter-123"
}
```

### Response Format
All resource endpoints return FHIR resources with embedded review status:

**Individual Resource:**
```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "reviewed": true,
  "reviewTime": "2024-01-15T10:30:00Z",
  "subject": { "reference": "Patient/patient-456" },
  "participant": [...]
}
```

**List Resources:**
```json
[
  {
    "id": "encounter-123",
    "resourceType": "Encounter",
    "reviewed": true,
    "reviewTime": "2024-01-15T10:30:00Z",
    "subject": { "reference": "Patient/patient-456" },
    "participant": [...]
  }
]
```

## FHIR Data Relationships

### Encounter Structure
Encounters contain references to patients and practitioners:

```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "subject": {
    "reference": "Patient/patient-456"  // Patient ID
  },
  "participant": [
    {
      "individual": {
        "reference": "Practitioner/practitioner-789"  // Practitioner ID
      }
    }
  ]
}
```

### Identifier Extraction Rules

The system extracts identifiers from FHIR references following these rules:

1. **Valid References**: `ResourceType/ID` format
   - `Patient/123` → extracts `123`
   - `Practitioner/456` → extracts `456`

2. **Ignored References**: `urn:uuid:` format
   - `urn:uuid:abc-123-def` → **ignored** (not resolvable via FHIR API)
   - These are inline bundle references that cannot be synced

3. **Missing References**: Handled gracefully
   - Missing `subject.reference` → no patient sync
   - Missing `participant[].individual.reference` → no practitioner sync

### Data Denormalization

For performance and query efficiency, the system denormalizes relationships:

- **Encounter documents** include:
  - `subjectPatientId`: Direct patient ID reference
  - `practitionerIds`: Array of practitioner IDs
  - `docId`: Canonical document key (`Encounter/{id}`)

- **Patient/Practitioner documents** include:
  - `docId`: Canonical document key (`Patient/{id}` or `Practitioner/{id}`)

## Error Handling

### Tenant Validation
- **Invalid Tenant**: `400 Bad Request` - "invalid tenant in URL path"
- **JWT Mismatch**: `403 Forbidden` - "tenant in URL does not match JWT token"

### Resource Operations
- **Not Found**: `404 Not Found` - "resource not found"
- **Database Unavailable**: `503 Service Unavailable` - "database not initialized"
- **Invalid Entity**: `400 Bad Request` - "invalid entity" (for review requests)

## Observability

### Logging
- Structured JSON logs with zerolog
- Tenant ID included in all log entries
- Request/response correlation
- Error context and stack traces

### Metrics
- HTTP request counts and durations
- Business logic metrics (review requests, validation failures)
- System metrics (memory, threads, connections)
- Available at `/metrics` endpoint

### Monitoring
- Grafana dashboards available at `http://localhost:3000`
- Prometheus metrics for alerting and trending


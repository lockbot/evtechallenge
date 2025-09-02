# API REST Service

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/api-rest/README.pt-br.md)

This service exposes a multi-tenant REST API for clinical data access and review management. It integrates with Couchbase for data persistence and provides comprehensive observability through structured logging and metrics.

## Architecture Overview

The API service implements a **multi-tenant architecture** with logical isolation through tenant-specific review documents. Each tenant's review state is stored separately, ensuring complete data isolation between clients.

### Multi-Tenant Design
- **Tenant Identification**: All requests require `X-Tenant-ID` header
- **Review Isolation**: Reviews are stored as separate documents (`Review/{tenantID}`)
- **Data Access**: FHIR resources are shared, but review status is tenant-specific

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
- `COUCHBASE_URL=couchbase://evtechallenge-db`
- `COUCHBASE_USERNAME=evtechallenge_user`
- `COUCHBASE_PASSWORD=password`
- `COUCHBASE_BUCKET=evtechallenge`
- `API_PORT=8080`
- `ELASTICSEARCH_URL=http://elasticsearch:9200`
- `ELASTICSEARCH_INDEX=logs`

## API Endpoints

### Health & Status
- `GET /` - Health check with tenant validation
- `GET /hello` - Simple hello endpoint (requires tenant header)
- `POST /all-good` - Business logic validation endpoint (requires tenant header)
- `GET /metrics` - Prometheus metrics endpoint

### FHIR Resource Endpoints

All endpoints require `X-Tenant-ID` header and return review status for the requesting tenant.

#### Encounters
- `GET /encounters` - List all encounters with review status
- `GET /encounters/{id}` - Get specific encounter with review status

#### Patients  
- `GET /patients` - List all patients with review status
- `GET /patients/{id}` - Get specific patient with review status

#### Practitioners
- `GET /practitioners` - List all practitioners with review status
- `GET /practitioners/{id}` - Get specific practitioner with review status

### Pagination

All list endpoints support pagination using query parameters:

- `?count=<number>` - Number of items per page (default: 100, max: 10000)
- `?page=<number>` - Page number (default: 1)

**Example:**
```bash
# Get first 50 encounters
GET /encounters?count=50&page=1

# Get second page of 25 patients
GET /patients?count=25&page=2

# Get first 100 practitioners (default)
GET /practitioners
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
- `POST /review-request` - Mark a resource for review

## Multi-Tenant Review System

### Review Request Endpoint
```bash
POST /review-request
Headers: X-Tenant-ID: your-tenant-id
Body: {
  "entity": "encounter",
  "id": "encounter-123"
}
```

### Response Format
All resource endpoints return review status:

**Individual Resource:**
```json
{
  "reviewed": true,
  "reviewTime": "2024-01-15T10:30:00Z",
  "data": { /* FHIR resource data */ }
}
```

**List Resources:**
```json
[
  {
    "id": "encounter-123",
    "resource": {
      "reviewed": true,
      "reviewTime": "2024-01-15T10:30:00Z",
      "entityType": "Encounter",
      "entityID": "encounter-123",
      /* ... other FHIR data */
    }
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
- **Missing Header**: `400 Bad Request` - "missing required header: X-Tenant-ID"
- **Empty Header**: `400 Bad Request` - "tenant ID cannot be empty"

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


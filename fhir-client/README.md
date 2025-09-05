# FHIR Client Service

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/fhir-client/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/fhir-client/README.pt-br.md)

Go service that ingests FHIR resources from the public HAPI FHIR API into Couchbase with concurrent processing, resilience, and comprehensive observability.

## Architecture Overview

The FHIR client implements a **two-phase ingestion system**:

1. **Primary Ingestion**: Fetches and stores encounters, patients, and practitioners
2. **Reference Resolution**: Automatically syncs related resources when referenced in encounters
3. **Database Ready Flag**: Sets a global flag when ingestion is complete for API service coordination

### Design Principles
- **Concurrent Processing**: Multiple goroutines for parallel resource ingestion
- **Resilient Operations**: Graceful handling of API failures and timeouts
- **Reference Integrity**: Automatic resolution of FHIR references
- **Observability**: Comprehensive logging and metrics for operational visibility

## Quick Start

1) Start Couchbase and initialization:
```bash
docker-compose up -d evtechallenge-db evtechallenge-db-setup
```

2) Start the client:
```bash
docker-compose up -d fhir
```

## Configuration

Environment variables:
- `COUCHBASE_URL=couchbase://evt-db`
- `COUCHBASE_USERNAME=evtechallenge_user`
- `COUCHBASE_PASSWORD=password`
- `COUCHBASE_BUCKET=evtechallenge`
- `FHIR_BASE_URL=https://hapi.fhir.org/baseR4`
- `FHIR_TIMEOUT=30s`
- `ELASTICSEARCH_URL=http://elasticsearch:9200`


## Ingestion Process

### Resource Types Ingested
- **Encounters**: Primary focus with patient/practitioner references
- **Patients**: Referenced by encounters via `subject.reference`
- **Practitioners**: Referenced by encounters via `participant[].individual.reference`

### Data Flow
1. **Bundle Fetching**: Retrieves FHIR bundles from public API
2. **Resource Classification**: Identifies resource types (Encounter/Patient/Practitioner)
3. **Primary Storage**: Stores resources with denormalized fields
4. **Reference Resolution**: Fetches missing referenced resources
5. **Database Ready**: Sets global flag when complete

### Document Structure

**Encounter Documents** (`Encounter/{id}`):
```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "docId": "Encounter/encounter-123",
  "subjectPatientId": "patient-456",
  "practitionerIds": ["practitioner-789", "practitioner-101"],
  "subject": { "reference": "Patient/patient-456" },
  "participant": [
    { "individual": { "reference": "Practitioner/practitioner-789" } }
  ]
}
```

**Patient/Practitioner Documents** (`Patient/{id}`, `Practitioner/{id}`):
```json
{
  "id": "patient-456",
  "resourceType": "Patient",
  "docId": "Patient/patient-456",
  // ... FHIR resource data
}
```

## Reference Resolution

### Valid Reference Patterns
- `Patient/123` → Fetches patient with ID "123"
- `Practitioner/456` → Fetches practitioner with ID "456"

### Ignored Reference Patterns
- `urn:uuid:abc-123-def` → **Skipped** (inline bundle references)
- These references cannot be resolved via the public FHIR API

### Missing Reference Handling
- Missing `subject.reference` → No patient sync attempted
- Missing `participant[].individual.reference` → No practitioner sync attempted
- Failed API calls → Logged as warnings, ingestion continues

## Observability

### Logging
- **Structured JSON logs** with zerolog
- **Resource-level tracking** (fetch, store, sync operations)
- **Error context** with stack traces
- **Performance metrics** (fetch duration, storage time)

### Metrics
- **FHIR API calls**: Success/failure rates, response times
- **Couchbase operations**: Upsert counts, duration, errors
- **Resource counts**: Encounters, patients, practitioners ingested
- **System metrics**: Memory usage, goroutine count

### Monitoring
- **Grafana dashboards**: `http://localhost:3000`
- **Prometheus metrics**: Available for alerting

## Error Handling & Resilience

### API Failures
- **Timeout handling**: Configurable timeouts with retry logic
- **HTTP errors**: Graceful degradation with detailed logging
- **Network issues**: Automatic retry with exponential backoff

### Data Inconsistencies
- **Missing fields**: Graceful handling of incomplete FHIR data
- **Invalid references**: Logged warnings, ingestion continues
- **Duplicate resources**: Idempotent upsert operations

### Database Issues
- **Connection failures**: Automatic reconnection attempts
- **Storage errors**: Detailed error logging with context
- **Query failures**: Fallback to key-value operations

## Improvement Suggestions

### 🔧 **Flag for Failed Identifiers**
**Current State**: Failed identifier resolution is logged but not flagged for review.

**Suggested Enhancement**:
```go
// Add to ReviewDocument structure
type ReviewDocument struct {
    TenantID string                 `json:"tenantId"`
    Encounters map[string]interface{} `json:"encounters"`
    Patients map[string]interface{} `json:"patients"`
    Practitioners map[string]interface{} `json:"practitioners"`
    FailedIdentifiers []FailedIdentifier `json:"failedIdentifiers,omitempty"`
    Updated  time.Time              `json:"updated"`
}

type FailedIdentifier struct {
    Reference   string `json:"reference"`
    ResourceType string `json:"resourceType"`
    Reason      string `json:"reason"` // "urn:uuid", "api_failure", "not_found"
    Timestamp   time.Time `json:"timestamp"`
}
```

**Benefits**:
- Track all failed identifier resolutions
- Distinguish between `urn:uuid` (expected) vs API failures (actionable)
- Provide audit trail for data quality issues
- Enable automated alerts for systematic failures

### 🔧 **Enhanced Error Classification**
**Current State**: All failures are logged as warnings.

**Suggested Enhancement**:
```go
type IngestionError struct {
    Type        string `json:"type"` // "urn_uuid", "api_failure", "network_timeout"
    Reference   string `json:"reference"`
    ResourceType string `json:"resourceType"`
    Severity    string `json:"severity"` // "info", "warning", "error"
    Retryable   bool   `json:"retryable"`
}
```

**Benefits**:
- Better error categorization for monitoring
- Distinguish between expected vs unexpected failures
- Enable targeted alerting and response
- Support for retry strategies

### 🔧 **Ingestion Status Tracking**
**Current State**: Simple "dbReady" flag.

**Suggested Enhancement**:
```go
type IngestionStatus struct {
    Status      string    `json:"status"` // "running", "completed", "failed"
    StartTime   time.Time `json:"startTime"`
    EndTime     time.Time `json:"endTime,omitempty"`
    Resources   ResourceCounts `json:"resources"`
    Errors      []IngestionError `json:"errors,omitempty"`
    FailedRefs  []FailedIdentifier `json:"failedRefs,omitempty"`
}

type ResourceCounts struct {
    Encounters    int `json:"encounters"`
    Patients      int `json:"patients"`
    Practitioners int `json:"practitioners"`
}
```


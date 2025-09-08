# EVT Challenge - Clinical Data Platform

[![en](https://img.shields.io/badge/lang-en-red.svg)](https://github.com/lockbot/evtechallenge/blob/main/README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](https://github.com/lockbot/evtechallenge/blob/main/README.pt-br.md)

A multi-tenant clinical data ingestion and API platform built with Go, featuring FHIR data ingestion, Couchbase persistence, and comprehensive observability.

## Architecture Overview

This platform consists of **two microservices** working together to provide a complete clinical data solution:

### Services
- **FHIR Client** (`fhir-client/`): Ingests FHIR resources from public API into Couchbase
- **API REST** (`api-rest/`): Multi-tenant REST API for data access and review management

### Infrastructure
- **Couchbase**: Multi-tenant document database with N1QL support
- **Elasticsearch**: Centralized logging with structured JSON logs
- **Prometheus**: Metrics collection and monitoring
- **Grafana**: Visualization and dashboards

## Quick Start

### Full Stack (Recommended)
Start the complete platform with observability:
```bash
docker-compose --profile observability up
```

### Individual Services

#### Start Just the Database
```bash
docker-compose up -d evtechallenge-db evtechallenge-db-setup
```

#### Start Just FHIR Client
```bash
docker-compose up -d --no-deps fhir
```

#### Start Just API REST
```bash
docker-compose up -d --no-deps api
```

### Service Management

#### Stop Individual Services
```bash
# Stop FHIR client only
docker-compose stop fhir

# Stop API only  
docker-compose stop api

# Stop database only
docker-compose stop evtechallenge-db
```

#### Complete Cleanup ⚠️
```bash
# Stop all services and remove volumes (⚠️WARNING⚠️: DELETES DATABASE)
docker-compose down -v

# Stop all services but preserve data
docker-compose down
```

## Configuration

Create a `.env` file in the repository root:

```bash
# API Configuration
API_PORT=8080
API_LOG_LEVEL="info"

# FHIR Client Configuration
FHIR_PORT=8081
FHIR_LOG_LEVEL="info"
FHIR_BASE_URL=http://hapi.fhir.org/baseR4
FHIR_TIMEOUT=30s

# Couchbase Configuration
COUCHBASE_URL=couchbase://evt-db
COUCHBASE_ADMINISTRATOR_USERNAME=Administrator
COUCHBASE_ADMINISTRATOR_PASSWORD=password
COUCHBASE_USERNAME=evtechallenge_user
COUCHBASE_PASSWORD=password
COUCHBASE_BUCKET=evtechallenge
COUCHBASE_MANAGEMENT_HOST=evt-db:8091

# Observability (optional)
ENABLE_ELASTICSEARCH=false
ENABLE_SYSTEM_METRICS=false
ENABLE_BUSINESS_METRICS=false
ELASTICSEARCH_URL=http://elasticsearch:9200

# Grafana Configuration (optional - defaults work)
GRAFANA_ADMIN_PASSWORD=admin
GRAFANA_PORT=3000

# Elasticsearch Configuration (optional - defaults work)
ELASTICSEARCH_PORT=9200
ELASTICSEARCH_TRANSPORT_PORT=9300

# Prometheus Configuration (optional - defaults work)
PROMETHEUS_PORT=9090

# Keycloak Configuration
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_PORT=8082
KEYCLOAK_REALM=evtechallenge
KEYCLOAK_CLIENT_ID=api-client
KEYCLOAK_CLIENT_SECRET=
KEYCLOAK_ADMIN_USER=admin
KEYCLOAK_ADMIN_PASSWORD=admin
# Note: These map to KC_BOOTSTRAP_ADMIN_USERNAME and KC_BOOTSTRAP_ADMIN_PASSWORD in docker-compose.yml
KEYCLOAK_LOG_LEVEL=INFO

# Tenant Configuration
TENANT1_USERNAME=tenant1
TENANT1_PASSWORD=tnt1
TENANT2_USERNAME=tenant2
TENANT2_PASSWORD=tnt2
```

## API Endpoints

### Authentication (No tenant required)
- `POST /auth/login` - User login
- `POST /auth/refresh` - Refresh token
- `GET /auth/userinfo` - Get user information
- `GET /health` - System health check

### FHIR Resources (Tenant-based routing)
- `GET /api/{tenant}/encounters` - List encounters for tenant
- `GET /api/{tenant}/encounters/{id}` - Get specific encounter
- `GET /api/{tenant}/patients` - List patients for tenant
- `GET /api/{tenant}/patients/{id}` - Get specific patient
- `GET /api/{tenant}/practitioners` - List practitioners for tenant
- `GET /api/{tenant}/practitioners/{id}` - Get specific practitioner

### Review System (Tenant-based routing)
- `POST /api/{tenant}/review-request` - Submit review request

### System
- `GET /` - API information
- `GET /metrics` - Prometheus metrics

### Legacy Endpoints (Deprecated - use X-Tenant-ID header)
- `GET /legacy/encounters` - List encounters (legacy)
- `GET /legacy/patients` - List patients (legacy)
- `GET /legacy/practitioners` - List practitioners (legacy)
- `POST /legacy/review-request` - Submit review request (legacy)

### Authentication
All tenant-based endpoints require JWT authentication via `Authorization: Bearer <token>` header. The tenant in the URL path must match the tenant in the JWT token.

## Technical Decisions

### Architecture Decisions

#### **Microservice Separation**
**Decision**: Separate ingestion (FHIR client) and API (REST service) into distinct containers.

**Rationale**:
- **Independent Scaling**: Can scale ingestion and API separately based on load
- **Fault Isolation**: API failures don't affect data ingestion
- **Deployment Flexibility**: Can deploy updates independently
- **Resource Optimization**: Different resource requirements for each service

#### **Couchbase as Primary Database**
**Decision**: Use Couchbase for data persistence instead of traditional RDBMS.

**Rationale**:
- **Schema Flexibility**: FHIR data structure varies and evolves
- **Fast Development**: No schema migrations or complex modeling
- **Scalability**: Horizontal scaling with automatic sharding
- **Multi-Model**: Key-value, document, and N1QL query support
- **Performance**: In-memory caching with disk persistence

**Trade-offs**:
- Less ACID compliance than traditional databases
- Learning curve for N1QL vs SQL
- Operational complexity for cluster management

#### **Multi-Tenant Design**
**Decision**: Implement tenant isolation through separate review documents.

**Rationale**:
- **Logical Isolation**: Each tenant's review state is completely separate
- **Shared Data**: FHIR resources are shared (cost-effective)
- **Scalability**: Easy to add new tenants without schema changes
- **Security**: Clear data boundaries between tenants

**Implementation**:
- Tenant identification via `X-Tenant-ID` header
- Review documents stored as `Review/{tenantID}` with separate maps for each resource type
- All API endpoints require tenant header

### Data Modeling Decisions

#### **Document Structure**
**Decision**: Denormalize relationships for query performance.

**Encounter Documents**:
```json
{
  "id": "encounter-123",
  "resourceType": "Encounter",
  "docId": "Encounter/encounter-123",
  "subjectPatientId": "patient-456",
  "practitionerIds": ["practitioner-789"],
  "subject": { "reference": "Patient/patient-456" },
  "participant": [...]
}
```

**Benefits**:
- Fast queries without joins
- Direct access to related IDs
- Maintains original FHIR structure
- Supports both key-value and N1QL access

#### **Reference Resolution Strategy**
**Decision**: Automatic resolution of FHIR references with graceful failure handling.

**Valid References**: `Patient/123`, `Practitioner/456`
**Ignored References**: `urn:uuid:abc-123-def` (inline bundle references)

**Benefits**:
- Complete data relationships
- Handles inconsistent FHIR data gracefully
- Distinguishes between resolvable and non-resolvable references

### Observability Decisions

#### **Structured Logging & Metrics**
**Decision**: Use zerolog with JSON formatting and Elasticsearch integration, plus comprehensive Prometheus metrics.

**Benefits**:
- Machine-readable logs for analysis
- Centralized log aggregation and correlation across services
- Comprehensive metrics coverage (HTTP requests, business logic, system resources, FHIR API calls)

## Monitoring & Observability

### Grafana Dashboards
Access at `http://localhost:3000`

**Login Credentials**:
- Username: `admin`
- Password: `admin`

**Available Dashboards**:
- **System Metrics**: Memory usage, CPU, thread counts
- **API Performance**: Request rates, response times, error rates
- **FHIR Ingestion**: Resource counts, API call success rates
- **Business Metrics**: Review requests, tenant activity

**Note**: Logs are available through Elasticsearch integration within Grafana dashboards.

## Development

### Project Structure
```
evtechallenge/
├── api-rest/           # Multi-tenant REST API service
├── fhir-client/        # FHIR data ingestion service
├── config/             # Configuration files
│   ├── grafana/        # Grafana dashboards and configuration
│   └── prometheus/     # Prometheus basic configuration
├── docker-compose.yml  # Service orchestration
└── README.md          # This file
```

### Development Workflow
**Key Files**:
- `docker-compose.yml`: Service definitions and networking
- `api-rest/internal/api/`: API service implementation
- `fhir-client/internal/fhir/`: FHIR ingestion logic
- `config/grafana/dashboards/`: Pre-configured Grafana dashboards
- `config/prometheus/`: Basic Prometheus configuration (connection/healthcheck)

**Adding New Features**:
1. **API Endpoints**: Add to `api-rest/internal/api/handlers.go`
2. **Data Models**: Define in `api-rest/internal/api/types.go`
3. **Database Operations**: Implement in `api-rest/internal/api/database.go`
4. **Review Logic**: Extend `api-rest/internal/api/review.go`

## Troubleshooting

### Common Issues

#### Database Connection Failures
```bash
# Check Couchbase status
docker-compose logs evtechallenge-db

# Restart database
docker-compose restart evtechallenge-db
```

#### API Service Unavailable
```bash
# Check API logs
docker-compose logs api

# Verify database is ready
curl http://localhost:8080/hello -H "X-Tenant-ID: test"
```

#### FHIR Ingestion Issues
```bash
# Check FHIR client logs
docker-compose logs fhir

# Verify external API access
curl https://hapi.fhir.org/baseR4/Patient?_count=1
```

**Health Checks**:
- **API Health**: `GET /` (requires tenant header)
- **Database Health**: Check Couchbase web UI at `http://localhost:8091`
- **Metrics Health**: `GET /metrics`

## Documentation

- **API REST**: [api-rest/README.md](api-rest/README.md)
- **FHIR Client**: [fhir-client/README.md](fhir-client/README.md)
- **Docker Compose**: [docker-compose.yml](docker-compose.yml)
- **Keycloak Setup**: [docs/keycloak-setup.md](docs/keycloak-setup.md)
- **ADR (Architecture Decision Records)**: [docs/README.md](docs/README.md)

## Security & Future Enhancements

**Security Considerations**:
- **Multi-tenant isolation** ensures data separation
- **Input validation** on all API endpoints
- **Environment variables** for sensitive configuration
- **No hardcoded credentials** in source code

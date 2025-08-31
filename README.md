# evtechallenge
Stealth Company Technical Challenge

Monorepo with two services:
- API REST (`api-rest/`)
- FHIR Client (`fhir-client/`)

## Environment

Create a `.env` file in the repository root before starting the stack. Example:

```
# Couchbase
COUCHBASE_URL=couchbase://evtechallenge-db
COUCHBASE_USERNAME=evtechallenge_user
COUCHBASE_PASSWORD=password

# FHIR client
FHIR_BASE_URL=https://hapi.fhir.org/baseR4
FHIR_TIMEOUT=30s
# Optional: port if the client exposes one
FHIR_PORT=8081

# Observability (optional if you enable these services in docker-compose)
ENABLE_ELASTICSEARCH=false
ENABLE_SYSTEM_METRICS=true
ENABLE_BUSINESS_METRICS=true
ELASTICSEARCH_URL=http://localhost:9200
ELASTICSEARCH_INDEX=logs

# API (if/when you enable the API service)
API_PORT=8080
```

Adjust values as needed for your environment. `docker-compose.yml` reads these variables when bringing up services.

## Getting Started

Start the full stack (services + observability) in one command:
```bash
docker-compose --profile observability up
```

Notes:
- Add `-d` to run in the background.
- The FHIR client and Couchbase initialization run automatically. See `api-rest/README.md` to enable and use the API service.

## Documentation
- API REST: see `api-rest/README.md`
- FHIR Client: see `fhir-client/README.md`

## Observability
Grafana, Prometheus, and Elasticsearch can be enabled via `docker-compose.yml`.
# API REST

This service exposes the REST API for evtechallenge. It integrates with Couchbase and publishes logs/metrics.

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

Environment variables (examples):
- COUCHBASE_URL=couchbase://evtechallenge-db
- COUCHBASE_USERNAME=evtechallenge_user
- COUCHBASE_PASSWORD=password
- ENABLE_ELASTICSEARCH=true
- ELASTICSEARCH_URL=http://localhost:9200

## Endpoints

Documented within the service code. Typical examples:
- GET /health
- GET /metrics



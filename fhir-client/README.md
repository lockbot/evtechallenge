# FHIR Client

Go service that ingests FHIR resources into Couchbase.

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

Environment variables (examples):
- COUCHBASE_URL=couchbase://evtechallenge-db
- COUCHBASE_USERNAME=evtechallenge_user
- COUCHBASE_PASSWORD=password
- FHIR_BASE_URL=https://hapi.fhir.org/baseR4
- FHIR_TIMEOUT=30s

## Notes
- N1QL is enabled by default via compose (data,index,query). A primary index on `evtechallenge` is created automatically.
- Inline references `urn:uuid:*` are kept in documents but not resolved against the public FHIR API.

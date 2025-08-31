# FHIR Couchbase N1QL Mock

Run the mock from the repository root:

```bash
docker-compose -f docker-compose.mock.yml up --build --abort-on-container-exit
```

This builds a small container that connects to Couchbase using the default envs:
- COUCHBASE_URL=couchbase://evtechallenge-db
- COUCHBASE_USERNAME=evtechallenge_user
- COUCHBASE_PASSWORD=password

Override these with environment variables if needed.

FROM golang:1.25.0-alpine3.22 AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

# Copy dependency files first (for better layer caching)
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build from the root context
RUN go build -o /mock ./fhir-client/cmd/index-query-couchbase

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates
COPY --from=builder /mock /mock

# Default envs (can be overridden at runtime)
ENV COUCHBASE_URL=couchbase://evtechallenge-db \
    COUCHBASE_USERNAME=evtechallenge_user \
    COUCHBASE_PASSWORD=password

ENTRYPOINT ["/mock"]


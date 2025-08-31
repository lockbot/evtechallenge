FROM golang:1.25.0-alpine3.22 AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

# Copy module files and download deps
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source
COPY . .

# Build the mock binary
RUN go build -o /mock ./cmd/index-query-couchbase

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates
COPY --from=builder /mock /mock

# Default envs (can be overridden at runtime)
ENV COUCHBASE_URL=couchbase://evtechallenge-db \
    COUCHBASE_USERNAME=evtechallenge_user \
    COUCHBASE_PASSWORD=password

ENTRYPOINT ["/mock"]


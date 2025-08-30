FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o orch .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o ingest ./cmd/ingest

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/orch .
COPY --from=builder /app/api ./api
COPY --from=builder /app/ingest ./ingest

EXPOSE $API_PORT $INGEST_PORT $ORCHESTRATOR_PORT

CMD ["./orch"]
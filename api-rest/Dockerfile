FROM golang:1.25.0-alpine3.22 AS builder

WORKDIR /app

# Copy dependency files first (for better layer caching)
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build from the api-rest directory
RUN go build -o api ./api-rest/main.go

EXPOSE ${API_PORT}

CMD ["./api"]
FROM golang:1.25.0-alpine3.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -o fhir main.go

EXPOSE ${FHIR_PORT}

CMD ["./fhir"]
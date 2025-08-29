FROM golang:1.25.0-alpine3.22

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -o evtechallenge-orch main.go

EXPOSE 8080

CMD ["./evtechallenge-orch"]
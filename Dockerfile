# build
FROM golang:1.24 AS builder

ENV GO111MODULE=on

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

RUN go mod download

# Copy source code
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ulak cmd/main.go

# run
FROM alpine:latest

EXPOSE 8080
COPY --from=builder /app/ulak /app/ulak

WORKDIR /app

ENTRYPOINT [ "./ulak" ]

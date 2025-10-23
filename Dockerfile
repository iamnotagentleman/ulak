# build
FROM golang:1.23 AS builder

ENV GO111MODULE=on

WORKDIR /app

COPY ulak .

RUN go mod download

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ulak cmd/main.go

# run
FROM alpine:latest

EXPOSE 8080
COPY --from=builder /app/ulak /app/ulak

WORKDIR /app

ENTRYPOINT [ "./ulak" ]

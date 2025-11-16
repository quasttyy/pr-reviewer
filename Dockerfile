FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api ./cmd/api

FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/api /app/api
COPY config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/api"]


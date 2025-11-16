SHELL := /bin/sh

.PHONY: build run migrate test compose-up compose-down

build:
	GOOS=linux GOARCH=amd64 go build -o bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -o bin/migrator ./cmd/migrator

run:
	go run ./cmd/api

migrate:
	go run ./cmd/migrator

test:
	go test ./...

compose-up:
	docker compose up --build

compose-down:
	docker compose down -v


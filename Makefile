SHELL := /bin/sh

.PHONY: build run migrate test compose-up compose-down

build:
\tGOOS=linux GOARCH=amd64 go build -o bin/api ./cmd/api
\tGOOS=linux GOARCH=amd64 go build -o bin/migrator ./cmd/migrator

run:
\tgo run ./cmd/api

migrate:
\tgo run ./cmd/migrator

test:
\tgo test ./...

compose-up:
\tdocker compose up --build

compose-down:
\tdocker compose down -v


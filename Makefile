APP_NAME := sso
DATABASE_URL ?= postgres://sso:sso@localhost:5432/sso?sslmode=disable

.PHONY: up down migrate test tidy build

up:
	docker compose up --build

down:
	docker compose down

migrate:
	docker compose run --rm migrate

test:
	go test ./...

tidy:
	go mod tidy

build:
	go build ./cmd/sso

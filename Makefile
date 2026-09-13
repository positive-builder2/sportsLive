.PHONY: nats ingestion scoring commentary gateway test build dev up down logs

nats:
	docker compose up -d nats redis postgres

ingestion:
	go run ./cmd/ingestion

scoring:
	go run ./cmd/scoring

commentary:
	go run ./cmd/commentary

gateway:
	go run ./cmd/gateway

dev:
	bash scripts/dev.sh

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f scoring commentary gateway ingestion

test:
	go test ./...

build:
	go build ./...

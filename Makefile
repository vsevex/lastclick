.PHONY: build run dev migrate-up migrate-down docker-up docker-down deploy deploy-down

build:
	go build -o bin/lastclick ./cmd/server

run: build
	./bin/lastclick

dev:
	go run ./cmd/server

migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir migrations postgres "$$DATABASE_URL" down

docker-up:
	docker compose up -d

docker-down:
	docker compose down

deploy:
	docker compose -f docker-compose.prod.yml up -d --build

deploy-down:
	docker compose -f docker-compose.prod.yml down

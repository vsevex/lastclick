.PHONY: build run dev migrate-up migrate-down docker-up docker-down

build:
	go build -o bin/lastclick ./cmd/server

run: build
	./bin/lastclick

dev:
	go run ./cmd/server

migrate-up:
	goose -dir migrations postgres "postgres://vsevex:1596225600@localhost:5432/lastclick?sslmode=disable" up

migrate-down:
	goose -dir migrations postgres "postgres://vsevex:1596225600@localhost:5432/lastclick?sslmode=disable" down

docker-up:
	docker compose up -d

docker-down:
	docker compose down

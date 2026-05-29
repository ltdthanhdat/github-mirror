.PHONY: run build test worker

BASIC_AUTH_USERNAME ?= admin
BASIC_AUTH_PASSWORD ?= admin123
DATABASE_URL ?= postgres://mirror:mirrorpass@localhost:5432/mirror?sslmode=disable

run:
	BASIC_AUTH_USERNAME=$(BASIC_AUTH_USERNAME) \
	BASIC_AUTH_PASSWORD=$(BASIC_AUTH_PASSWORD) \
	DATABASE_URL=$(DATABASE_URL) \
	go run ./cmd/mirror server

worker:
	DATABASE_URL=$(DATABASE_URL) \
	BASIC_AUTH_USERNAME=$(BASIC_AUTH_USERNAME) \
	BASIC_AUTH_PASSWORD=$(BASIC_AUTH_PASSWORD) \
	go run ./cmd/mirror worker

build:
	go build ./cmd/mirror

test:
	go test ./...

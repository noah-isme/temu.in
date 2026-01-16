
AIR_BIN ?= $(shell command -v air 2>/dev/null)

.PHONY: help dev build run test lint migrate-create

help:
	@echo "Makefile targets:"
	@echo "  make dev            # start backend with hot reload (requires 'air' installed)"
	@echo "  make build          # build the backend binary"
	@echo "  make run            # build and run the backend binary"
	@echo "  make test           # run tests"
	@echo "  make lint           # run linter"
	@echo "  make migrate-create # create a new migration (usage: make migrate-create NAME=migration_name)"

dev:
	@# If air not on PATH, try go env GOBIN or GOPATH/bin
	@A_BIN="$(AIR_BIN)"; \
	if [ -z "$$A_BIN" ]; then \
		if [ -x "$(shell go env GOBIN)/air" ]; then \
			A_BIN="$(shell go env GOBIN)/air"; \
		elif [ -x "$(shell go env GOPATH)/bin/air" ]; then \
			A_BIN="$(shell go env GOPATH)/bin/air"; \
		fi; \
	fi; \
	if [ -z "$$A_BIN" ]; then \
		echo "'air' not found. Install it with: go install github.com/air-verse/air@latest"; exit 1; \
	fi; \
	exec "$$A_BIN" -c .air.toml

build:
	go build -o ./tmp/main ./cmd/api

run: build
	./tmp/main

test:
	go test -v ./...

lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; exit 1; \
	fi
	@NEXT=$$(ls -1 internal/database/migrations/*.sql 2>/dev/null | wc -l); \
	NEXT=$$((NEXT / 2 + 1)); \
	PADDED=$$(printf "%06d" $$NEXT); \
	touch "internal/database/migrations/$${PADDED}_$(NAME).up.sql"; \
	touch "internal/database/migrations/$${PADDED}_$(NAME).down.sql"; \
	echo "Created migrations/$${PADDED}_$(NAME).up.sql and .down.sql"

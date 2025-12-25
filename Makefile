.PHONY: help build run dev test lint clean db-up db-down db-reset frontend-dev frontend-build docker-build docker-up docker-down

# Default target
help:
	@echo "doc-analyzer - Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Run server in development mode"
	@echo "  make frontend-dev - Run frontend dev server"
	@echo "  make test         - Run all tests"
	@echo "  make lint         - Run linter"
	@echo ""
	@echo "Database:"
	@echo "  make db-up        - Start PostgreSQL container"
	@echo "  make db-down      - Stop PostgreSQL container"
	@echo "  make db-reset     - Reset database (drop and recreate)"
	@echo ""
	@echo "Build:"
	@echo "  make build        - Build Go binary"
	@echo "  make frontend-build - Build frontend for production"
	@echo "  make docker-build - Build Docker images"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up    - Start all services with Docker"
	@echo "  make docker-down  - Stop all Docker services"
	@echo ""
	@echo "Legacy (Python - in src-old/):"
	@echo "  make legacy-install - Install old Python package"
	@echo "  make legacy-analyze - Run old Python analyzer"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean        - Remove build artifacts"

# Build
build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

dev:
	go run ./cmd/server

# Testing
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Linting
lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

# Database
db-up:
	docker compose up -d db

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d db
	@echo "Waiting for database to be ready..."
	@sleep 3
	@echo "Database reset complete"

db-logs:
	docker compose logs -f db

db-shell:
	docker compose exec db psql -U docanalyzer -d docanalyzer

# Frontend
frontend-dev:
	cd web && npm run dev

frontend-build:
	cd web && npm run build

frontend-install:
	cd web && npm install

# Docker
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# Cleanup
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -f coverage.out coverage.html

# Legacy Python support (from src-old/)
legacy-install:
	cd src-old && pip install -e .

legacy-analyze:
	cd src-old && doc-analyzer analyze $(DOCS_PATH) --verbose

# Development setup
setup: frontend-install db-up
	@echo "Development environment ready!"
	@echo "Run 'make dev' to start the backend server"
	@echo "Run 'make frontend-dev' in another terminal for frontend"

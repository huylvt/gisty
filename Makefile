.PHONY: up down logs ps restart clean mongo-shell redis-cli help build run test lint \
        prod-build prod-up prod-down prod-logs prod-ps

# Default target
help:
	@echo "Gisty Development Commands"
	@echo "=========================="
	@echo "  make up          - Start all services (dev)"
	@echo "  make down        - Stop all services (dev)"
	@echo "  make logs        - View logs (all services)"
	@echo "  make logs-mongo  - View MongoDB logs"
	@echo "  make logs-redis  - View Redis logs"
	@echo "  make ps          - List running services"
	@echo "  make restart     - Restart all services"
	@echo "  make clean       - Stop and remove volumes"
	@echo "  make mongo-shell - Open MongoDB shell"
	@echo "  make redis-cli   - Open Redis CLI"
	@echo "  make health      - Check services health"
	@echo ""
	@echo "Build Commands"
	@echo "=============="
	@echo "  make build       - Build the Go binary"
	@echo "  make run         - Run the server locally"
	@echo "  make test        - Run all tests"
	@echo "  make test-unit   - Run unit tests only"
	@echo "  make test-int    - Run integration tests only"
	@echo "  make lint        - Run linter"
	@echo ""
	@echo "Production Commands"
	@echo "==================="
	@echo "  make prod-build  - Build production Docker image"
	@echo "  make prod-up     - Start production stack"
	@echo "  make prod-down   - Stop production stack"
	@echo "  make prod-logs   - View production logs"
	@echo "  make prod-ps     - List production services"
	@echo "  make prod-clean  - Stop and remove production volumes"

# Start services
up:
	docker compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@make health

# Stop services
down:
	docker compose down

# View logs
logs:
	docker compose logs -f

logs-mongo:
	docker compose logs -f mongodb

logs-redis:
	docker compose logs -f redis

# List services
ps:
	docker compose ps

# Restart services
restart:
	docker compose restart

# Clean up (remove volumes)
clean:
	docker compose down -v
	@echo "Volumes removed."

# MongoDB shell
mongo-shell:
	docker exec -it gisty-mongodb mongosh -u gisty -p gisty123 --authenticationDatabase admin gisty

# Redis CLI
redis-cli:
	docker exec -it gisty-redis redis-cli

# Health check
health:
	@echo "Checking MongoDB..."
	@docker exec gisty-mongodb mongosh --eval "db.adminCommand('ping')" --quiet || echo "MongoDB: NOT READY"
	@echo "Checking Redis..."
	@docker exec gisty-redis redis-cli ping || echo "Redis: NOT READY"
	@echo ""
	@echo "Service Status:"
	@docker compose ps --format "table {{.Name}}\t{{.Status}}"

# ===========================================
# Build Commands
# ===========================================

# Build Go binary
build:
	CGO_ENABLED=1 go build -ldflags="-w -s" -o bin/gisty ./cmd/server

# Run server locally
run:
	go run ./cmd/server

# Run all tests
test:
	go test ./... -v

# Run unit tests only
test-unit:
	go test ./internal/... ./pkg/... -v

# Run integration tests only
test-int:
	go test ./tests/integration/... -v -timeout 10m

# Run linter
lint:
	golangci-lint run ./...

# Generate Swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs

# ===========================================
# Production Commands
# ===========================================

# Build production Docker image
prod-build:
	docker build -t gisty:latest .

# Start production stack
prod-up:
	docker compose -f docker-compose.prod.yml up -d
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@make prod-ps

# Stop production stack
prod-down:
	docker compose -f docker-compose.prod.yml down

# View production logs
prod-logs:
	docker compose -f docker-compose.prod.yml logs -f

# List production services
prod-ps:
	docker compose -f docker-compose.prod.yml ps

# Clean production (remove volumes)
prod-clean:
	docker compose -f docker-compose.prod.yml down -v
	@echo "Production volumes removed."

# Restart production app only
prod-restart:
	docker compose -f docker-compose.prod.yml restart gisty

# View app logs only
prod-logs-app:
	docker compose -f docker-compose.prod.yml logs -f gisty

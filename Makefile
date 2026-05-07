.PHONY: help dev-up dev-down build-all test-all lint-all clean \
        user-service resume-service github-service interview-service \
        scoring-service report-service notification-service analytics-service admin-service \
	ai-service api-gateway frontend proto generate \
        docker-build docker-push docker-compose-up docker-compose-down \
        k8s-deploy k8s-rollback k8s-status \
        db-migrate db-seed db-backup

# Default target
help:
	@echo "AI Interview Platform - Makefile"
	@echo ""
	@echo "Development:"
	@echo "  make dev-up          Start development environment"
	@echo "  make dev-down        Stop development environment"
	@echo "  make build-all       Build all services"
	@echo "  make test-all        Run all tests"
	@echo "  make lint-all        Run linters on all services"
	@echo ""
	@echo "Services:"
	@echo "  make user-service        Build user service"
	@echo "  make resume-service      Build resume service"
	@echo "  make github-service      Build github service"
	@echo "  make interview-service   Build interview service"
	@echo "  make scoring-service     Build scoring service"
	@echo "  make report-service      Build report service"
	@echo "  make notification-service Build notification service"
	@echo "  make analytics-service   Build analytics service"
	@echo "  make admin-service       Build admin service"
	@echo "  make api-gateway         Build API gateway"
	@echo "  make ai-service          Build AI service"
	@echo "  make frontend            Build frontend"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make docker-build        Build all Docker images"
	@echo "  make docker-push         Push all Docker images"
	@echo "  make k8s-deploy          Deploy to Kubernetes"
	@echo "  make k8s-rollback        Rollback Kubernetes deployment"
	@echo ""
	@echo "Database:"
	@echo "  make db-migrate          Run database migrations"
	@echo "  make db-seed             Seed database with test data"
	@echo "  make db-backup           Backup database"

# Development
dev-up:
	docker compose -f infrastructure/docker/docker-compose.yml up -d
	@$(MAKE) -s dev-migrate

dev-down:
	docker compose -f infrastructure/docker/docker-compose.yml down

# Reset everything: stop, drop volumes, start fresh + migrate.
dev-reset:
	docker compose -f infrastructure/docker/docker-compose.yml down -v
	$(MAKE) dev-up

# Apply all *.up.sql migrations against their service-specific databases.
# Safe to run multiple times — the script tolerates already-applied migrations.
dev-migrate:
	./scripts/run-migrations.sh

# Promote a user to the admin role.
# Usage: make grant-admin EMAIL=user@example.com
grant-admin:
	@if [ -z "$(EMAIL)" ]; then echo "Usage: make grant-admin EMAIL=user@example.com" >&2; exit 1; fi
	@./scripts/grant-admin.sh "$(EMAIL)"

# Build all services
build-all: user-service resume-service github-service interview-service scoring-service report-service notification-service analytics-service admin-service api-gateway ai-service frontend

# Test all services
test-all:
	@echo "Running all tests..."
	cd services/user-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/resume-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/github-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/interview-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/scoring-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/report-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/notification-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/analytics-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/admin-service && go test -v -race -coverprofile=coverage.out ./...
	cd services/ai-service && python -m pytest tests/ -v --cov=src
	cd frontend && npm test -- --coverage

# Lint all services
lint-all:
	golangci-lint run ./services/user-service/...
	golangci-lint run ./services/resume-service/...
	golangci-lint run ./services/github-service/...
	golangci-lint run ./services/interview-service/...
	golangci-lint run ./services/scoring-service/...
	golangci-lint run ./services/report-service/...
	golangci-lint run ./services/notification-service/...
	golangci-lint run ./services/analytics-service/...
	golangci-lint run ./services/admin-service/...
	cd services/ai-service && ruff check src/
	cd frontend && npm run lint

# Individual Go services
user-service:
	cd services/user-service && go build -o bin/user-service ./cmd/

resume-service:
	cd services/resume-service && go build -o bin/resume-service ./cmd/

github-service:
	cd services/github-service && go build -o bin/github-service ./cmd/

interview-service:
	cd services/interview-service && go build -o bin/interview-service ./cmd/

scoring-service:
	cd services/scoring-service && go build -o bin/scoring-service ./cmd/

report-service:
	cd services/report-service && go build -o bin/report-service ./cmd/

notification-service:
	cd services/notification-service && go build -o bin/notification-service ./cmd/

analytics-service:
	cd services/analytics-service && go build -o bin/analytics-service ./cmd/

admin-service:
	cd services/admin-service && go build -o bin/admin-service ./cmd/

# Python AI service
ai-service:
	cd services/ai-service && pip install -r requirements.txt

api-gateway:
	cd services/api-gateway && go build -o bin/api-gateway ./cmd

# Frontend
frontend:
	cd frontend && npm install && npm run build

# Proto generation
proto:
	protoc --go_out=. --go-grpc_out=. shared/proto/**/*.proto

generate: proto
	@echo "Generated protobuf files"

# Docker
docker-build:
	docker compose -f infrastructure/docker/docker-compose.yml build

docker-push:
	docker compose -f infrastructure/docker/docker-compose.yml push

docker-compose-up:
	docker compose -f infrastructure/docker/docker-compose.yml up -d

docker-compose-down:
	docker compose -f infrastructure/docker/docker-compose.yml down

# Kubernetes
k8s-deploy:
	kubectl apply -f infrastructure/k8s/base/
	kubectl apply -f infrastructure/k8s/services/
	kubectl apply -f infrastructure/k8s/databases/
	kubectl apply -f infrastructure/k8s/messaging/
	kubectl apply -f infrastructure/k8s/ingress/
	kubectl apply -f infrastructure/k8s/monitoring/

k8s-rollback:
	kubectl rollout undo deployment -l app=interview-platform

k8s-status:
	kubectl get pods -n interview-platform
	kubectl get services -n interview-platform
	kubectl get ingress -n interview-platform

# Database
db-migrate:
	./infrastructure/scripts/migrate.sh

db-seed:
	./infrastructure/scripts/seed-db.sh

db-backup:
	./infrastructure/scripts/backup-db.sh

# Clean
clean:
	rm -rf services/*/bin/
	rm -rf frontend/dist/
	rm -rf frontend/node_modules/
	find . -name "*.cover" -delete
	find . -name "coverage.out" -delete

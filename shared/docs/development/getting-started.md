# Getting Started

This guide will help you set up the Real Assessment Platform for local development.

---

## Prerequisites

Before you begin, ensure you have the following installed:

### Required

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.21+ | Backend services (user, resume, interview, API gateway, admin) |
| Python | 3.11+ | Backend services (AI, scoring) |
| Node.js | 18+ | Frontend application |
| Docker | 24+ | Containerized dependencies |
| Docker Compose | 2.20+ | Local infrastructure stack |
| Git | 2.40+ | Version control |
| Make | 3.82+ | Build automation |

### Recommended

| Tool | Purpose |
|------|---------|
| VS Code | Primary editor (recommended extensions below) |
| PostgreSQL client (psql) | Direct database access |
| kafkacat | Kafka debugging |
| Redis CLI | Redis debugging |
| buf | Protobuf tooling |

### VS Code Extensions

```json
{
  "recommendations": [
    "golang.go",
    "ms-python.python",
    "ms-python.vscode-pylance",
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "zxh404.vscode-proto3",
    "hashicorp.hcl",
    "eamodio.gitlens"
  ]
}
```

---

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/real-assessment.git
cd real-assessment
```

### 2. Set Up Environment

```bash
# Copy environment template
cp shared/config/secrets/.env.example .env

# Edit with your values (development defaults are fine)
nano .env
```

### 3. Start Infrastructure

```bash
# Start PostgreSQL, Redis, Kafka, and other dependencies
make infra-up

# Wait for services to be healthy (approx. 30 seconds)
make infra-health
```

### 4. Initialize Databases

```bash
# Run database migrations
make db-migrate
```

### 5. Install Dependencies

```bash
# Go services
make go-deps

# Python services
make python-deps

# Frontend
make frontend-deps
```

### 6. Generate Protobuf

```bash
# Generate Go and Python stubs from .proto files
make proto-generate
```

### 7. Start Services

```bash
# Option A: Start all services via docker-compose
make dev-up

# Option B: Start services individually
cd services/api-gateway && go run .
cd services/ai-service && python -m app
cd frontend && npm run dev
```

### 8. Verify

Open http://localhost:3000 to access the frontend. The API is available at http://localhost:8080.

```bash
# Health check
curl http://localhost:8080/health

# Expected response
# {"status": "ok", "services": {...}}
```

---

## Project Structure

```
real-assessment/
├── services/
│   ├── api-gateway/        # Go - API Gateway
│   ├── user-service/       # Go - User management
│   ├── resume-service/     # Go - Resume processing
│   ├── interview-service/  # Go - Interview management
│   ├── ai-service/         # Python - AI analysis
│   ├── scoring-service/    # Python - Score aggregation
│   └── admin-service/      # Go - Admin operations
├── frontend/               # React/TypeScript frontend
├── shared/
│   ├── packages/
│   │   └── python-common/  # Shared Python library
│   ├── protobuf/           # Protobuf definitions
│   └── config/             # Shared configuration
├── infrastructure/         # Docker Compose, Helm charts
├── tests/                  # Integration and e2e tests
└── docs/                   # Documentation
```

---

## Common Tasks

### Database

```bash
# Run migrations
make db-migrate

# Create a new migration
make db-migrate-create name=add_user_avatar

# Reset database (WARNING: destroys all data)
make db-reset

# Open psql shell
make db-shell
```

### Testing

```bash
# Run all tests
make test

# Run tests for specific service
make test-service SERVICE=user-service

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

### Code Quality

```bash
# Lint all code
make lint

# Format all code
make fmt

# Check proto files
make proto-lint
```

### Docker

```bash
# Build all images
make docker-build

# Build single service image
make docker-build SERVICE=user-service

# Stop all containers
make infra-down

# View logs
make logs
```

---

## Troubleshooting

### PostgreSQL won't start

```bash
# Check container status
docker compose -f infrastructure/docker-compose.infra.yml ps

# View logs
docker compose -f infrastructure/docker-compose.infra.yml logs postgres

# Restart
docker compose -f infrastructure/docker-compose.infra.yml restart postgres
```

### Kafka connection refused

Kafka takes ~30 seconds to start. Wait and retry.

```bash
# Check Kafka readiness
make infra-health
```

### Go module issues

```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
```

### Python import errors

```bash
# Ensure you're using the right virtual environment
cd services/ai-service
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### Protobuf generation fails

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Regenerate
make proto-generate
```

---

## Next Steps

- Read the [Coding Standards](./coding-standards.md) guide
- Learn about our [Git Workflow](./git-workflow.md)
- Explore the [Testing Guide](./testing-guide.md)
- Check the [Architecture Docs](../architecture/system-design.md)

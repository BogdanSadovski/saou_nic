# AI Interview Platform

A comprehensive, microservices-based AI-powered interview platform for automated candidate assessment, resume analysis, GitHub profile evaluation, and intelligent interview sessions.

## Overview

This platform provides:
- **User Management** - Authentication, authorization, OAuth integration
- **Resume Analysis** - NLP-powered resume parsing and skill extraction
- **GitHub Integration** - Repository and contribution analysis
- **AI Interviews** - Real-time video interviews with AI-generated questions
- **Scoring Engine** - Multi-dimensional candidate evaluation
- **Report Generation** - PDF/DOCX candidate reports
- **Notifications** - Email, push, and SMS notifications
- **Analytics Dashboard** - Business intelligence and metrics

## Architecture

- **Backend**: Go microservices with gRPC communication
- **AI Service**: Python/FastAPI with LLM integration
- **Frontend**: React 18 + TypeScript + Vite
- **Infrastructure**: Docker, Kubernetes, RabbitMQ, Kafka

## Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd project

# Start development environment
make dev-up

# Run all tests
make test

# Build all services
make build-all
```

## Documentation

- [Getting Started](docs/development/getting-started.md)
- [Architecture](docs/architecture/system-design.md)
- [API Reference](docs/api/openapi.yaml)
- [Deployment Guide](docs/deployment/production-deployment.md)

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.21+, gRPC, PostgreSQL |
| AI Service | Python 3.12+, FastAPI, OpenAI |
| Frontend | React 18, TypeScript, Vite |
| Message Queue | RabbitMQ, Kafka |
| Cache | Redis |
| Storage | MinIO/S3 |
| Monitoring | Prometheus, Grafana, Loki |
| CI/CD | GitHub Actions |

## License

MIT License - see [LICENSE](LICENSE) for details

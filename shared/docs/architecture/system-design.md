# System Design Document

## Overview

The Real Assessment Platform is a microservices-based system designed to automate candidate evaluation through AI-powered interview analysis and resume scoring. This document captures the functional and non-functional requirements, architectural decisions, and trade-offs that shaped the system.

---

## 1. Functional Requirements

### 1.1 Core Features

| ID | Requirement | Description |
|----|------------|-------------|
| FR-1 | User Management | Register, authenticate, and manage user accounts with role-based access control (candidates, interviewers, admins). |
| FR-2 | Resume Processing | Upload, parse, and analyze resumes/CVs to extract skills, experience, education, and generate structured profiles. |
| FR-3 | Interview Management | Schedule, conduct, and record interviews (video, audio, text-based). Support structured and unstructured formats. |
| FR-4 | AI Analysis | Apply NLP models to analyze interview transcripts and resumes for competency scoring, sentiment analysis, and keyword extraction. |
| FR-5 | Scoring Engine | Aggregate scores from multiple sources (resume, interview, AI) into unified candidate evaluation reports. |
| FR-6 | Reporting | Generate detailed candidate reports with score breakdowns, strengths/weaknesses, and recommendations. |
| FR-7 | Admin Dashboard | Administrative tools for user management, system monitoring, configuration, and audit logging. |

### 1.2 Integrations

- OAuth 2.0 providers (Google, GitHub)
- OpenAI API for NLP processing
- Email service for notifications
- S3-compatible storage for file uploads

---

## 2. Non-Functional Requirements

### 2.1 Performance

| Metric | Target |
|--------|--------|
| API response time (p95) | < 200ms |
| AI scoring latency | < 5s |
| File upload size limit | 50MB |
| Concurrent users | 10,000+ |
| Throughput | 1,000 req/s per service |

### 2.2 Availability

- **Target uptime:** 99.9% (production)
- **Graceful degradation:** Services should remain functional if AI service is unavailable (queue for later processing)
- **Zero-downtime deployments:** Rolling updates with health checks

### 2.3 Scalability

- Horizontal scaling for all stateless services
- Database read replicas for query-heavy workloads
- Kafka for decoupled, scalable event processing

### 2.4 Security

- JWT-based authentication with RS256 signing
- Role-based access control (RBAC)
- TLS 1.2+ for all external communication
- Encryption at rest for sensitive data (PII)
- Input validation on all endpoints
- Rate limiting and DDoS protection

### 2.5 Observability

- Structured JSON logging across all services
- Prometheus metrics (request rates, latencies, error rates)
- Distributed tracing via Jaeger/OTEL
- Health check endpoints per service

---

## 3. Architecture Decisions

### ADR-001: Microservices Architecture

**Decision:** Use a microservices architecture with separate services for each domain.

**Rationale:**
- Independent deployment and scaling of services
- Team autonomy per service
- Technology diversity where appropriate (Go for high-throughput services, Python for AI/ML)
- Clear domain boundaries reduce coupling

**Consequences:**
- Increased operational complexity
- Requires service discovery, distributed tracing
- Network latency between services
- Distributed transactions require eventual consistency

---

### ADR-002: Go for Core Services, Python for AI Services

**Decision:** Implement the API gateway, user service, and interview service in Go. Implement the AI service and scoring service in Python.

**Rationale:**
- Go: High performance, excellent concurrency, small binaries, fast startup -- ideal for API gateways and high-throughput services
- Python: Rich ML/NLP ecosystem (OpenAI SDK, scikit-learn, NLTK), easier model integration

**Consequences:**
- Two language ecosystems to maintain
- Requires shared contracts (gRPC/Protobuf, OpenAPI specs)
- Different tooling and CI/CD pipelines

---

### ADR-003: gRPC for Inter-Service Communication

**Decision:** Use gRPC with Protocol Buffers for synchronous service-to-service calls.

**Rationale:**
- Strongly typed contracts via .proto files
- Better performance than REST (binary protocol, HTTP/2)
- Built-in code generation for Go and Python
- Streaming support for real-time data

**Consequences:**
- Requires protobuf compilation step
- Less human-readable than REST for debugging
- Need gRPC-Gateway for external-facing REST APIs

---

### ADR-004: Kafka for Event-Driven Communication

**Decision:** Use Apache Kafka for asynchronous, event-driven communication between services.

**Rationale:**
- Decouples services temporally (producer and consumer don't need to be online simultaneously)
- Supports replay and reprocessing of events
- Scales horizontally with partitions
- Durable message log prevents data loss

**Consequences:**
- Additional infrastructure to manage
- Eventual consistency model
- Requires careful idempotency handling

---

### ADR-005: PostgreSQL as Primary Data Store

**Decision:** Use PostgreSQL as the primary relational database, with one database per service.

**Rationale:**
- ACID compliance for data integrity
- Rich query capabilities and JSON support
- Mature ecosystem with excellent Go/Python drivers
- Logical replication for read replicas

**Consequences:**
- Each service manages its own schema
- Cross-service queries require API composition
- Need connection pooling at scale

---

### ADR-006: Kubernetes for Orchestration

**Decision:** Deploy services on Kubernetes with Helm charts.

**Rationale:**
- Industry-standard container orchestration
- Built-in service discovery, load balancing, autoscaling
- Declarative configuration
- Multi-environment support (dev, staging, production)

**Consequences:**
- Steep learning curve
- Requires infrastructure expertise
- Local development needs docker-compose parity

---

## 4. Trade-offs Summary

| Decision | Benefits | Costs |
|----------|----------|-------|
| Microservices | Independent scaling, team autonomy | Operational complexity, network overhead |
| Go + Python | Best tool for each domain | Two ecosystems, different tooling |
| gRPC | Performance, type safety | Debugging complexity, compilation step |
| Kafka | Decoupling, replayability | Infrastructure overhead, eventual consistency |
| PostgreSQL per service | Data isolation, autonomy | Cross-service queries harder |
| Kubernetes | Scalability, standardization | Complexity, resource overhead |

---

## 5. System Context

```
                    +----------------+
                    |    End Users   |
                    | (Candidates,   |
                    |  Interviewers) |
                    +-------+--------+
                            |
                     HTTPS / WSS
                            |
                    +-------v--------+
                    |    API Gateway |
                    |   (Go / NGINX) |
                    +-------+--------+
                            |
              +-------------+-------------+
              |             |             |
       +------v----+ +------v----+ +------v----+
       |   User    | |  Resume   | | Interview |
       |  Service  | |  Service  | |  Service  |
       |   (Go)    | |   (Go)    | |   (Go)    |
       +-----------+ +-----+-----+ +-----+-----+
                           |             |
                     +-----v-------------v-----+
                     |         Kafka           |
                     |     (Event Bus)         |
                     +-----+-------------+-----+
                           |             |
                    +------v----+ +------v----+
                    |    AI     | |  Scoring  |
                    |  Service  | |  Service  |
                    |  (Python) | | (Python)  |
                    +-----------+ +-----------+
```

---

*See also: [Microservices Overview](./microservices-overview.md) | [Data Flow](./data-flow.md) | [Deployment Architecture](./deployment-architecture.md)*

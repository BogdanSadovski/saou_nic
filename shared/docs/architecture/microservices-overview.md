# Microservices Overview

## Service Catalog

| Service | Language | Port | Responsibility |
|---------|----------|------|----------------|
| API Gateway | Go | 8080 | Routing, auth, rate limiting, request aggregation |
| User Service | Go | 8081 | User accounts, authentication, RBAC |
| Resume Service | Go | 8082 | Resume upload, parsing, skill extraction |
| Interview Service | Go | 8083 | Interview scheduling, recording, transcripts |
| AI Service | Python | 8084 | NLP analysis, scoring models, text processing |
| Scoring Service | Python | 8085 | Score aggregation, report generation |
| Admin Service | Go | 8086 | Admin operations, system config, audit logs |

---

## Service Responsibilities

### API Gateway (`api-gateway`)

The API Gateway is the single entry point for all client requests.

**Responsibilities:**
- Route incoming requests to appropriate backend services
- JWT validation and token refresh
- Rate limiting and throttling
- Request/response transformation
- CORS handling
- Request aggregation for composite responses
- Health check aggregation

**Tech Stack:** Go, Chi router, JWT middleware, Redis (rate limiting)

**External Endpoints:**
- `/api/v1/*` -- proxied to backend services
- `/auth/*` -- authentication endpoints
- `/health` -- aggregated health check

---

### User Service (`user-service`)

Manages user accounts, authentication, and authorization.

**Responsibilities:**
- User registration and profile management
- Password hashing and verification (bcrypt)
- OAuth integration (Google, GitHub)
- JWT token generation and refresh
- Role-based access control (admin, interviewer, candidate)
- Email verification workflow
- User search and listing

**Tech Stack:** Go, chi, PostgreSQL, Redis (sessions), gRPC client

**Database:** `users` schema (PostgreSQL)

**Key Tables:**
- `users` -- core user records
- `user_roles` -- role assignments
- `oauth_connections` -- third-party auth links
- `sessions` -- refresh token store

---

### Resume Service (`resume-service`)

Handles resume/CV upload, parsing, and structured data extraction.

**Responsibilities:**
- File upload handling (PDF, DOCX, TXT)
- Resume parsing and text extraction
- Skill identification and categorization
- Experience timeline extraction
- Education history parsing
- Resume-to-candidate matching
- File storage integration (S3)

**Tech Stack:** Go, PostgreSQL, S3 client, gRPC to AI Service

**Database:** `resumes` schema (PostgreSQL)

**Key Tables:**
- `resumes` -- file metadata and status
- `parsed_resumes` -- structured extraction results
- `skills` -- extracted skill entries
- `education_entries` -- education history
- `work_entries` -- employment history

---

### Interview Service (`interview-service`)

Manages the interview lifecycle from scheduling to completion.

**Responsibilities:**
- Interview creation and scheduling
- Interviewer and candidate assignment
- Interview status tracking (scheduled, in-progress, completed, cancelled)
- Recording metadata management
- Transcript storage and retrieval
- Question bank management
- Publishing interview completion events to Kafka

**Tech Stack:** Go, PostgreSQL, Kafka producer, gRPC clients

**Database:** `interviews` schema (PostgreSQL)

**Key Tables:**
- `interviews` -- interview sessions
- `interview_participants` -- participant assignments
- `interview_recordings` -- media metadata
- `interview_transcripts` -- text transcripts
- `question_banks` -- question templates
- `interview_questions` -- assigned questions

---

### AI Service (`ai-service`)

Provides AI-powered analysis of resumes and interview transcripts.

**Responsibilities:**
- Resume text analysis via NLP models
- Interview transcript analysis (sentiment, competency detection)
- Skill relevance scoring
- Candidate-requirement matching
- Keyword and phrase extraction
- Summarization of interview content
- Integration with OpenAI API (or other LLM providers)

**Tech Stack:** Python, FastAPI, OpenAI SDK, Kafka consumer/producer

**External Dependencies:**
- OpenAI API (GPT-4o)
- Internal Kafka topics for event consumption

---

### Scoring Service (`scoring-service`)

Aggregates scores from multiple sources into unified candidate evaluations.

**Responsibilities:**
- Receiving scores from AI Service via Kafka events
- Score aggregation algorithm (weighted average)
- Candidate ranking
- Threshold evaluation (pass/fail)
- Report data assembly
- Score history tracking
- Publishing score events for notifications

**Tech Stack:** Python, FastAPI, PostgreSQL, Kafka consumer

**Database:** `scoring` schema (PostgreSQL)

**Key Tables:**
- `candidate_scores` -- aggregated scores per candidate
- `score_components` -- individual score entries (resume, interview, AI)
- `score_reports` -- generated report metadata
- `scoring_rules` -- configurable weights and thresholds

---

### Admin Service (`admin-service`)

Provides administrative capabilities for platform management.

**Responsibilities:**
- System configuration management
- Feature flag management
- Audit log viewing and export
- User management (impersonation, deactivation)
- Service health monitoring
- Bulk data operations
- System statistics and dashboards

**Tech Stack:** Go, PostgreSQL, gRPC clients to all services

**Database:** `admin` schema (PostgreSQL)

**Key Tables:**
- `system_config` -- configuration entries
- `feature_flags` -- feature toggle state
- `audit_logs` -- admin action history
- `service_status` -- health check history

---

## Communication Patterns

### Synchronous (gRPC)

Used when the caller needs an immediate response.

```
Client --> API Gateway --> User Service (gRPC: validate token)
Client --> API Gateway --> Resume Service --> AI Service (gRPC: analyze)
Client --> API Gateway --> Scoring Service (gRPC: get report)
```

**Protobuf definitions:** `shared/protobuf/`

### Asynchronous (Kafka)

Used for event-driven workflows and decoupled processing.

```
Interview Service --> Kafka [interview.completed] --> AI Service
AI Service --> Kafka [analysis.completed] --> Scoring Service
Scoring Service --> Kafka [score.updated] --> Notification Service
User Service --> Kafka [user.created] --> [Subscribers]
```

**Topics:**
- `user-events.{env}` -- user lifecycle events
- `interview-events.{env}` -- interview state changes
- `scoring-events.{env}` -- score updates
- `notifications.{env}` -- notification triggers

### Request/Response (HTTP/REST)

Used for external-facing APIs through the API Gateway.

```
Browser/Client --> HTTPS --> API Gateway --> Service --> Response
```

---

## Service Dependencies

```
api-gateway
  ├── user-service (gRPC + HTTP)
  ├── resume-service (HTTP)
  ├── interview-service (HTTP)
  ├── scoring-service (HTTP)
  └── admin-service (HTTP)

user-service
  ├── PostgreSQL (own DB)
  ├── Redis (sessions)
  └── Kafka (publish user events)

resume-service
  ├── PostgreSQL (own DB)
  ├── S3 (file storage)
  └── ai-service (gRPC: analyze resume)

interview-service
  ├── PostgreSQL (own DB)
  ├── Kafka (publish interview events)
  └── ai-service (gRPC: analyze transcript)

ai-service
  ├── Kafka (consume interview/resume events)
  ├── Kafka (publish analysis results)
  └── OpenAI API (external)

scoring-service
  ├── PostgreSQL (own DB)
  └── Kafka (consume analysis results, publish scores)

admin-service
  ├── PostgreSQL (own DB)
  └── All services (gRPC: health/status queries)
```

---

*See also: [System Design](./system-design.md) | [Data Flow](./data-flow.md)*

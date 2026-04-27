# Data Flow Documentation

This document describes how data flows through the Real Assessment Platform, covering request/response patterns, event flows, and data lifecycle.

---

## 1. Request/Response Flow

### 1.1 Standard API Request Flow

```
Client (Browser/App)
    |
    |  HTTPS POST /api/v1/interviews
    |  Authorization: Bearer <token>
    v
+-------------------+
|   API Gateway     |  1. Validate JWT
|   (Go :8080)      |  2. Check rate limit
|                   |  3. Route to service
+--------+----------+
         |
         |  HTTP POST /interviews (internal)
         |  X-Request-ID: abc-123
         |  X-User-ID: user-42
         v
+-------------------+
| Interview Service |  1. Validate request body
|   (Go :8083)      |  2. Begin DB transaction
|                   |  3. Insert interview record
|                   |  4. Commit transaction
+--------+----------+
         |
         |  HTTP 201 Created
         |  { "id": "int-789", "status": "scheduled" }
         v
+-------------------+
|   API Gateway     |  Transform response, add headers
+--------+----------+
         |
         |  HTTPS 201 Created
         v
Client (Browser/App)
```

### 1.2 Composite Request (Aggregated Response)

```
Client --> GET /api/v1/candidates/{id}/report

API Gateway fans out to:
  +--> Resume Service  --> GET /resumes?user_id={id}
  +--> Interview Service --> GET /interviews?user_id={id}
  +--> Scoring Service --> GET /scores?user_id={id}

API Gateway aggregates responses and returns:
{
  "user": { ... },
  "resumes": [ ... ],
  "interviews": [ ... ],
  "scores": { ... },
  "report": { ... }
}
```

---

## 2. Event Flow

### 2.1 Interview Completion Pipeline

This is the primary event-driven pipeline where an interview completion triggers AI analysis and scoring.

```
┌─────────────────┐
│ Interview       │
│ Service         │
│                 │
│ 1. Interviewer  │
│    marks as     │
│    completed    │
│ 2. Saves final  │
│    transcript   │
│ 3. Publishes    │
│    event        │
└────────┬────────┘
         │
         │ Kafka: interview.completed
         │ {
         │   "interview_id": "int-789",
         │   "user_id": "user-42",
         │   "transcript_url": "s3://...",
         │   "metadata": { ... }
         │ }
         v
┌─────────────────┐
│     Kafka       │
│   Topic:        │
│   interview-    │
│   events.prod   │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ AI Service      │
│                 │
│ 1. Consumes     │
│    event        │
│ 2. Downloads    │
│    transcript   │
│ 3. Calls OpenAI │
│    for analysis │
│ 4. Extracts:    │
│    - sentiments │
│    - skills     │
│    - competenc. │
│    - summary    │
│ 5. Publishes    │
│    result       │
└────────┬────────┘
         │
         │ Kafka: analysis.completed
         │ {
         │   "interview_id": "int-789",
         │   "user_id": "user-42",
         │   "analysis": {
         │     "sentiment": { "positive": 0.7 },
         │     "skills_detected": ["Go", "AWS"],
         │     "competency_scores": {
         │       "communication": 85,
         │       "technical": 78
         │     },
         │     "summary": "..."
         │   }
         │ }
         v
┌─────────────────┐
│     Kafka       │
│   Topic:        │
│   scoring-      │
│   events.prod   │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ Scoring Service │
│                 │
│ 1. Consumes     │
│    analysis     │
│    event        │
│ 2. Fetches      │
│    resume score │
│    from DB      │
│ 3. Aggregates   │
│    weighted     │
│    scores       │
│ 4. Determines   │
│    pass/fail    │
│ 5. Saves report │
│ 6. Publishes    │
│    score event  │
└────────┬────────┘
         │
         │ Kafka: score.updated
         │ (consumed by notification
         │  service, admin dashboard)
         v
    [Downstream Consumers]
```

### 2.2 Resume Processing Flow

```
User uploads resume (PDF/DOCX)
    |
    v
API Gateway --> Resume Service
    |
    |  1. Validate file
    |  2. Store in S3
    |  3. Extract text
    v
Resume Service calls AI Service (gRPC)
    |
    |  AnalyzeResume(text)
    v
AI Service processes text:
    |  - Parse skills
    |  - Parse experience
    |  - Parse education
    |  - Generate profile
    v
AI Service returns structured data
    |
    v
Resume Service saves parsed data to DB
    |
    v
Resume Service publishes event (optional)
    |
    |  Kafka: resume.parsed
    v
Scoring Service updates candidate score
```

---

## 3. Data Lifecycle

### 3.1 Candidate Data Lifecycle

```
Created --> Active --> Evaluated --> Archived
   |          |           |            |
   |    Interview    Scores are     After 2
   |    scheduled,   calculated,    years, PII
   |    resumes      reports        is anonym-
   |    uploaded     generated      ized
```

**State Transitions:**

| From | To | Trigger |
|------|-----|---------|
| (none) | Created | User registration |
| Created | Active | Email verification |
| Active | Evaluated | Interview completed + scored |
| Evaluated | Archived | Data retention policy (2 years) |

### 3.2 Interview Data Lifecycle

```
Scheduled --> In Progress --> Completed --> (Archived)
     |            |              |
     |   Cancelled  |     Analysis triggered
     |              |
     |          Rescheduled
```

---

## 4. Data Ownership

Each service owns its data. Cross-service data access is only through APIs or events.

| Service | Owns | Consumes From |
|---------|------|---------------|
| User Service | Users, roles, sessions | -- |
| Resume Service | Resumes, parsed data, skills | AI Service (analysis results) |
| Interview Service | Interviews, transcripts | -- |
| AI Service | -- (stateless analysis) | Interview Service (transcripts), Resume Service (text) |
| Scoring Service | Scores, reports | AI Service (analysis), Resume Service (resume scores) |
| Admin Service | Config, audit logs | All services (health/status queries) |

---

## 5. Data Consistency Model

### 5.1 Within a Service: Strong Consistency

Each service uses PostgreSQL transactions for ACID guarantees.

```sql
BEGIN;
  INSERT INTO interviews (...) VALUES (...);
  INSERT INTO interview_participants (...) VALUES (...);
  INSERT INTO interview_questions (...) VALUES (...);
COMMIT;
```

### 5.2 Between Services: Eventual Consistency

Cross-service data consistency is achieved through Kafka events with idempotent consumers.

```
Service A: commits data --> publishes event
                                      |
                                      v
Service B: consumes event --> applies change --> commits
```

**Idempotency Strategy:**
- Each event has a unique `event_id` (UUID)
- Consumers track processed `event_id`s in a deduplication table
- Re-processing the same event_id is a no-op

### 5.3 Saga Pattern for Multi-Service Operations

For operations that span multiple services, use the Saga pattern:

```
1. Create Interview (Interview Service)
2. Notify Candidate (Notification via Kafka)
3. Assign Interviewer (User Service via gRPC)
   |
   | If step 3 fails:
   |--> Compensate: Cancel Interview
```

---

*See also: [System Design](./system-design.md) | [Microservices Overview](./microservices-overview.md)*

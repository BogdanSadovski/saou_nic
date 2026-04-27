# Interview Service API

## Overview

The Interview Service manages the complete interview lifecycle: creation, scheduling, conducting, and completion. It publishes events to Kafka when interviews transition to completed status, triggering downstream AI analysis.

**Base URL:** `/api/v1`
**Tech Stack:** Go, PostgreSQL, Kafka producer, gRPC to AI Service

---

## Endpoints

### Interview Management

#### POST /interviews

Create a new interview session.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
  "interviewer_id": "660e8400-e29b-41d4-a716-446655440000",
  "scheduled_at": "2026-04-15T14:00:00Z",
  "questions": [
    "Describe your experience with distributed systems.",
    "How do you handle production incidents?",
    "Walk me through your approach to system design."
  ],
  "notes": "Technical interview - backend focus",
  "duration_minutes": 60
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `candidate_id` | string (UUID) | Yes | Candidate user ID |
| `interviewer_id` | string (UUID) | Yes | Interviewer user ID |
| `scheduled_at` | string (ISO 8601) | Yes | Interview start time |
| `questions` | string[] | No | Pre-defined questions |
| `notes` | string | No | Additional notes |
| `duration_minutes` | int | No | Expected duration (default: 60) |

**Response (201 Created):**

```json
{
  "id": "int-789abc-def0-1234-5678-90abcdef1234",
  "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
  "interviewer_id": "660e8400-e29b-41d4-a716-446655440000",
  "status": "scheduled",
  "scheduled_at": "2026-04-15T14:00:00Z",
  "questions": [
    "Describe your experience with distributed systems.",
    "How do you handle production incidents?",
    "Walk me through your approach to system design."
  ],
  "notes": "Technical interview - backend focus",
  "duration_minutes": 60,
  "created_at": "2026-04-07T10:00:00Z"
}
```

---

#### GET /interviews

List interviews with optional filters.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `candidate_id` | string | - | Filter by candidate |
| `interviewer_id` | string | - | Filter by interviewer |
| `status` | string | - | Filter by status |

**Status Filters:** `scheduled`, `in_progress`, `completed`, `cancelled`

**Response (200 OK):**

```json
{
  "items": [
    {
      "id": "int-789abc-def0-1234-5678-90abcdef1234",
      "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
      "interviewer_id": "660e8400-e29b-41d4-a716-446655440000",
      "status": "scheduled",
      "scheduled_at": "2026-04-15T14:00:00Z",
      "duration_minutes": 60,
      "created_at": "2026-04-07T10:00:00Z"
    }
  ],
  "total": 25,
  "page": 1,
  "limit": 20
}
```

---

#### GET /interviews/{interview_id}

Get detailed interview information including transcript.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "id": "int-789abc-def0-1234-5678-90abcdef1234",
  "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
  "interviewer_id": "660e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "scheduled_at": "2026-04-15T14:00:00Z",
  "started_at": "2026-04-15T14:02:00Z",
  "completed_at": "2026-04-15T15:05:00Z",
  "questions": [
    "Describe your experience with distributed systems.",
    "How do you handle production incidents?"
  ],
  "transcript": "Interviewer: Can you tell me about your experience...\n\nCandidate: Sure, I've been working with...\n\n[...]",
  "interviewer_notes": "Strong candidate, good communication skills",
  "interviewer_rating": 4,
  "duration_minutes": 60,
  "created_at": "2026-04-07T10:00:00Z"
}
```

---

#### PUT /interviews/{interview_id}

Update an existing interview.

**Headers:** `Authorization: Bearer <token>`

**Request Body:** (any subset of creation fields)

```json
{
  "scheduled_at": "2026-04-16T14:00:00Z",
  "notes": "Rescheduled - candidate requested time change"
}
```

**Response (200 OK):** Updated interview object.

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `INVALID_TRANSITION` | Cannot update completed interview |
| 404 | `INTERVIEW_NOT_FOUND` | Interview does not exist |

---

#### POST /interviews/{interview_id}/start

Mark an interview as in progress.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "id": "int-789abc-def0-1234-5678-90abcdef1234",
  "status": "in_progress",
  "started_at": "2026-04-15T14:02:00Z"
}
```

---

#### POST /interviews/{interview_id}/cancel

Cancel a scheduled interview.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "reason": "Candidate requested reschedule"
}
```

**Response (200 OK):**

```json
{
  "id": "int-789abc-def0-1234-5678-90abcdef1234",
  "status": "cancelled",
  "cancelled_at": "2026-04-14T10:00:00Z"
}
```

---

#### POST /interviews/{interview_id}/complete

Mark interview as complete and submit the transcript. This triggers the AI analysis pipeline via Kafka.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "transcript": "Interviewer: Can you describe your experience with microservices?\n\nCandidate: I have 3 years of experience building microservices in Go...\n\n[Full transcript text]",
  "interviewer_notes": "Excellent technical knowledge, clear communication",
  "interviewer_rating": 5
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `transcript` | string | Yes | Full interview transcript text |
| `interviewer_notes` | string | No | Interviewer's subjective notes |
| `interviewer_rating` | int | No | Rating 1-5 |

**Response (200 OK):**

```json
{
  "id": "int-789abc-def0-1234-5678-90abcdef1234",
  "status": "completed",
  "completed_at": "2026-04-15T15:05:00Z",
  "analysis_status": "queued"
}
```

**Note:** AI analysis runs asynchronously. Check the Scoring Service for analysis results once complete.

---

### Question Banks

#### GET /interviews/questions

List available question templates.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `category` | string | - | Filter by category |

**Response (200 OK):**

```json
{
  "questions": [
    {
      "id": "q-001",
      "text": "Describe your experience with distributed systems.",
      "category": "technical",
      "difficulty": "senior"
    },
    {
      "id": "q-002",
      "text": "Tell me about a time you resolved a conflict in your team.",
      "category": "behavioral",
      "difficulty": "all"
    }
  ]
}
```

---

## gRPC Interface

| Method | Description |
|--------|-------------|
| `GetInterview` | Retrieve interview by ID |
| `CreateInterview` | Create new interview |
| `ListInterviews` | List interviews with filters |
| `GetTranscript` | Retrieve interview transcript |
| `CompleteInterview` | Mark interview complete |

## Kafka Events

The Interview Service publishes the following events:

| Topic | Event | Triggered When |
|-------|-------|----------------|
| `interview-events.{env}` | `interview.created` | New interview created |
| `interview-events.{env}` | `interview.started` | Interview marked as in-progress |
| `interview-events.{env}` | `interview.completed` | Interview completed (triggers AI analysis) |
| `interview-events.{env}` | `interview.cancelled` | Interview cancelled |

---

*See also: [OpenAPI Spec](./openapi.yaml) | [gRPC API](./grpc-api.md) | [Scoring Service API](./scoring-service-api.md)*

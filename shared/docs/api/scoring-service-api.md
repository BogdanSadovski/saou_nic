# Scoring Service API

## Overview

The Scoring Service aggregates scores from multiple evaluation sources (resume analysis, interview analysis, manual ratings) into unified candidate scores and generates comprehensive evaluation reports.

**Base URL:** `/api/v1/scores`
**Tech Stack:** Python, FastAPI, PostgreSQL, Kafka consumer

---

## Endpoints

### Candidate Scores

#### GET /scores/{candidate_id}

Get a candidate's aggregated scores from all evaluation sources.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
  "candidate_name": "John Doe",
  "overall_score": 81,
  "status": "evaluated",
  "components": [
    {
      "source": "resume",
      "score": 75,
      "weight": 0.30,
      "weighted_score": 22.5,
      "analyzed_at": "2026-04-07T10:30:00Z"
    },
    {
      "source": "interview",
      "score": 85,
      "weight": 0.50,
      "weighted_score": 42.5,
      "analyzed_at": "2026-04-15T15:30:00Z"
    },
    {
      "source": "ai_analysis",
      "score": 78,
      "weight": 0.20,
      "weighted_score": 15.6,
      "analyzed_at": "2026-04-15T15:35:00Z"
    }
  ],
  "pass_fail": true,
  "passing_threshold": 70,
  "last_updated": "2026-04-15T15:35:00Z",
  "created_at": "2026-04-07T10:30:00Z"
}
```

**Scoring Formula:**

```
overall_score = SUM(component_score * component_weight)
```

Default weights: Resume (30%), Interview (50%), AI Analysis (20%).

---

### Reports

#### GET /scores/{candidate_id}/report

Generate a comprehensive candidate evaluation report.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
  "candidate_name": "John Doe",
  "candidate_email": "john@example.com",
  "overall_score": 81,
  "recommendation": "hire",
  "percentile": 85,
  "sections": [
    {
      "title": "Resume Analysis",
      "score": 75,
      "summary": "Strong technical background with 4+ years of relevant experience",
      "strengths": [
        "Modern technology stack (Go, Kubernetes)",
        "Progressive career growth",
        "Relevant education background"
      ],
      "weaknesses": [
        "Limited leadership experience mentioned",
        "No open source contributions visible"
      ]
    },
    {
      "title": "Interview Performance",
      "score": 85,
      "summary": "Demonstrated strong technical knowledge and excellent communication",
      "strengths": [
        "Deep distributed systems understanding",
        "Clear, structured communication",
        "Practical production experience"
      ],
      "weaknesses": [
        "Could expand on system design methodology"
      ]
    },
    {
      "title": "AI Assessment",
      "score": 78,
      "summary": "AI analysis confirms strong backend engineering profile",
      "strengths": [
        "Consistent skill signals across resume and interview",
        "Positive sentiment throughout interview"
      ],
      "weaknesses": [
        "Limited evidence of mentoring capability"
      ]
    }
  ],
  "interview_history": [
    {
      "interview_id": "int-789abc",
      "date": "2026-04-15",
      "interviewer": "Jane Smith",
      "rating": 4,
      "status": "completed"
    }
  ],
  "generated_at": "2026-04-15T16:00:00Z"
}
```

**Recommendation Values:**

| Value | Score Range | Meaning |
|-------|------------|---------|
| `strong_hire` | 90-100 | Top candidate, strongly recommend hiring |
| `hire` | 75-89 | Good fit, recommend hiring |
| `consider` | 60-74 | Potential candidate, needs further evaluation |
| `reject` | 0-59 | Does not meet requirements |

---

#### POST /scores/{candidate_id}/report/regenerate

Force regeneration of a candidate's report.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):** Regenerated report.

---

### Score History

#### GET /scores/{candidate_id}/history

Get the score change history for a candidate.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
  "history": [
    {
      "timestamp": "2026-04-07T10:30:00Z",
      "event": "resume_score_added",
      "overall_score": 22.5,
      "component": {
        "source": "resume",
        "score": 75
      }
    },
    {
      "timestamp": "2026-04-15T15:30:00Z",
      "event": "interview_score_added",
      "overall_score": 65.0,
      "component": {
        "source": "interview",
        "score": 85
      }
    },
    {
      "timestamp": "2026-04-15T15:35:00Z",
      "event": "ai_score_added",
      "overall_score": 81.0,
      "component": {
        "source": "ai_analysis",
        "score": 78
      }
    }
  ]
}
```

---

### Score Configuration

#### GET /scores/config

Get current scoring configuration (weights, thresholds).

**Headers:** `Authorization: Bearer <token>` (admin only)

**Response (200 OK):**

```json
{
  "weights": {
    "resume": 0.30,
    "interview": 0.50,
    "ai_analysis": 0.20
  },
  "passing_threshold": 70,
  "recommendation_thresholds": {
    "strong_hire": 90,
    "hire": 75,
    "consider": 60
  }
}
```

#### PUT /scores/config

Update scoring configuration (admin only).

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "weights": {
    "resume": 0.25,
    "interview": 0.55,
    "ai_analysis": 0.20
  },
  "passing_threshold": 75
}
```

**Response (200 OK):** Updated configuration.

---

## gRPC Interface

| Method | Description |
|--------|-------------|
| `GetScores` | Get aggregated scores for candidate |
| `SubmitScore` | Submit a score component |
| `GetReport` | Generate comprehensive report |
| `ListRankings` | Get candidate rankings |

---

## Kafka Events

### Consumed Events

| Topic | Event | Action |
|-------|-------|--------|
| `scoring-events.{env}` | `analysis.completed` | Add AI score component, recalculate overall |
| `interview-events.{env}` | `interview.completed` | Prepare for incoming AI analysis |

### Published Events

| Topic | Event | Consumers |
|-------|-------|-----------|
| `scoring-events.{env}` | `score.updated` | Admin dashboard, notifications |
| `scoring-events.{env}` | `report.generated` | Admin service, email notifications |

---

*See also: [OpenAPI Spec](./openapi.yaml) | [gRPC API](./grpc-api.md) | [Admin Service API](./admin-service-api.md)*

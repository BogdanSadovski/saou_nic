# Admin Service API

## Overview

The Admin Service provides administrative capabilities for platform management, system configuration, monitoring, and audit logging. All endpoints require admin role.

**Base URL:** `/api/v1/admin`
**Tech Stack:** Go, PostgreSQL, gRPC clients to all services

---

## Endpoints

### System Statistics

#### GET /admin/system/stats

Get system-wide statistics and health status.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Response (200 OK):**

```json
{
  "total_users": 1523,
  "active_users_30d": 892,
  "total_interviews": 3456,
  "active_interviews": 45,
  "completed_interviews": 3200,
  "total_resumes": 1890,
  "total_reports_generated": 2980,
  "avg_overall_score": 73.5,
  "service_health": {
    "api_gateway": {
      "status": "healthy",
      "uptime_hours": 168,
      "requests_per_second": 250,
      "error_rate": 0.002
    },
    "user_service": {
      "status": "healthy",
      "uptime_hours": 168,
      "db_connections_active": 12,
      "db_connections_max": 30
    },
    "resume_service": {
      "status": "healthy",
      "uptime_hours": 168,
      "pending_parses": 3
    },
    "interview_service": {
      "status": "healthy",
      "uptime_hours": 168,
      "active_interviews": 45
    },
    "ai_service": {
      "status": "healthy",
      "uptime_hours": 168,
      "queue_depth": 5,
      "avg_processing_time_ms": 3200
    },
    "scoring_service": {
      "status": "healthy",
      "uptime_hours": 168,
      "kafka_consumer_lag": 0
    }
  },
  "timestamp": "2026-04-07T14:00:00Z"
}
```

---

### Candidate Management

#### GET /admin/reports/candidates

Get a ranked list of all evaluated candidates.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `sort_by` | string | `overall_score` | `overall_score`, `interview_score`, `resume_score`, `name` |
| `order` | string | `desc` | `asc`, `desc` |
| `status` | string | - | `evaluated`, `partial`, `not_started` |
| `search` | string | - | Search by name or email |

**Response (200 OK):**

```json
{
  "items": [
    {
      "rank": 1,
      "candidate_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "John Doe",
      "email": "john@example.com",
      "overall_score": 91,
      "resume_score": 85,
      "interview_score": 95,
      "ai_score": 88,
      "status": "evaluated",
      "recommendation": "strong_hire",
      "interviews_completed": 2,
      "last_evaluated": "2026-04-15T16:00:00Z"
    },
    {
      "rank": 2,
      "candidate_id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "Jane Smith",
      "email": "jane@example.com",
      "overall_score": 84,
      "resume_score": 90,
      "interview_score": 80,
      "ai_score": 82,
      "status": "evaluated",
      "recommendation": "hire",
      "interviews_completed": 1,
      "last_evaluated": "2026-04-14T10:00:00Z"
    }
  ],
  "total": 250,
  "page": 1,
  "limit": 20
}
```

---

### User Management

#### GET /admin/users

List all users with admin-level details.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page |
| `role` | string | - | Filter by role |
| `status` | string | - | `active`, `disabled`, `pending_verification` |
| `search` | string | - | Search by name or email |

**Response (200 OK):**

```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "candidate",
      "status": "active",
      "email_verified": true,
      "last_login": "2026-04-07T09:00:00Z",
      "created_at": "2026-01-15T10:00:00Z"
    }
  ],
  "total": 1523,
  "page": 1,
  "limit": 20
}
```

---

#### PUT /admin/users/{user_id}

Update user details including role and status.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Request Body:**

```json
{
  "role": "interviewer",
  "status": "active"
}
```

**Response (200 OK):** Updated user object.

---

#### DELETE /admin/users/{user_id}

Disable a user account (soft delete).

**Headers:** `Authorization: Bearer <token>` (admin only)

**Request Body:**

```json
{
  "reason": "Account violated terms of service"
}
```

**Response (204 No Content)**

---

### Configuration

#### GET /admin/config

Get current system configuration.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Response (200 OK):**

```json
{
  "scoring": {
    "weights": {
      "resume": 0.30,
      "interview": 0.50,
      "ai_analysis": 0.20
    },
    "passing_threshold": 70
  },
  "interviews": {
    "default_duration_minutes": 60,
    "max_duration_minutes": 120,
    "allow_rescheduling": true
  },
  "resumes": {
    "max_file_size_mb": 10,
    "allowed_formats": ["pdf", "docx", "txt"]
  },
  "ai": {
    "provider": "openai",
    "model": "gpt-4o"
  }
}
```

---

#### PUT /admin/config

Update system configuration.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Request Body:**

```json
{
  "scoring": {
    "passing_threshold": 75
  }
}
```

**Response (200 OK):** Updated configuration.

---

### Feature Flags

#### GET /admin/feature-flags

List all feature flags and their current state.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Response (200 OK):**

```json
{
  "flags": [
    {
      "name": "ai_scoring",
      "enabled": true,
      "description": "AI-powered candidate scoring"
    },
    {
      "name": "oauth_google",
      "enabled": true,
      "description": "Google OAuth login"
    },
    {
      "name": "oauth_github",
      "enabled": true,
      "description": "GitHub OAuth login"
    },
    {
      "name": "email_verification",
      "enabled": true,
      "description": "Require email verification on signup"
    },
    {
      "name": "rate_limiting",
      "enabled": true,
      "description": "API rate limiting"
    },
    {
      "name": "beta_dashboard",
      "enabled": false,
      "description": "New analytics dashboard (beta)"
    }
  ]
}
```

---

#### PUT /admin/feature-flags/{flag_name}

Toggle a feature flag.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Request Body:**

```json
{
  "enabled": true
}
```

**Response (200 OK):**

```json
{
  "name": "beta_dashboard",
  "enabled": true,
  "updated_at": "2026-04-07T14:00:00Z"
}
```

---

### Audit Logs

#### GET /admin/audit-logs

Retrieve audit log entries.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 50 | Items per page (max 200) |
| `user_id` | string | - | Filter by actor user ID |
| `action` | string | - | Filter by action type |
| `from` | string | - | Start date (ISO 8601) |
| `to` | string | - | End date (ISO 8601) |

**Response (200 OK):**

```json
{
  "items": [
    {
      "id": "audit-001",
      "timestamp": "2026-04-07T14:00:00Z",
      "actor": {
        "user_id": "admin-uuid",
        "email": "admin@example.com",
        "role": "admin"
      },
      "action": "user.role_changed",
      "resource": {
        "type": "user",
        "id": "550e8400-e29b-41d4-a716-446655440000"
      },
      "details": {
        "old_role": "candidate",
        "new_role": "interviewer"
      },
      "ip_address": "192.168.1.100"
    }
  ],
  "total": 15000,
  "page": 1,
  "limit": 50
}
```

**Common Actions:**

| Action | Description |
|--------|-------------|
| `user.created` | New user registered |
| `user.role_changed` | User role modified by admin |
| `user.disabled` | User account disabled |
| `interview.created` | Interview session created |
| `interview.completed` | Interview marked complete |
| `score.updated` | Candidate score recalculated |
| `config.changed` | System configuration updated |
| `feature_flag.toggled` | Feature flag changed |

---

### Bulk Operations

#### POST /admin/bulk/export

Export data for a date range.

**Headers:** `Authorization: Bearer <token>` (admin only)

**Request Body:**

```json
{
  "type": "candidates",
  "from": "2026-01-01T00:00:00Z",
  "to": "2026-04-07T23:59:59Z",
  "format": "csv"
}
```

**Response (202 Accepted):**

```json
{
  "export_id": "export-001",
  "status": "processing",
  "estimated_completion": "2026-04-07T14:05:00Z"
}
```

---

#### GET /admin/bulk/export/{export_id}

Check export status and download when ready.

**Response (200 OK):**

```json
{
  "export_id": "export-001",
  "status": "completed",
  "download_url": "https://cdn.example.com/exports/export-001.csv",
  "expires_at": "2026-04-14T14:00:00Z",
  "record_count": 250,
  "file_size_bytes": 1048576
}
```

---

*See also: [OpenAPI Spec](./openapi.yaml) | [gRPC API](./grpc-api.md) | [Scoring Service API](./scoring-service-api.md)*

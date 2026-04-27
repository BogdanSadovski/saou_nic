# Resume Service API

## Overview

The Resume Service handles resume/CV upload, parsing, skill extraction, and structured data storage. It integrates with the AI Service for NLP-based text analysis.

**Base URL:** `/api/v1`
**Tech Stack:** Go, PostgreSQL, S3, gRPC to AI Service

---

## Endpoints

### Upload & Management

#### POST /resumes

Upload a resume file for parsing and analysis.

**Headers:** `Authorization: Bearer <token>`
**Content-Type:** `multipart/form-data`

**Request:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | file | Yes | Resume file (PDF, DOCX, TXT) |
| `user_id` | string | No | Associate with specific user (admin only) |

**Supported Formats:**
- PDF (`.pdf`)
- Word Document (`.docx`)
- Plain Text (`.txt`)

**File Limits:**
- Maximum size: 10 MB
- Maximum pages: 10

**Response (201 Created):**

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "john_doe_resume.pdf",
  "file_size": 245678,
  "content_type": "application/pdf",
  "status": "pending",
  "created_at": "2026-04-07T10:00:00Z"
}
```

**Status Values:**

| Status | Description |
|--------|-------------|
| `pending` | File uploaded, awaiting parsing |
| `parsing` | Currently being processed |
| `parsed` | Parsing complete, data available |
| `error` | Parsing failed |

---

#### GET /resumes

List resumes with optional filters.

**Headers:** `Authorization: Bearer <token>`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max 100) |
| `user_id` | string | - | Filter by user |
| `status` | string | - | Filter by parsing status |

**Response (200 OK):**

```json
{
  "items": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "filename": "john_doe_resume.pdf",
      "file_size": 245678,
      "content_type": "application/pdf",
      "status": "parsed",
      "created_at": "2026-04-07T10:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "limit": 20
}
```

---

#### GET /resumes/{resume_id}

Get resume details including parsed data (if available).

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "john_doe_resume.pdf",
  "status": "parsed",
  "parsed_data": {
    "skills": [
      {
        "name": "Go",
        "category": "programming_language",
        "proficiency": "advanced",
        "years_experience": 4
      },
      {
        "name": "PostgreSQL",
        "category": "database",
        "proficiency": "intermediate",
        "years_experience": 3
      },
      {
        "name": "Kubernetes",
        "category": "devops",
        "proficiency": "intermediate",
        "years_experience": 2
      }
    ],
    "experience": [
      {
        "company": "Tech Corp",
        "title": "Senior Software Engineer",
        "start_date": "2022-01",
        "end_date": null,
        "description": "Lead backend development for microservices platform"
      },
      {
        "company": "StartupXYZ",
        "title": "Software Engineer",
        "start_date": "2019-06",
        "end_date": "2021-12",
        "description": "Built REST APIs and data pipelines"
      }
    ],
    "education": [
      {
        "institution": "State University",
        "degree": "Bachelor of Science",
        "field": "Computer Science",
        "graduation_year": 2019
      }
    ],
    "summary": "Experienced backend engineer with strong Go and distributed systems background"
  },
  "created_at": "2026-04-07T10:00:00Z"
}
```

---

#### DELETE /resumes/{resume_id}

Delete a resume and its parsed data.

**Headers:** `Authorization: Bearer <token>`

**Response (204 No Content)**

---

### Analysis

#### POST /resumes/{resume_id}/analyze

Trigger AI-powered analysis of a resume. If the resume is already parsed, returns cached results.

**Headers:** `Authorization: Bearer <token>`

**Response (200 OK):**

```json
{
  "resume_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "analysis": {
    "skill_summary": "Strong backend engineering profile with modern cloud-native skills",
    "experience_level": "senior",
    "years_total_experience": 6,
    "strengths": [
      "Strong Go programming expertise",
      "Microservices architecture experience",
      "Cloud-native technology stack"
    ],
    "weaknesses": [
      "Limited frontend experience",
      "No machine learning background indicated"
    ],
    "overall_score": 82,
    "recommended_roles": [
      "Senior Backend Engineer",
      "Platform Engineer",
      "DevOps Engineer"
    ]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `INVALID_FILE` | File format not supported |
| 404 | `RESUME_NOT_FOUND` | Resume does not exist |
| 500 | `ANALYSIS_FAILED` | AI service returned an error |

---

## gRPC Interface

| Method | Description |
|--------|-------------|
| `GetResume` | Retrieve resume by ID |
| `GetResumeByUserId` | Get latest resume for a user |
| `ListUserResumes` | List all resumes for a user |
| `GetParsedData` | Get structured parsing results |

See [gRPC API Documentation](./grpc-api.md) for protobuf definitions.

---

*See also: [OpenAPI Spec](./openapi.yaml) | [gRPC API](./grpc-api.md) | [AI Service API](./ai-service-api.md)*

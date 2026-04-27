# AI Service API

## Overview

The AI Service provides NLP-powered analysis of resumes and interview transcripts. It integrates with OpenAI's API (or other LLM providers) and exposes both a REST API and gRPC interface.

**Base URL:** `/api/v1/ai`
**Tech Stack:** Python, FastAPI, OpenAI SDK, Kafka consumer/producer

---

## REST Endpoints

### Resume Analysis

#### POST /ai/resumes/analyze

Analyze resume text and extract structured information.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "text": "John Doe\nSoftware Engineer\n\nExperience:\n- Senior Engineer at TechCorp (2022-Present)\n  Built microservices in Go...\n\nEducation:\n- BS Computer Science, State University (2019)\n\nSkills: Go, PostgreSQL, Kubernetes, AWS",
  "extract_skills": true,
  "extract_experience": true,
  "extract_education": true,
  "generate_summary": true
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `text` | string | Required | Raw resume text |
| `extract_skills` | bool | true | Extract and categorize skills |
| `extract_experience` | bool | true | Extract work history |
| `extract_education` | bool | true | Extract education history |
| `generate_summary` | bool | true | Generate natural language summary |

**Response (200 OK):**

```json
{
  "skills": [
    {
      "name": "Go",
      "category": "programming_language",
      "proficiency": "advanced",
      "confidence": 0.95
    },
    {
      "name": "PostgreSQL",
      "category": "database",
      "proficiency": "intermediate",
      "confidence": 0.88
    },
    {
      "name": "Kubernetes",
      "category": "devops",
      "proficiency": "intermediate",
      "confidence": 0.85
    },
    {
      "name": "AWS",
      "category": "cloud_platform",
      "proficiency": "intermediate",
      "confidence": 0.80
    }
  ],
  "experience": [
    {
      "company": "TechCorp",
      "title": "Senior Engineer",
      "start_date": "2022-01",
      "end_date": null,
      "is_current": true,
      "description": "Built microservices in Go"
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
  "summary": "Experienced backend engineer with 4+ years of professional experience. Strong proficiency in Go programming with demonstrated expertise in building microservices architectures. Familiar with modern cloud-native technologies including Kubernetes and AWS.",
  "experience_level": "senior",
  "total_years_experience": 4,
  "processing_time_ms": 1250
}
```

---

### Interview Transcript Analysis

#### POST /ai/interviews/analyze

Analyze an interview transcript for competency scoring.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "transcript": "Interviewer: Can you describe your experience with distributed systems?\n\nCandidate: I've been working with microservices for 3 years...",
  "interview_type": "technical",
  "analyze_sentiment": true,
  "detect_competencies": true,
  "generate_feedback": true
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `transcript` | string | Required | Full interview transcript |
| `interview_type` | string | "general" | `technical`, `behavioral`, `general` |
| `analyze_sentiment` | bool | true | Sentiment analysis of responses |
| `detect_competencies` | bool | true | Detect and score competencies |
| `generate_feedback` | bool | true | Generate feedback for candidate |

**Response (200 OK):**

```json
{
  "overall_score": 78,
  "competency_scores": {
    "technical_knowledge": {
      "score": 85,
      "evidence": ["Demonstrated deep understanding of microservices", "Provided concrete examples of production issues"]
    },
    "communication": {
      "score": 80,
      "evidence": ["Articulate explanations", "Structured responses well"]
    },
    "problem_solving": {
      "score": 75,
      "evidence": ["Methodical approach to debugging", "Considered trade-offs"]
    },
    "cultural_fit": {
      "score": 70,
      "evidence": ["Collaborative mindset", "Values team communication"]
    }
  },
  "sentiment": {
    "overall": "positive",
    "confidence_score": 0.85,
    "stress_indicators": false
  },
  "strengths": [
    "Strong distributed systems knowledge",
    "Clear communication style",
    "Practical production experience"
  ],
  "areas_for_improvement": [
    "Could elaborate more on system design trade-offs",
    "Limited discussion of mentoring experience"
  ],
  "feedback": "Candidate demonstrates solid technical capabilities with practical experience in distributed systems. Communication is clear and structured. Recommend for senior backend engineering roles.",
  "red_flags": [],
  "processing_time_ms": 3500
}
```

---

### Skill Extraction

#### POST /ai/skills/extract

Extract skills from arbitrary text.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "text": "Built REST APIs in Go with PostgreSQL, deployed on Kubernetes clusters in AWS. Implemented CI/CD pipelines using GitHub Actions."
}
```

**Response (200 OK):**

```json
{
  "skills": [
    { "name": "Go", "category": "programming_language", "confidence": 0.95 },
    { "name": "PostgreSQL", "category": "database", "confidence": 0.90 },
    { "name": "Kubernetes", "category": "devops", "confidence": 0.88 },
    { "name": "AWS", "category": "cloud_platform", "confidence": 0.85 },
    { "name": "REST API", "category": "architecture", "confidence": 0.92 },
    { "name": "CI/CD", "category": "devops", "confidence": 0.87 },
    { "name": "GitHub Actions", "category": "devops", "confidence": 0.83 }
  ]
}
```

---

### Text Summarization

#### POST /ai/summarize

Generate a summary of text content.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**

```json
{
  "text": "[Long text content to summarize]",
  "max_length": 200,
  "style": "bullet_points"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `text` | string | Required | Text to summarize |
| `max_length` | int | 200 | Maximum summary length (words) |
| `style` | string | "paragraph" | `paragraph`, `bullet_points`, `executive` |

**Response (200 OK):**

```json
{
  "summary": "- 4+ years backend engineering experience\n- Primary stack: Go, PostgreSQL, Kubernetes\n- Current role: Senior Engineer at TechCorp\n- Strong distributed systems background\n- Experience with AWS cloud platform",
  "original_length": 1500,
  "summary_length": 48
}
```

---

## gRPC Interface

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `AnalyzeResume` | `AnalyzeResumeRequest` | `AnalyzeResumeResponse` | Full resume analysis |
| `AnalyzeTranscript` | `AnalyzeTranscriptRequest` | `AnalyzeTranscriptResponse` | Interview transcript analysis |
| `ExtractSkills` | `ExtractSkillsRequest` | `ExtractSkillsResponse` | Skill extraction from text |
| `GenerateSummary` | `GenerateSummaryRequest` | `GenerateSummaryResponse` | Text summarization |
| `MatchCandidate` | `MatchCandidateRequest` | `MatchCandidateResponse` | Candidate-job matching |

---

## Kafka Events

### Consumed Events

| Topic | Event | Action |
|-------|-------|--------|
| `interview-events.{env}` | `interview.completed` | Analyze the attached transcript |
| `resume-events.{env}` | `resume.uploaded` | Parse and analyze the resume |

### Published Events

| Topic | Event | Content |
|-------|-------|---------|
| `scoring-events.{env}` | `analysis.completed` | Analysis results with scores |
| `scoring-events.{env}` | `analysis.failed` | Error details if analysis failed |

---

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `AI_PROVIDER` | `openai` | LLM provider (`openai`, `anthropic`, `local`) |
| `OPENAI_API_KEY` | - | OpenAI API key |
| `AI_MODEL` | `gpt-4o` | Model to use for analysis |
| `AI_MAX_TOKENS` | `4096` | Maximum tokens per request |
| `AI_TEMPERATURE` | `0.3` | Sampling temperature (0.0-1.0) |
| `AI_TIMEOUT_SECONDS` | `60` | Request timeout |
| `AI_MOCK_RESPONSES` | `false` | Return mock data (dev only) |

---

*See also: [OpenAPI Spec](./openapi.yaml) | [gRPC API](./grpc-api.md) | [Resume Service API](./resume-service-api.md)*

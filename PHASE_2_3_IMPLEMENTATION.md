# Phase 2+3: Complete Implementation Roadmap

## Status Summary

✅ **Phase 1 COMPLETE:**
- [x] Live Code Executor Service (microservice with Python/JS execution)
- [x] Code submission handler integration (API handlers + repository)
- [ ] Frontend code editor component (in progress)

## Phase 2: Advanced Features (Interviews & Recording)

### Feature #3: Multi-Interviewer Collaboration 👥

**Priority:** HIGH | **Effort:** 3 days | **Impact:** +200% hiring accuracy

#### Database Schema Changes
```sql
-- Add collaboration support
ALTER TABLE interview_sessions ADD COLUMN IF NOT EXISTS
    collaborators JSONB DEFAULT '[]'::jsonb;

-- Track real-time edits
CREATE TABLE IF NOT EXISTS collaboration_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id),
    interviewer_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Independent scoring
CREATE TABLE IF NOT EXISTS interviewer_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id),
    interviewer_id UUID NOT NULL,
    technical_score INT,
    communication_score INT,
    problem_solving_score INT,
    culture_fit_score INT,
    notes TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE
);
```

#### Implementation Tasks
- [ ] Add collaborator management endpoints
- [ ] WebSocket sync for real-time notes
- [ ] Independent scoring endpoints
- [ ] Consensus scoring calculation
- [ ] Scoring summary view for interviewers

#### Files to Create/Update
- services/interview-service/internal/migration/005_collaboration_schema.up.sql
- services/interview-service/internal/repository/collaboration_queries.go
- services/interview-service/internal/api/collaboration_handlers.go

---

### Feature #4: Interview Templates & Question Bank 📋

**Priority:** HIGH | **Effort:** 4 days | **Impact:** +500% template creation speed

#### Database Schema
```sql
CREATE TABLE IF NOT EXISTS question_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    role VARCHAR(120) NOT NULL,
    level VARCHAR(20) NOT NULL,
    category VARCHAR(100),
    created_by UUID NOT NULL,
    published BOOLEAN DEFAULT FALSE,
    usage_count INT DEFAULT 0,
    effectiveness_score NUMERIC(3,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS template_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES question_templates(id),
    question_text TEXT NOT NULL,
    expected_keywords JSON,
    difficulty INT,
    estimated_time_minutes INT,
    sequence INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS question_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL,
    session_id UUID NOT NULL,
    rating INT,
    difficulty_was_accurate BOOLEAN,
    time_was_accurate BOOLEAN,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Implementation Tasks
- [ ] Template CRUD endpoints
- [ ] Question bank management
- [ ] Template search with filters
- [ ] Clone session from template
- [ ] Effectiveness analytics
- [ ] Admin approval workflow

#### Files to Create
- services/interview-service/internal/migration/006_question_templates.up.sql
- services/interview-service/internal/repository/template_queries.go
- services/interview-service/internal/api/template_handlers.go

---

### Feature #2: Interview Recording & Playback 📹

**Priority:** HIGH | **Effort:** 5 days | **Impact:** +400% ATS integration

#### Infrastructure Setup
```yaml
# docker-compose addition
minio:
  image: minio/minio
  env:
    MINIO_ROOT_USER: minioadmin
    MINIO_ROOT_PASSWORD: minioadmin
  volumes:
    - minio_data:/minio_data

whisper-service:
  image: openai/whisper:latest
  ports:
    - "8084:8000"
  environment:
    WORKERS: 4
```

#### Database Schema
```sql
CREATE TABLE IF NOT EXISTS interview_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id),
    video_url VARCHAR(512),
    audio_url VARCHAR(512),
    duration_seconds INT,
    file_size_bytes BIGINT,
    storage_key VARCHAR(512),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS interview_transcriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recording_id UUID NOT NULL REFERENCES interview_recordings(id),
    full_text TEXT,
    segments JSONB, -- { timestamp, speaker, text }
    language VARCHAR(10),
    transcribed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS recording_playback_markers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recording_id UUID NOT NULL REFERENCES interview_recordings(id),
    timestamp_seconds INT,
    marker_type VARCHAR(50), -- key_moment, question, answer, etc
    description TEXT,
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Implementation Tasks
- [ ] MediaRecorder setup in frontend
- [ ] Video chunk upload to MinIO
- [ ] Whisper integration for transcription
- [ ] Playback component with timeline
- [ ] Key moments highlighting
- [ ] Playback search/index

#### Files to Create
- services/interview-service/internal/migration/007_recordings.up.sql
- services/interview-service/internal/service/recording_service.go
- services/interview-service/internal/api/recording_handlers.go
- services/interview-service/pkg/storage/minio_client.go
- services/interview-service/pkg/whisper/transcription_client.go

---

## Phase 3: Analytics & Integrations

### Feature #5: Candidate Comparison & Analytics Dashboard 📊

**Priority:** MEDIUM | **Effort:** 4 days | **Impact:** +2x decision quality

#### Database Enhancements
```sql
CREATE TABLE IF NOT EXISTS candidate_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    metric_name VARCHAR(100),
    metric_value NUMERIC(10,2),
    measured_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    INDEX idx_candidate_metrics_user (user_id)
);

CREATE TABLE IF NOT EXISTS interv viewer_consensus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    consensus_score NUMERIC(3,2),
    disagreement_level INT,
    recommendation VARCHAR(20), -- HIRE, PASS, MAYBE
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Implementation Tasks
- [ ] Side-by-side comparison endpoint
- [ ] Skill matrix visualization API
- [ ] Leaderboard computation
- [ ] Benchmarking stats
- [ ] Export comparison as PDF

#### Files to Create
- services/analytics-service/internal/api/comparison_handlers.go
- services/analytics-service/internal/service/comparison_service.go

---

### Feature #6: Smart Scheduling & Calendar Integration 🗓️

**Priority:** MEDIUM | **Effort:** 3 days | **Impact:** +80% compliance

#### Integration Points
```
External APIs:
- Google Calendar API
- Microsoft Graph (Outlook)
- Twilio (SMS reminders)
```

#### Database Schema
```sql
CREATE TABLE IF NOT EXISTS interview_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id),
    scheduled_at TIMESTAMP WITH TIME ZONE,
    interviewer_id UUID NOT NULL,
    candidate_id UUID NOT NULL,
    timezone VARCHAR(50),
    reminder_sent BOOLEAN DEFAULT FALSE,
    calendar_event_id VARCHAR(255),
    status VARCHAR(20), -- scheduled, confirmed, completed, cancelled
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS calendar_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    provider VARCHAR(50), -- google, outlook, etc
    access_token VARCHAR(1024),
    refresh_token VARCHAR(1024),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Implementation Tasks
- [ ] OAuth integration setup (Google/Outlook)
- [ ] Time slot availability checking
- [ ] Auto-suggest best times
- [ ] Calendar sync background job
- [ ] Reminder scheduling (email/SMS)
- [ ] Timezone handling

#### Files to Create
- services/notification-service/internal/calendar/google_calendar.go
- services/notification-service/internal/calendar/outlook_calendar.go
- services/notification-service/internal/api/calendar_handlers.go

---

### Feature #7: Real-time AI Coaching for Candidates 🎓

**Priority:** MEDIUM | **Effort:** 3 days | **Impact:** +150% candidate performance

#### Database Schema
```sql
CREATE TABLE IF NOT EXISTS coaching_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_session_id UUID NOT NULL,
    coaching_events JSONB, -- array of coaching messages
    confidence_trend JSONB, -- { timestamp, score }
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Implementation Tasks
- [ ] Real-time confidence scoring
- [ ] Answer quality assessment
- [ ] Time management warnings
- [ ] Hint generation
- [ ] Post-interview improvement plan
- [ ] WebSocket coaching messages

#### Files to Create
- services/ai-service/src/coaching/coach.py
- services/interview-service/internal/api/coaching_handlers.go

---

### Feature #8: Interview Feedback & 360 Reviews ⭐

**Priority:** MEDIUM | **Effort:** 3 days | **Impact:** +60% holistic view

#### Database Schema
```sql
CREATE TABLE IF NOT EXISTS interview_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id),
    provided_by UUID NOT NULL,
    category VARCHAR(50), -- technical, communication, culture_fit
    rating INT,
    short_comment TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS peer_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL,
    reviewer_id UUID NOT NULL,
    rating INT,
    strengths TEXT,
    areas_for_improvement TEXT,
    review_type VARCHAR(50), -- peer, manager, 360
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### Implementation Tasks
- [ ] Structured feedback forms
- [ ] Response aggregation
- [ ] 360 review workflow
- [ ] Anonymous feedback option
- [ ] Feedback summary generation
- [ ] Bias detection

#### Files to Create
- services/interview-service/internal/migration/008_feedback.up.sql
- services/interview-service/internal/api/feedback_handlers.go

---

### Feature #9: ATS Integration & Webhook Sync 🔗

**Priority:** MEDIUM | **Effort:** 4 days | **Impact:** +1000% efficiency

#### Supported ATS Platforms
```
- Workable
- Greenhouse
- Lever
- BambooHR
- iCIMS
- Taleo
```

#### Implementation Tasks
- [ ] ATS adapter pattern
- [ ] OAuth token management
- [ ] Candidate sync endpoint
- [ ] Result push to ATS
- [ ] Webhook receiver for ATS events
- [ ] API rate limiting & retry logic
- [ ] Batch import/export

#### Files to Create
- services/api-gateway/internal/ats/workable_adapter.go
- services/api-gateway/internal/ats/greenhouse_adapter.go
- services/api-gateway/internal/ats/ats_service.go
- services/api-gateway/internal/api/ats_handlers.go

---

### Feature #10: Predictive Analytics & ML Recommendations 🤖

**Priority:** LOW | **Effort:** 5 days | **Impact:** +40% success rate

#### ML Model Training
```python
# services/ai-service/src/models/hiring_predictor.py
Features:
- Technical score
- Communication ability
- Problem-solving skills
- Culture fit alignment
- Interview duration
- Answer confidence

Target:
- Hire probability (0-1)
- Success likelihood within 6 months
```

#### Implementation Tasks
- [ ] Feature engineering pipeline
- [ ] Model training infrastructure
- [ ] Prediction API endpoint
- [ ] A/B testing framework
- [ ] Bias detection & mitigation
- [ ] Historical accuracy metrics
- [ ] Explainability scores

#### Files to Create
- services/ai-service/src/models/hiring_predictor.py
- services/ai-service/src/models/bias_detector.py
- services/ai-service/src/api/prediction_endpoints.py

---

## Implementation Timeline

```
Week 1:
  - Complete Feature #3 (Multi-Interviewer)
  - Start Feature #4 (Templates)

Week 2:
  - Complete Feature #4 (Templates)
  - Start Feature #2 (Recording)

Week 3:
  - Complete Feature #2 (Recording)
  - Start Feature #5 (Analytics)
  - Start Feature #6 (Scheduling)

Week 4:
  - Complete Feature #5 & #6
  - Start Feature #7 (Coaching)
  - Start Feature #8 (Feedback)

Week 5:
  - Complete Feature #7 & #8
  - Start Feature #9 (ATS)

Week 6:
  - Complete Feature #9
  - Start Feature #10 (ML)
  - Testing & QA

Week 7:
  - Complete Feature #10
  - Load testing
  - Production deployment
```

## Dependencies & Integration Points

```
Feature #3 (Collab) → Uses interview_sessions from Feature #1
Feature #4 (Templates) → Uses interview_messages from Feature #1
Feature #2 (Recording) → Uses interview_sessions
Feature #5 (Analytics) → Uses Reports from Feature #1
Feature #6 (Scheduling) → Uses interview_sessions
Feature #7 (Coaching) → Uses AI Service
Feature #8 (Feedback) → Uses interview_sessions
Feature #9 (ATS) → Uses all session/report data
Feature #10 (ML) → Uses Features #5 + #8 data
```

## Deployment Checklist

- [ ] All services compile and pass tests
- [ ] Database migrations applied to production DB
- [ ] Docker images built and pushed
- [ ] Kubernetes manifests updated
- [ ] API documentation updated
- [ ] Frontend components built
- [ ] E2E tests passing
- [ ] Load tests passing (1000 concurrent users)
- [ ] Security audit completed
- [ ] User documentation prepared
- [ ] Training materials created
- [ ] Rollback plan documented

## Risk Mitigation

| Feature | Risk | Mitigation |
|---------|------|-----------|
| Code Executor | Security (arbitrary code) | Sandboxing, resource limits, pattern validation |
| Recording | Storage costs | Video compression, retention policy, tiered storage |
| Multi-Interviewer | WebSocket scale | Connection pooling, message batching |
| Templates | Adoption | Templates from top companies, recommendations |
| ML Models | Bias | Fairness audits, demographic parity monitoring |
| ATS Sync | Data consistency | Idempotency keys, retry logic |
| Scheduling | Calendar conflicts | Real-time availability checking |
| Coaching | LLM latency | Caching, pre-generation of hints |

---

## Success Metrics

After completing all 10 features:

| Metric | Target | Measurement |
|--------|--------|-------------|
| User satisfaction | >4.5/5 | NPS survey |
| Time to hire | -30% | Median days |
| Hire success rate | +15% | Retention at 6 months |
| Interviewer efficiency | +200% | Sessions per interviewer/week |
| Candidate experience | >90% | NPS score |
| System reliability | 99.99% | Uptime SLA |
| API response time | <200ms | P95 latency |
| Code execution cost | <$0.001 | Cost per submission |

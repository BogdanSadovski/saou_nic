# Local Startup Checklist

This checklist collects the minimum data and infrastructure needed to run the platform locally.

## 1. Fixed environment requirements

- Go backend services run with the Go toolchain already declared in each module.
- AI service must run on Python 3.12.
- Avoid Python 3.14 for `services/ai-service`; dependency installation fails there.
- Create a dedicated virtual environment for AI service before installing dependencies.

## 2. Core infrastructure to have running

- PostgreSQL
- Redis
- RabbitMQ
- Kafka
- ClickHouse
- MinIO or another S3-compatible storage
- OpenAI API access for the AI service
- SMTP credentials for email delivery
- Firebase service account if push notifications are used
- Twilio or another SMS provider if SMS notifications are used

## 3. Application data and secrets

### User service

- PostgreSQL database: `user_service`
- JWT secret
- Google OAuth client ID and secret, if Google login is enabled
- GitHub OAuth client ID and secret, if GitHub login is enabled

### Admin service

- PostgreSQL database: `admin_service`
- JWT secret
- Admin RBAC data in the same database

### Analytics service

- PostgreSQL database: `analytics`
- ClickHouse database: `analytics`
- Kafka broker list
- Kafka topics: `events`, `user-activity`

### GitHub service

- PostgreSQL database for GitHub sync state
- GitHub personal access token
- GitHub API base URL if a non-default endpoint is used

### Interview service

- PostgreSQL database: `interview_service`
- Redis instance
- JWT secret for interview auth/session flow

### Notification service

- PostgreSQL database: `notifications_db`
- RabbitMQ URL
- SMTP host, username, and password
- Firebase service account key file, if push notifications are enabled
- SMS provider credentials, if SMS notifications are enabled

### Report service

- PostgreSQL database: `report_service`
- S3 or MinIO endpoint
- S3 access key ID and secret access key
- S3 bucket name: `reports`

### Resume service

- PostgreSQL database: `resume_db`
- S3 or MinIO endpoint
- S3 access key ID and secret access key
- S3 bucket name: `resumes`

### Scoring service

- PostgreSQL database: `scoring_db`
- gRPC port available for internal service communication

### AI service

- OpenAI API key
- Optional custom LLM base URL if you use a compatible local model endpoint

### Frontend

- `VITE_API_BASE_URL`
- `VITE_API_WS_URL`
- `VITE_APP_NAME`
- `VITE_APP_VERSION`
- `VITE_ENABLE_ANALYTICS`

## 4. Suggested local startup order

1. Start PostgreSQL, Redis, RabbitMQ, Kafka, ClickHouse, and MinIO.
2. Create the required databases and users for each service.
3. Export or place the required environment variables and secrets.
4. Create the AI service virtual environment with Python 3.12 and install its dependencies.
5. Start the backend services.
6. Start the frontend.

## 5. Minimal sanity checks

- `go test ./...` passes in every Go service directory.
- AI service installs cleanly in the Python 3.12 virtual environment.
- Each service can connect to its declared database or message broker.
- Frontend can reach the backend API and websocket endpoint.
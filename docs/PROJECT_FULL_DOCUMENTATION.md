# Полная техническая документация проекта AI Interview Platform

Версия документа: 1.0

Дата: 27.04.2026

Статус: рабочая инженерная документация (поддерживается командой разработки)

---

## 1. Назначение документа

Этот документ описывает проект целиком: архитектуру, сервисы, границы ответственности, модели данных, интеграционные контракты, инфраструктуру, процессы разработки, тестирования, эксплуатации, мониторинга и масштабирования.

Цель документа:
- дать единую точку входа для новых разработчиков;
- зафиксировать текущую фактическую структуру системы;
- упростить сопровождение, внедрение изменений и аудит;
- снизить риски при релизах и инцидентах.

Целевая аудитория:
- backend/frontend инженеры;
- DevOps/SRE;
- QA;
- техлиды и архитекторы;
- аналитики и менеджеры, которым нужна техническая прозрачность.

---

## 2. Краткое описание продукта

AI Interview Platform — микросервисная платформа автоматизации технических интервью и оценки кандидатов. Система объединяет:
- управление пользователями и ролями;
- анализ резюме;
- интеграцию с GitHub-профилем;
- AI-интервью в реальном времени;
- скоринг кандидатов;
- формирование отчетов;
- отправку уведомлений;
- аналитику и дашборды.

Платформа предназначена для внутренних и внешних интервью-процессов компаний, где важны скорость обработки кандидатов, стандартизированная оценка и воспроизводимость результатов.

---

## 3. Технологический стек

### 3.1 Backend
- Go (микросервисы доменного контура)
- gRPC для межсервисного взаимодействия
- REST/HTTP API на уровне сервисов и API Gateway

### 3.2 AI-контур
- Python 3.12
- FastAPI
- LLM-интеграция (OpenAI API и совместимые endpoints)

### 3.3 Frontend
- React 18
- TypeScript
- Vite

### 3.4 Данные и хранилища
- PostgreSQL (операционные данные)
- Redis (кэш/сессии/временные данные)
- ClickHouse (аналитические события и агрегации)
- S3/MinIO (файлы резюме, отчеты, артефакты)

### 3.5 Интеграция и messaging
- RabbitMQ (очереди задач и уведомлений)
- Kafka (потоковая аналитика и event-driven контур)

### 3.6 Инфраструктура и сопровождение
- Docker / Docker Compose
- Kubernetes
- Prometheus, Grafana, Loki
- GitHub Actions / GitLab CI (по структуре репозитория)

---

## 4. Архитектурные принципы

### 4.1 Микросервисная декомпозиция
Каждый сервис отвечает за отдельный bounded context и имеет:
- собственную кодовую базу;
- собственные миграции и схему БД;
- четкие контракты взаимодействия.

### 4.2 Изоляция данных
Сервисы ориентированы на pattern database-per-service (в рамках общего кластера PostgreSQL). Это снижает связанность и упрощает независимые релизы.

### 4.3 Контрактный подход
Используются явные контракты:
- OpenAPI для REST;
- protobuf для gRPC;
- versioned envelope для WebSocket событий.

### 4.4 Event-driven расширяемость
Часть сценариев построена на обмене событиями через Kafka/RabbitMQ, что упрощает асинхронные процессы и интеграцию новых потребителей.

### 4.5 Наблюдаемость по умолчанию
В архитектуру заложены метрики, логи и health-check endpoints для устойчивой эксплуатации в продакшене.

---

## 5. Карта репозитория

Ключевые директории:
- `services/` — все бизнес-сервисы платформы
- `frontend/` — клиентское приложение
- `infrastructure/` — docker, k8s, ci/cd, скрипты
- `shared/proto/` — protobuf контракты
- `shared/packages/` — общие библиотеки (Go/Python)
- `docs/` — архитектурная и эксплуатационная документация
- `tests/` — unit/integration/e2e/load тесты уровня платформы

Сервисный слой включает:
- user-service
- resume-service
- github-service
- interview-service
- ai-service
- scoring-service
- report-service
- notification-service
- analytics-service
- admin-service
- api-gateway

---

## 6. Сервисный каталог (детальное описание)

## 6.1 user-service

Назначение:
- регистрация/аутентификация;
- OAuth-провайдеры;
- управление профилем пользователя;
- выдача/валидация JWT в рамках пользовательского контура.

Структура:
- `internal/api` — HTTP handlers, middleware, routes
- `internal/service` — auth_service, oauth_service, user_service
- `internal/repository` — PostgreSQL layer
- `internal/grpc` — gRPC сервер/handlers

Данные:
- таблица `users` (ID, email, username, роль, статус, provider, timestamps)

Интеграции:
- Redis, Kafka, RabbitMQ
- OAuth (Google/GitHub)

Риски:
- компрометация JWT секрета;
- рассинхронизация OAuth callback flow;
- высокий риск lockout при ошибках в role/status политике.

---

## 6.2 admin-service

Назначение:
- админское управление пользователями;
- RBAC;
- подписки;
- аудит действий.

Структура:
- `internal/service`: admin_service, user_management, subscription_service, audit_service
- `pkg/rbac`: permissions

Данные:
- `users` (расширенная админская модель, включая 2FA и soft-delete)
- `subscriptions`
- `audit_logs`

Особенности:
- полнотекстовый индекс по user identity полям;
- дефолтный super-admin seed в миграции;
- журнал аудита критичных действий.

---

## 6.3 resume-service

Назначение:
- прием резюме;
- хранение и извлечение структурированных данных;
- NLP-пайплайн анализа.

Структура:
- `internal/nlp`: parser, extractor
- `internal/service`: resume_service, nlp_analyzer
- `internal/repository`: postgres, s3_storage

Данные:
- `resumes` с JSONB-полями: skills, experience, education, languages, certifications

Интеграции:
- S3/MinIO bucket `resumes`
- PostgreSQL

Сценарии отказа:
- ошибки парсинга/формата файла;
- недоступность S3;
- длинные NLP-процессы (таймауты).

---

## 6.4 github-service

Назначение:
- интеграция с GitHub API;
- анализ репозиториев и активности;
- получение contribution сигналов для скоринга.

Структура:
- `internal/github`: client, rate_limiter
- `internal/service`: github_service, repo_analyzer, contribution_analyzer

Интеграции:
- GitHub Token/Client credentials
- PostgreSQL для кэша/состояния синхронизации

Риски:
- rate limits GitHub;
- токены доступа;
- частичная деградация внешнего API.

---

## 6.5 interview-service

Назначение:
- оркестрация интервью-сессий;
- управление вопросами и ответами;
- WebSocket взаимодействие;
- расширенные режимы AI и multi-interviewer.

Структура:
- `internal/service`: interview_service, session_manager, question_generator
- `internal/websocket`: handler, room
- `internal/repository`: session_queries

Данные (базовый контур):
- `interviews`
- `questions`
- `sessions`
- `answers`

Данные (AI контур):
- `interview_sessions`
- `interview_messages`
- `interview_reports`
- `request_log` (идемпотентность)

Данные (кодинг):
- `code_submissions`
- `code_execution_results`
- `code_test_cases`

Данные (multi-interviewer):
- `interview_collaborators`
- `collaboration_notes`
- `interviewer_scores`
- `interview_consensus`
- `score_audit_log`

Особенности:
- контроль сложности и pressure-level;
- web socket stream с versioned event envelope;
- поддержка идемпотентных запросов через `Idempotency-Key`.

---

## 6.6 ai-service

Назначение:
- генерация вопросов;
- оценка ответов кандидата;
- пост-анализ интервью;
- динамическая адаптация поведения интервьюера.

Структура:
- `src/core`: llm_client, prompt_templates, embeddings
- `src/services`: question_service, analysis_service, transcription_service
- `src/api/routes.py`: API endpoints и orchestration

Особенности текущей логики (по состоянию кода):
- усиленные prompt guardrails для interviewer turn;
- response policy с разделением поведения для weak/partial/strong ответов;
- эвристики пост-анализа для кратких/уклончивых ответов;
- fallback логика при ошибке LLM на endpoint next-question.

Тесты:
- `tests/test_api.py`
- `tests/test_services.py`

Критичные зависимости:
- Python 3.12
- OpenAI API key

---

## 6.7 scoring-service

Назначение:
- вычисление оценок;
- применение рубрик;
- агрегирование breakdown по критериям.

Структура:
- `internal/service`: scoring_service, evaluation_engine, rubric_calculator
- `pkg/scoring`: algorithms, weights

Данные:
- `scores`
- `rubrics`

Особенности:
- status-driven scoring pipeline (`pending` -> final statuses)
- привязка к rubric через nullable FK.

---

## 6.8 report-service

Назначение:
- генерация PDF/DOCX отчетов;
- хранение метаданных файлов;
- отдача ссылок/статусов генерации.

Структура:
- `internal/service`: report_service, pdf_generator, docx_generator
- `internal/templates`
- `pkg/generator`

Данные:
- таблица `reports`

Форматы:
- `pdf`
- `docx`

Типы отчетов:
- interview_report
- candidate_summary
- assessment_report
- comparative_analysis

---

## 6.9 notification-service

Назначение:
- доставлять уведомления через email/push/sms;
- повторные попытки и dead-letter-like поведение на уровне статусов.

Структура:
- `internal/service`: notification_service, email_service, push_service, sms_service
- `internal/consumer`: rabbitmq handlers
- `templates`: html шаблоны уведомлений

Данные:
- `notifications`

Особенности:
- поля retry_count/max_retries;
- pending partial index для ускорения очереди обработки.

---

## 6.10 analytics-service

Назначение:
- сбор и агрегация продуктовой/операционной аналитики;
- дашборды и экспорт.

Структура:
- `internal/service`: analytics_service, dashboard_service, export_service
- `internal/repository`: postgres + clickhouse
- `internal/consumer`: kafka event_handlers

Данные PostgreSQL:
- `dashboards`
- `export_requests`
- `user_sessions`
- `funnels`

Данные ClickHouse:
- `events`
- `aggregated_metrics`
- `event_counts`
- materialized view `event_counts_mv`

Особенности:
- retention policy в ClickHouse (`TTL 90 DAY`)
- pre-aggregations для дешевых дашбордов.

---

## 6.11 api-gateway

Назначение:
- единая входная точка для frontend;
- маршрутизация запросов по сервисам;
- унификация внешнего API.

Судя по compose-конфигурации, gateway проксирует:
- user-service
- resume-service
- github-service
- interview-service
- ai-service
- report-service
- admin-service

---

## 7. Межсервисные коммуникации

Каналы:
- HTTP/REST
- gRPC
- WebSocket
- Kafka
- RabbitMQ

Рекомендованный принцип:
- синхронные запросы для коротких query-like операций;
- асинхронные события для тяжелых/долгих процессов;
- идемпотентность и correlation IDs для повторов и трассировки.

### 7.1 Пример event envelope (WebSocket v1)

```json
{
  "version": "v1",
  "type": "event.type",
  "timestamp": "2026-04-27T10:00:00Z",
  "payload": {}
}
```

### 7.2 Идемпотентность

Для `POST /interviews/sessions` и `POST /interviews/sessions/{id}/messages` поддерживается `Idempotency-Key`.

---

## 8. API-контракты (высокоуровнево)

На основе `docs/api/interview-module.openapi.yaml`:

### 8.1 Interview module endpoints
- `POST /api/v1/interviews/sessions` — создать сессию
- `GET /api/v1/interviews/sessions/{session_id}` — получить сессию
- `GET /api/v1/interviews/sessions/{session_id}/messages` — получить сообщения
- `POST /api/v1/interviews/sessions/{session_id}/messages` — отправить сообщение, получить async next question
- `POST /api/v1/interviews/sessions/{session_id}/finish` — завершить сессию
- `GET /api/v1/interviews/sessions/{session_id}/report` — получить отчет
- `GET /api/v1/interviews/sessions/{session_id}/ws` — WebSocket поток

### 8.2 Безопасность
- Bearer JWT для защищенных маршрутов.

### 8.3 Базовые схемы
- `ApiResponse { success, data, error }`
- `CreateSessionRequest`
- `SendMessageRequest`

---

## 9. Модель данных (полный перечень таблиц)

Ниже перечислены все обнаруженные сущности из `.up.sql` миграций.

### 9.1 user-service

#### Таблица users
Поля:
- id UUID PK
- email VARCHAR(255) unique
- username VARCHAR(100) unique
- password_hash VARCHAR(255)
- first_name VARCHAR(100)
- last_name VARCHAR(100)
- avatar_url TEXT
- role VARCHAR(20)
- status VARCHAR(20)
- provider VARCHAR(20)
- provider_id VARCHAR(255)
- email_verified BOOLEAN
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ
- last_login_at TIMESTAMPTZ

Ограничения:
- role in (user, admin)
- status in (active, inactive, banned)
- provider in (local, google, github)

Индексы:
- email, username, provider+provider_id, status, created_at desc

---

### 9.2 admin-service

#### Таблица users
Поля:
- id UUID PK
- email, username, password_hash
- role, status
- first_name, last_name, avatar_url
- last_login_at
- email_verified
- two_factor_enabled
- created_at, updated_at
- deleted_at

Ограничения:
- role in (super_admin, admin, moderator, user)
- status in (active, inactive, suspended, banned)

Индексы:
- email, username, role, status, created_at desc, deleted_at partial, full-text search gin

#### Таблица subscriptions
Поля:
- id UUID PK
- user_id UUID FK users(id)
- tier, status
- start_date, end_date
- auto_renew
- max_users
- max_storage_gb
- features TEXT[]
- metadata JSONB
- created_at, updated_at

Ограничения:
- tier in (free, basic, pro, enterprise)
- status in (active, expired, canceled, pending)

Индексы:
- user_id, tier, status, end_date desc, created_at desc, active partial

#### Таблица audit_logs
Поля:
- id UUID PK
- admin_id UUID
- admin_email VARCHAR(255)
- action VARCHAR(100)
- resource_type VARCHAR(100)
- resource_id UUID
- details TEXT
- ip_address VARCHAR(45)
- user_agent TEXT
- created_at TIMESTAMPTZ

Ограничения:
- action in (create, update, delete, login, logout, suspend_user, ban_user, change_role, change_subscription)

Индексы:
- admin_id, action, resource_type, resource_id, created_at desc, admin_id+created_at desc

---

### 9.3 resume-service

#### Таблица resumes
Поля:
- id UUID PK
- user_id VARCHAR(255)
- file_name VARCHAR(500)
- file_url TEXT
- content_type VARCHAR(100)
- status VARCHAR(50)
- first_name, last_name, email, phone
- summary TEXT
- skills JSONB
- experience JSONB
- education JSONB
- languages JSONB
- certifications JSONB
- metadata JSONB
- created_at, updated_at
- error TEXT

Индексы:
- user_id, status, created_at desc, user_id+status

---

### 9.4 interview-service (base)

#### Таблица interviews
Поля:
- id UUID PK
- interviewer_id UUID
- candidate_id UUID
- title VARCHAR(255)
- description TEXT
- status VARCHAR(50)
- scheduled_at TIMESTAMPTZ
- duration INTEGER
- language VARCHAR(50)
- created_at, updated_at

Ограничения:
- status in (scheduled, in_progress, completed, cancelled, no_show)
- duration 15..180

Индексы:
- interviewer_id, candidate_id, status, scheduled_at

#### Таблица questions
Поля:
- id UUID PK
- interview_id UUID FK interviews(id)
- title VARCHAR(255)
- description TEXT
- type VARCHAR(50)
- difficulty VARCHAR(50)
- tags JSONB
- starter_code TEXT
- solution TEXT
- test_cases JSONB
- points INTEGER
- question_order INTEGER
- created_at TIMESTAMPTZ

Ограничения:
- type in (coding, system_design, behavioral, debugging)
- difficulty in (easy, medium, hard, expert)

Индексы:
- interview_id, type, difficulty, question_order

#### Таблица sessions
Поля:
- id UUID PK
- interview_id UUID FK interviews(id)
- status VARCHAR(50)
- current_question_index INTEGER
- start_time TIMESTAMPTZ
- end_time TIMESTAMPTZ
- score INTEGER
- feedback TEXT
- created_at, updated_at

Индексы:
- interview_id, status

#### Таблица answers
Поля:
- id UUID PK
- session_id UUID FK sessions(id)
- question_id UUID FK questions(id)
- code TEXT
- language VARCHAR(50)
- is_correct BOOLEAN
- score INTEGER
- submitted_at TIMESTAMPTZ
- created_at TIMESTAMPTZ

Индексы:
- session_id, question_id, submitted_at

---

### 9.5 interview-service (AI)

#### Таблица interview_sessions
Поля:
- id UUID PK
- user_id UUID
- role VARCHAR(120)
- level VARCHAR(20)
- status VARCHAR(20)
- current_topic VARCHAR(120)
- difficulty_score INTEGER
- pressure_level INTEGER
- question_count INTEGER
- question_limit INTEGER
- started_at TIMESTAMPTZ
- ended_at TIMESTAMPTZ
- duration_seconds INTEGER
- metadata JSONB
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ
- interview_mode VARCHAR(50)

Ограничения:
- level in (junior, middle, senior)
- status in (created, active, finished, failed)
- difficulty_score 1..10
- pressure_level 1..5
- question_limit 1..200
- question_count 0..question_limit
- duration_seconds 60..21600

Индексы:
- user_id, status, started_at desc

#### Таблица interview_messages
Поля:
- id UUID PK
- session_id UUID FK interview_sessions(id)
- sender VARCHAR(16)
- content TEXT
- topic VARCHAR(120)
- difficulty INTEGER
- created_at TIMESTAMPTZ
- token_usage JSONB
- coding_task JSONB
- test_cases_count INT

Ограничения:
- sender in (ai, user, system)
- difficulty null or 1..10

Индексы:
- session_id+created_at
- sender

#### Таблица interview_reports
Поля:
- id UUID PK
- session_id UUID unique FK interview_sessions(id)
- correctness NUMERIC(5,2)
- clarity NUMERIC(5,2)
- completeness NUMERIC(5,2)
- relevance NUMERIC(5,2)
- overall_score NUMERIC(5,2)
- strengths JSONB[]
- weaknesses JSONB[]
- recommendations JSONB[]
- generated_at TIMESTAMPTZ

Ограничения:
- все score-поля 0..100

#### Таблица request_log
Поля:
- id BIGSERIAL PK
- idempotency_key VARCHAR(128) unique
- session_id UUID FK interview_sessions(id)
- response_hash VARCHAR(128)
- created_at TIMESTAMPTZ

Индексы:
- session_id+created_at desc

---

### 9.6 interview-service (coding)

#### Таблица code_submissions
Поля:
- id UUID PK
- session_id UUID FK interview_sessions(id)
- user_id UUID
- language VARCHAR(50)
- code TEXT
- input_data TEXT
- submission_sequence INT
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- session_id, user_id

#### Таблица code_execution_results
Поля:
- id UUID PK
- submission_id UUID FK code_submissions(id)
- status VARCHAR(50)
- output TEXT
- error_message TEXT
- execution_time_ms BIGINT
- memory_used_bytes BIGINT
- exit_code INT
- test_results JSONB
- created_at TIMESTAMPTZ

Индексы:
- submission_id, status

#### Таблица code_test_cases
Поля:
- id UUID PK
- question_id UUID
- test_name VARCHAR(255)
- input_data TEXT
- expected_output TEXT
- is_hidden BOOLEAN
- sequence INT
- created_at TIMESTAMPTZ

Индексы:
- question_id

---

### 9.7 interview-service (multi-interviewer)

#### Таблица interview_collaborators
Поля:
- id UUID PK
- session_id UUID FK interview_sessions(id)
- user_id UUID
- role VARCHAR(50)
- joined_at TIMESTAMPTZ
- left_at TIMESTAMPTZ
- is_active BOOLEAN
- created_at TIMESTAMPTZ

Ограничения:
- role in (lead, observer, co-interviewer)

Индексы:
- session_id, user_id, is_active

#### Таблица collaboration_notes
Поля:
- id UUID PK
- session_id UUID FK interview_sessions(id)
- author_id UUID
- content TEXT
- version INT
- is_pinned BOOLEAN
- mentions JSONB
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- session_id, author_id, created_at desc

#### Таблица interviewer_scores
Поля:
- id UUID PK
- session_id UUID FK interview_sessions(id)
- interviewer_id UUID
- technical_score INT
- communication_score INT
- problem_solving_score INT
- culture_fit_score INT
- coding_quality_score INT
- recommendation VARCHAR(20)
- strengths TEXT
- areas_for_improvement TEXT
- additional_comments TEXT
- submitted_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Ограничения:
- score категории 0..10
- recommendation in (STRONG_YES, YES, MAYBE, NO, STRONG_NO)

Индексы:
- session_id, interviewer_id, submitted_at

#### Таблица interview_consensus
Поля:
- id UUID PK
- session_id UUID unique FK interview_sessions(id)
- avg_technical_score NUMERIC(3,1)
- avg_communication_score NUMERIC(3,1)
- avg_problem_solving_score NUMERIC(3,1)
- avg_culture_fit_score NUMERIC(3,1)
- avg_coding_quality_score NUMERIC(3,1)
- score_variance NUMERIC(5,2)
- disagreement_level VARCHAR(20)
- consensus_recommendation VARCHAR(20)
- confidence_score NUMERIC(3,2)
- alignments JSONB
- calculated_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Ограничения:
- disagreement_level in (LOW, MEDIUM, HIGH)

Индексы:
- session_id

#### Таблица score_audit_log
Поля:
- id UUID PK
- session_id UUID
- interviewer_id UUID
- action VARCHAR(50)
- old_scores JSONB
- new_scores JSONB
- change_reason TEXT
- created_at TIMESTAMPTZ

Ограничения:
- action in (CREATED, UPDATED, SUBMITTED, DELETED)

Индексы:
- session_id, interviewer_id

---

### 9.8 scoring-service

#### Таблица scores
Поля:
- id UUID PK
- submission_id VARCHAR(255)
- score_type VARCHAR(50)
- total_score DOUBLE PRECISION
- max_score DOUBLE PRECISION
- percentage DOUBLE PRECISION
- grade VARCHAR(5)
- breakdown JSONB
- status VARCHAR(20)
- rubric_id UUID nullable FK rubrics(id)
- error_message TEXT
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- submission_id, score_type, status, created_at desc

#### Таблица rubrics
Поля:
- id UUID PK
- name VARCHAR(255)
- score_type VARCHAR(50)
- criteria JSONB
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- score_type

---

### 9.9 report-service

#### Таблица reports
Поля:
- id UUID PK
- candidate_id VARCHAR(255)
- interview_id VARCHAR(255)
- assessment_id VARCHAR(255)
- type VARCHAR(50)
- format VARCHAR(10)
- status VARCHAR(20)
- title VARCHAR(500)
- description TEXT
- file_url TEXT
- file_name VARCHAR(500)
- file_size BIGINT
- error_message TEXT
- metadata JSONB
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ
- expires_at TIMESTAMPTZ
- generated_by VARCHAR(255)

Ограничения:
- status in (pending, generating, completed, failed, expired)
- format in (pdf, docx)
- type in (interview_report, candidate_summary, assessment_report, comparative_analysis)

Индексы:
- candidate_id, status, type, format, created_at desc, expires_at partial
- candidate_id+status, status+created_at desc

---

### 9.10 notification-service

#### Таблица notifications
Поля:
- id BIGSERIAL PK
- user_id BIGINT
- type VARCHAR(50)
- channel VARCHAR(50)
- priority VARCHAR(20)
- status VARCHAR(20)
- subject VARCHAR(500)
- body TEXT
- recipient VARCHAR(500)
- metadata JSONB
- retry_count INTEGER
- max_retries INTEGER
- error_message TEXT
- sent_at TIMESTAMPTZ
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- user_id, status, created_at desc, user_id+created_at desc, type, channel, pending partial

---

### 9.11 analytics-service (PostgreSQL)

#### Таблица dashboards
Поля:
- id UUID PK
- name VARCHAR(255)
- tenant_id VARCHAR(255)
- description TEXT
- widgets JSONB
- created_by VARCHAR(255)
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- tenant_id

#### Таблица export_requests
Поля:
- id UUID PK
- tenant_id VARCHAR(255)
- format VARCHAR(50)
- filter JSONB
- status VARCHAR(50)
- file_url TEXT
- error TEXT
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- tenant_id, status

#### Таблица user_sessions
Поля:
- id UUID PK
- user_id VARCHAR(255)
- session_id VARCHAR(255) unique
- duration FLOAT8
- created_at TIMESTAMPTZ

Индексы:
- user_id, session_id, created_at

#### Таблица funnels
Поля:
- id UUID PK
- name VARCHAR(255)
- tenant_id VARCHAR(255)
- steps JSONB
- created_at TIMESTAMPTZ
- updated_at TIMESTAMPTZ

Индексы:
- tenant_id

---

### 9.12 analytics-service (ClickHouse)

#### Таблица events
Назначение:
- хранение сырых событий аналитики.

Поля (основные):
- id String
- type LowCardinality(String)
- user_id String
- session_id String
- tenant_id LowCardinality(String)
- url, referrer, user_agent, ip
- country, city, device, os, browser
- properties String
- timestamp DateTime64(3)
- processed_at DateTime64(3)

Техпараметры:
- ENGINE MergeTree
- PARTITION BY toYYYYMM(timestamp)
- ORDER BY (tenant_id, type, timestamp)
- TTL 90 дней

#### Таблица aggregated_metrics
Назначение:
- предрасчитанные агрегаты по временным окнам.

Ключевые поля:
- tenant_id, window_start/window_end, granularity
- total_events, unique_users, unique_sessions
- page_views, clicks, conversions, errors
- avg_session_duration, bounce_rate

#### Таблица event_counts
Назначение:
- агрегированные счетчики событий по дате и типу.

#### Материализованный view event_counts_mv
Назначение:
- автоматическая запись агрегатов из events в event_counts.

---

## 10. Инфраструктура

## 10.1 Docker Compose окружение

Файл: `infrastructure/docker/docker-compose.yml`

Поднимает:
- Zookeeper
- Kafka
- RabbitMQ
- PostgreSQL
- Redis
- ClickHouse
- MinIO
- прикладные сервисы

Сети:
- backend-network
- frontend-network
- monitoring-network

Volume хранилища:
- postgres-data
- redis-data
- clickhouse-data
- rabbitmq-data
- kafka-data
- zookeeper-data
- minio-data

Ключевые особенности:
- healthcheck у большинства контейнеров;
- env-параметры с разумными дефолтами;
- разделение портов по сервисам.

## 10.2 dev compose

Файл: `infrastructure/docker/docker-compose.dev.yml`

Назначение:
- ускоренный локальный цикл разработки с mount исходников.

Важно:
- в dev-файле есть TS/npm-ориентированные команды и маппинги для большинства сервисов, что может быть шаблонным артефактом и требовать валидации под текущую Go/Python реализацию.

Рекомендация:
- проверить соответствие Dockerfile targets и фактического runtime каждого сервиса.

## 10.3 Kubernetes

Структура:
- `base/` — namespace, configmaps, secrets
- `services/*` — deployment/service/ingress/hpa
- `databases/` — postgres/redis/clickhouse
- `messaging/` — rabbitmq/kafka
- `ingress/` — ingress controller + tls
- `monitoring/` — prometheus/grafana/loki

Рекомендации production:
- external secret manager;
- ограничение ресурсов requests/limits;
- PDB/anti-affinity;
- rolling/blue-green стратегии.

---

## 11. Локальный запуск

Минимальные требования:
- Docker + Docker Compose
- Go toolchain
- Python 3.12 для AI service
- Node.js для frontend

Базовые команды:

```bash
make dev-up
make test-all
make build-all
```

Остановка:

```bash
make dev-down
```

Порядок старта (рекомендуемый):
1. Инфраструктура (PostgreSQL/Redis/RabbitMQ/Kafka/ClickHouse/MinIO)
2. Миграции
3. Backend сервисы
4. AI service
5. Frontend

См. также: `docs/development/local-startup-checklist.md`.

---

## 12. Конфигурация и переменные окружения

Ключевые категории переменных:
- доступ к БД (`POSTGRES_*`, `DATABASE_URL`)
- cache (`REDIS_URL`)
- messaging (`KAFKA_BROKERS`, `RABBITMQ_URL`)
- auth (`JWT_SECRET`, OAuth client IDs/secrets)
- object storage (`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`)
- AI (`OPENAI_API_KEY`, optional LLM base URL)
- frontend (`VITE_*`)

Правила:
- не хранить секреты в репозитории;
- использовать `.env.example` только как шаблон;
- для Kubernetes — secrets/configmaps + external secret backend.

---

## 13. Безопасность

## 13.1 Аутентификация и авторизация
- JWT Bearer для API.
- Ролевые модели в user/admin контурах.
- Проверка ролей и статусов на критичных маршрутах.

## 13.2 Безопасное хранение данных
- пароли только в hash-виде;
- управление сроком жизни токенов;
- ротация секретов.

## 13.3 Безопасность API
- валидация входных payload;
- ограничения размера сообщений;
- rate limiting для публичных endpoints;
- идемпотентность mutating запросов.

## 13.4 Безопасность интеграций
- токены GitHub/OAuth в secret store;
- SMTP/Firebase/Twilio credentials из защищенных источников.

## 13.5 Аудит
- таблица `audit_logs` для админских действий;
- `score_audit_log` для трассировки изменений оценок.

---

## 14. Наблюдаемость и эксплуатация

## 14.1 Health checks
Compose конфигурация показывает `/health` и аналогичные endpoints в сервисах.

## 14.2 Логи
Рекомендуемый стандарт:
- JSON logs;
- поля `timestamp`, `level`, `service`, `request_id`, `trace_id`, `user_id?`.

## 14.3 Метрики
Через Prometheus/Grafana:
- RPS, latency p95/p99
- error rate
- queue lag
- DB pool saturation
- cache hit ratio
- AI token usage и время генерации

## 14.4 Alerting
Рекомендуемые алерты:
- 5xx > порога
- рост latency
- падение consumer'ов
- backlog queue
- рост retry_count у notifications
- деградация внешних интеграций (GitHub/OpenAI/SMTP)

---

## 15. Тестовая стратегия

## 15.1 Уровни тестирования
- Unit: бизнес-логика сервисов
- Integration: API/gRPC/репозитории
- E2E: пользовательские флоу
- Load: сценарии в `tests/load/k6`

## 15.2 Команды
`make test-all` запускает:
- go test по всем Go сервисам с race и coverage;
- pytest по AI service;
- frontend tests с coverage.

## 15.3 Практики качества
- contract-first изменения API;
- регрессионные тесты на баги;
- deterministic mocks для LLM-путей;
- обязательная проверка идемпотентности критичных endpoint'ов.

---

## 16. CI/CD

По структуре репозитория:
- `.github/workflows/` (ci, cd, codeql)
- `infrastructure/ci-cd/github-actions/workflows/`
- `infrastructure/ci-cd/gitlab-ci/.gitlab-ci.yml`

Рекомендуемый pipeline:
1. lint
2. unit/integration tests
3. security scan
4. build images
5. push images
6. deploy staging
7. smoke tests
8. manual/auto promote в production

---

## 17. Основные бизнес-флоу

## 17.1 Аутентификация пользователя
1. Пользователь логинится локально или через OAuth.
2. user-service выдает JWT.
3. frontend сохраняет токен и использует в API.

## 17.2 Загрузка и анализ резюме
1. frontend отправляет файл в resume-service.
2. файл сохраняется в S3/MinIO.
3. NLP-пайплайн извлекает сущности и сохраняет JSONB-поля.
4. данные доступны для интервью и скоринга.

## 17.3 Запуск AI интервью
1. создается interview session.
2. кандидат и AI обмениваются сообщениями.
3. сохраняется история, токен usage, контекст.
4. при finish запускается пост-анализ.
5. формируется interview report.

## 17.4 Кодинг в интервью
1. кандидат отправляет code submission.
2. система выполняет тесты.
3. результаты пишутся в code_execution_results.
4. scoring-service использует данные для оценки.

## 17.5 Multi-interviewer
1. к сессии подключаются несколько интервьюеров.
2. каждый оставляет заметки и оценки.
3. формируется consensus.
4. изменения оценок логируются в audit trail.

## 17.6 Генерация отчета и нотификация
1. report-service генерирует PDF/DOCX.
2. статус обновляется в reports.
3. notification-service отправляет уведомление о готовности.

## 17.7 Аналитика
1. события попадают в Kafka/ClickHouse.
2. агрегаты строятся в aggregated_metrics/event_counts.
3. dashboards и export requests обслуживаются analytics-service.

---

## 18. Производительность и масштабирование

## 18.1 Горизонтальное масштабирование
- Stateless API сервисы масштабируются через replicas/HPA.
- Stateful компоненты масштабируются отдельно (PostgreSQL, Kafka, ClickHouse) с учетом их природы.

## 18.2 Узкие места
- AI latency и внешние LLM API лимиты
- ClickHouse/analytics insert throughput
- Report generation CPU/memory spikes
- Notification bursts

## 18.3 Кэширование
- Redis для часто читаемых данных и short-lived состояния.

## 18.4 Batch/async вынос
- тяжелые операции в очереди;
- API остается responsive через async accepted pattern.

---

## 19. Надежность и отказоустойчивость

## 19.1 Паттерны устойчивости
- retries с backoff
- circuit breaker для внешних API
- idempotency для write endpoints
- outbox/inbox pattern (рекомендуется для критичных событий)

## 19.2 Recovery
- backup scripts в `infrastructure/scripts/backup-db.sh`
- rollback scripts в `infrastructure/scripts/rollback.sh`

## 19.3 Disaster Recovery
Необходимо формализовать:
- RTO/RPO по сервисам
- регулярные restore drills
- multi-zone strategy для production кластеров

---

## 20. Runbook (инциденты)

## 20.1 Общая процедура
1. Подтвердить инцидент по алертам/логам.
2. Определить затронутые сервисы.
3. Проверить инфраструктуру (DB, queues, external APIs).
4. Оценить blast radius.
5. Применить mitigations.
6. Восстановить SLA.
7. Провести postmortem.

## 20.2 Частые кейсы

Кейс: LLM недоступен
- Проверить ключ/квоты/endpoint.
- Активировать fallback логики next-question.
- Уменьшить нагрузку и временно снизить timeout.

Кейс: backlog в notifications
- Проверить consumer availability.
- Проверить RabbitMQ очереди и requeue цикл.
- Масштабировать workers.

Кейс: медленная аналитика
- Проверить ClickHouse inserts и partitions.
- Проверить materialized view задержки.
- Временно сократить окно аналитических запросов.

---

## 21. Известные зоны риска и технический долг

1. Возможная рассинхронизация dev compose и фактического runtime (Go/Python vs npm dev commands).
2. Дублирование сущности users в разных контекстах требует ясной документации границ.
3. Требуется единая схема версионирования API/gRPC контрактов.
4. Необходима унификация observability стандартов (log fields, trace propagation).
5. Желательна централизация управления секретами в production.

---

## 22. Рекомендации по развитию

### 22.1 Архитектурные
- Внедрить service mesh или единый слой mTLS между сервисами.
- Добавить contract testing для gRPC/REST.
- Формализовать event schema registry для Kafka событий.

### 22.2 Data governance
- Единый data catalog по таблицам и полям.
- Политика retention/archival для PostgreSQL и ClickHouse.
- Политика PII masking/anonymization.

### 22.3 AI quality
- A/B тестирование prompt policies.
- Метрики качества интервью (precision/recall по рекомендациям).
- Human-in-the-loop review для high-stakes решений.

### 22.4 Engineering productivity
- шаблоны сервиса (scaffolding)
- golden path для observability/security
- обязательные quality gates в CI

---

## 23. Глоссарий

- Bounded Context: функциональная область ответственности сервиса.
- Idempotency: повтор запроса не меняет итоговый результат сверх первого применения.
- TTL: время жизни данных.
- HPA: Horizontal Pod Autoscaler.
- PII: персональные данные.
- SLA/SLO: целевые показатели доступности/качества сервиса.

---

## 24. Приложение A: Makefile команды

Главные группы команд:
- `make dev-up` / `make dev-down`
- `make build-all`
- `make test-all`
- `make lint-all`
- `make docker-build` / `make docker-push`
- `make k8s-deploy` / `make k8s-rollback` / `make k8s-status`
- `make db-migrate` / `make db-seed` / `make db-backup`

---

## 25. Приложение B: Минимальный чек-лист готовности к production

1. Все секреты вынесены в secret manager.
2. Включены backup и restore проверки.
3. Настроены алерты и дашборды.
4. Проведены нагрузочные тесты на критичных флоу.
5. Подтверждены RTO/RPO.
6. Проверены планы rollback.
7. Проведен security review.
8. Зафиксирован релизный runbook.

---

## 26. Приложение C: Рекомендованный шаблон изменения схемы БД

1. Спроектировать миграцию `up/down`.
2. Проверить backward compatibility.
3. Добавить индексы под реальные запросы.
4. Обновить код репозиториев/моделей.
5. Добавить интеграционные тесты.
6. Обновить документацию таблиц.
7. Прокатить на staging с проверкой планов запросов.
8. Выпустить в production через контролируемый rollout.

---

## 27. Заключение

Текущая структура проекта уже покрывает полный контур платформы технических интервью: от пользовательского входа, анализа резюме и AI-диалога до скоринга, отчетности и аналитики.

Ключевая ценность архитектуры:
- четкая доменная декомпозиция;
- расширяемость за счет микросервисов и event-driven интеграций;
- готовность к эксплуатации в контейнерной и Kubernetes-среде.

Следующий шаг зрелости платформы — формализация единых стандартов контракта, наблюдаемости, безопасности и data governance на уровне всей экосистемы сервисов.

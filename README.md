# RealSync — AI-платформа для технических интервью

> Полная end-to-end платформа для подготовки к техническим собеседованиям с **многоуровневым каскадом LLM**, **гибридной AI-архитектурой** (cloud + custom ML), интеграцией с **GitHub API**, парсингом резюме, **live-coding workspace**, режимом **soft-skills** на собственной обученной модели и интеграцией с **HH.ru** для подбора подходящих вакансий.

---

## 📋 Оглавление

1. [Что это и для кого](#-что-это-и-для-кого)
2. [Ключевые возможности](#-ключевые-возможности)
3. [Технологический стек](#-технологический-стек)
4. [Архитектура системы](#-архитектура-системы)
5. [LLM-каскад: 4-уровневый failover](#-llm-каскад-4-уровневый-failover)
6. [Soft-skills ML модель](#-soft-skills-ml-модель-собственная-pytorch-разработка)
7. [Описание всех микросервисов](#-описание-всех-микросервисов)
8. [Frontend: страницы и архитектура](#-frontend-страницы-и-архитектура)
9. [База данных](#-база-данных)
10. [Поток данных: типовые сценарии](#-поток-данных-типовые-сценарии)
11. [Промпт-инжиниринг и режимы интервью](#-промпт-инжиниринг-и-режимы-интервью)
12. [Безопасность и аутентификация](#-безопасность-и-аутентификация)
13. [Запуск проекта](#-запуск-проекта)
14. [Конфигурация (.env)](#-конфигурация-env)
15. [Тестирование и отладка](#-тестирование-и-отладка)
16. [Структура репозитория](#-структура-репозитория)

---

## 🎯 Что это и для кого

**RealSync** — open-source платформа симулятора технических собеседований. Кандидат загружает резюме / GitHub, выбирает желаемую роль, и платформа в режиме реального чата проводит интервью голосом AI:

- **Theory** — концептуальные вопросы (архитектура, trade-offs, принципы)
- **Practice** — live-coding в встроенном редакторе с подсветкой
- **Soft-skills** — поведенческие вопросы, оценка через собственную PyTorch-модель

После сессии — отчёт с разбором ответов, рекомендациями, экспортом PDF. Резюме автоматически анализируется и матчится с реальными вакансиями HH.ru.

**Целевая аудитория:** разработчики, готовящиеся к смене работы, и компании, желающие развернуть подготовительную платформу для своих кандидатов.

---

## 🚀 Ключевые возможности

| Модуль | Возможности |
|--------|-------------|
| **Интервью (AI)** | Реальный голос интервьюера, адаптивная сложность, per-turn verdict-бейджи, пауза таймера во время мышления AI, семантическая дедупликация вопросов через embeddings |
| **Live-coding** | Встроенный редактор с подсветкой синтаксиса (Go/Python/TS/JS/Java/Rust/C++), Tab-индентация, авто-парные скобки, line-numbers, нумерация символов, AI-проверка решений |
| **Soft-skills** | Bank из 1420 вопросов, оценка ответа собственной PyTorch-моделью (rubert-tiny2 + регрессор) |
| **Резюме (AI)** | Парсинг PDF/DOCX/TXT, AI-инсайты (strengths, weaknesses, action plan), языки программирования, рекомендованные позиции, треки интервью |
| **GitHub-аналитика** | Импорт через публичный API, AI-разбор активности, рекомендованные роли с fit_score, top-репозитории, граф контрибуций |
| **HH.ru интеграция** | Автоматический поиск подходящих вакансий по навыкам резюме, кеш в Redis 1h, фильтр регионов (Беларусь / Россия / Москва / Все) |
| **Отчёты** | Per-session разбор (correctness/clarity/completeness/relevance), PDF-экспорт в editorial-стиле, JSON-экспорт, история сессий |
| **Биллинг** | 4 тарифа (Бесплатный / Стартовый / Профи / Команда), валюта BYN с собственным SVG-знаком нового белорусского рубля, mock checkout |
| **Админ-панель** | Пользователи, подписки, аудит-лог, 6 аналитических диаграмм (роли, тарифы, активность, доход, retention) |
| **Аутентификация** | JWT + bcrypt, refresh tokens, OAuth (Google/GitHub), полная обработка 401/403/409 |
| **i18n** | RU / EN переключатель в профиле |
| **Темы** | Light / Dark / System с editorial-дизайном (Bricolage Grotesque + Instrument Serif + JetBrains Mono) |

---

## 🛠 Технологический стек

### Backend (микросервисная архитектура)
- **Go 1.22+** — user / interview / scoring / report / admin / github / resume / analytics / notification / api-gateway / code-executor
- **Python 3.11+ / FastAPI** — AI-сервисы (ai-service, softskills-service)
- **PyTorch 2.4 + sentence-transformers 3.2** — собственная ML-модель soft-skills (rubert-tiny2 эмбеддинги + кастомный регрессор)
- **gorilla/mux, gorilla/websocket** — HTTP и WS на Go
- **OpenAI SDK (async)** — обёртка для всех OpenAI-совместимых LLM
- **logrus + zerolog** — структурное логирование
- **pq, go-redis** — драйверы БД

### Frontend
- **React 18 + TypeScript 5**
- **Vite 5** — dev server + bundler
- **React Router 6** — роутинг с nested routes
- **Zustand** — стейт-менеджмент (8 stores: auth, user, ui, subscription, preferences, chat, session, timer, network)
- **Axios** — HTTP клиент, 90s timeout под медленный LLM-каскад
- **Recharts** — диаграммы в админке
- **Кастомный CSS** (editorial-стиль RealSync) без Tailwind/SCSS — `tokens.css`, `v3.css`, `pages.css`

### Инфраструктура
- **Docker Compose** — оркестрация 14 сервисов
- **PostgreSQL 16** — основное хранилище (8 БД на сервис)
- **Redis 7** — кеш (embeddings, HH-vacancies, AI provider state)
- **Kafka + Zookeeper** — event streaming (audit-log, analytics)
- **RabbitMQ** — task queue для notifications
- **ClickHouse** — events analytics
- **MinIO (S3-совместимое)** — хранение резюме и аватаров
- **Prometheus + Grafana + Loki + Alertmanager** — наблюдаемость

### AI / ML
- **LLM-каскад из 4+ провайдеров**: Groq → OpenRouter → DeepSeek → Cerebras
- **Custom ML model** (PyTorch) для soft-skills скоринга
- **sentence-transformers/cointegrated/rubert-tiny2** — русскоязычные эмбеддинги (312-dim)
- **Семантическая дедупликация** вопросов через косинусное сходство
- **AI-вердикты** per-turn (correct/partial/wrong/skipped/off_topic)

---

## 🏗 Архитектура системы

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Browser (React + Vite)                          │
│  /workspace · /interview · /reports · /resume · /profile · /admin · ...     │
└────────────────────────────┬────────────────────────────────────────────────┘
                             │ HTTPS + WSS
                             ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│   API Gateway (Go, :8000)                                                   │
│   - JWT auth-middleware                                                     │
│   - Token-bucket rate-limiter (10 RPS, burst 40)                            │
│   - CORS                                                                    │
│   - Proxy + path-rewrite в backend сервисы                                  │
└─────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬───────┘
      │      │      │      │      │      │      │      │      │      │
      ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼
   ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐
   │USER│ │INT │ │SCR │ │REP │ │ADM │ │GH  │ │RES │ │AI  │ │SS  │ │ANL │
   │8080│ │8082│ │8080│ │8080│ │8080│ │8082│ │8080│ │8001│ │8090│ │8080│
   └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘
     │      │      │      │      │      │      │      │      │      │
     └──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┘
            │ shared infrastructure
            ▼
   ┌────────────────────────────────────────────────────────────────┐
   │  PostgreSQL    Redis    Kafka    RabbitMQ   ClickHouse   MinIO │
   └────────────────────────────────────────────────────────────────┘
                                │
                                ▼
   ┌────────────────────────────────────────────────────────────────┐
   │  External:  Groq · OpenRouter · DeepSeek · Cerebras · Gemini   │
   │             · HH.ru API · GitHub API · OpenAI (embeddings)     │
   └────────────────────────────────────────────────────────────────┘
```

**Связи между сервисами:**
- `interview-service` ↔ `ai-service` (HTTP, `/api/v1/interviewer/next-question`)
- `interview-service` ↔ `softskills-service` (HTTP, `/api/v1/score`, `/api/v1/questions`)
- `interview-service` ↔ `scoring-service` (HTTP, `/api/v1/scoring/generate`)
- `interview-service` ↔ `HH.ru` (HTTP, `https://api.hh.ru/vacancies`)
- `interview-service` ↔ `GitHub` (HTTP, `https://api.github.com/...`)
- `interview-service` ↔ `frontend` (WebSocket `/ws/sessions/{id}`)
- `ai-service` ↔ LLM-провайдеры (через AsyncOpenAI с каскадом)

---

## ⚡ LLM-каскад: 4-уровневый failover

В центре AI-логики — собственная реализация **каскадного роутера** (`services/ai-service/src/core/llm_client.py`).

### Архитектура

- **`LLMClient`** — один OpenAI-совместимый клиент с retry на 429
- **`LLMPool`** — round-robin между N ключами одного провайдера (для DeepSeek/OpenRouter с 5 ключами в одном tier)
- **`LLMRouter`** — каскад: пробует Tier 1, при ошибке → Tier 2 → Tier 3 → Tier 4

### Поведение

| Tier | Провайдер по умолчанию | Скорость | Лимит free |
|------|-------------------------|----------|------------|
| **1** | Groq `llama-3.3-70b-versatile` | 1-2 сек | 100k TPD |
| **2** | OpenRouter `meta-llama/llama-3.3-70b-instruct:free` | 3-10 сек | ~50 req/сутки |
| **3** | DeepSeek `deepseek-chat` или пул 5 ключей OpenRouter | 3-5 сек | $5 free кредитов |
| **4** | Cerebras `llama3.1-8b` или Gemini 2.0 Flash | 1-3 сек | 1500 RPD / 14400 RPD |

### Ключевые оптимизации

1. **Fast-fail на дневной квоте** — если `Retry-After > 3s` (например «try again in 55m55s») каскад **не ждёт**, сразу уходит в next tier. Парсер понимает форматы `Xs`, `Xm Y.Ys`.
2. **Round-robin внутри пула** — Tier 3 может содержать 5 ключей одного OpenRouter аккаунта на одну DeepSeek-модель → **5× daily quota**.
3. **404 fast-fail** — если первый ключ пула вернул «model not found», пропускаем оставшиеся 4 (та же модель = тот же 404).
4. **Lazy init** — каскад собирается на первом запросе, не при старте.
5. **Defensive JSON parsing** — coerce работает прозрачно, если LLM вернул `language_insights` массивом строк или `flags` list'ом.

Подробная инструкция: [`docs/LLM_SETUP.md`](docs/LLM_SETUP.md).

---

## 🧠 Soft-skills ML модель (собственная PyTorch разработка)

Отдельный микросервис `services/softskills-service/` — **полностью самостоятельная ML-модель**, тренируется на месте, без зависимости от LLM-каскада.

### Архитектура `SoftSkillRegressor`

```
Input (rubert-tiny2 embedding, 312-dim)
    │
    ├─→ Linear(312 → 192) + LayerNorm + GELU + Dropout(0.35)
    │           │
    │           └────────────────────────┐  (residual)
    │                                    ▼
    └─→ Linear(192 → 96) + LayerNorm + GELU + Dropout(0.20)
                           │
                           ▼
                       Linear(96 → 1) + Sigmoid
                           │
                           ▼
                       Score ∈ [0, 1] → × 100 → %
```

**Особенности vs оригинальная разработка:**
- LayerNorm вместо BatchNorm (стабильнее на one-by-one inference)
- GELU вместо ReLU (smoother gradient)
- Residual connection hidden1 → hidden2
- Kaiming init, Smooth L1 loss (устойчивость к шумным таргетам)
- Cosine annealing LR scheduler + grad clipping

### Датасет

- **Seed:** 644 пар (вопрос, ответ, target ∈ [0,1]) — оригинальная разработка автора
- **Augmentation:** для каждого вопроса добавляются три tier'а синтетических ответов:
  - **Low** (target 0.05-0.18): «Не знаю», «Делаю как сказали»
  - **Mid** (target 0.42-0.58): «Обсуждаю с командой», «Анализирую и принимаю решение»
  - **High** (target 0.85-0.96): STAR-формулировки с конкретными цифрами и метриками
- **Итого ~2000 примеров** после augmentation
- **Bank вопросов:** 1420 уникальных soft-skills вопросов

### Тренировка

- **200 эпох** с early stopping (patience=25)
- **Batch 32, AdamW, lr=8e-4 + cosine annealing**
- **Split:** 80% train / 10% val / 10% test
- На CPU 4 ядра — 5-8 минут
- Результаты: MAE ≈ 0.05, R² ≈ 0.78 (test)

### API сервиса (port 8090)

| Endpoint | Описание |
|---------|----------|
| `GET /health` | Liveness |
| `GET /api/v1/questions?n=5` | Случайные N вопросов из bank'а |
| `POST /api/v1/score` | `{question, answer}` → `{score 0-100, feedback, verdict}` |
| `POST /api/v1/score_batch` | Батч оценок |
| `POST /api/v1/session/score` | Полная оценка сессии с per-turn + рекомендациями |

### Контейнер

- Веса персистятся в volume `platform_softskills_weights` → тренировка только на первом старте
- Embedding модель кешируется в `platform_softskills_hf_cache`
- `start_period: 360s` (даёт время на тренировку)
- `restart: unless-stopped` — всегда онлайн

---

## 🔧 Описание всех микросервисов

### 1. **api-gateway** (Go, :8000)

Единая точка входа для frontend. Stateless.

- JWT-валидация (middleware)
- Token-bucket rate-limiter (10 RPS, burst 40, per IP+user+session)
- CORS preflight
- Path-rewrite: `/api/users/*` → `user-service:8080/api/v1/users/*`
- WebSocket-proxy для `/ws/*`

**Особые маршруты:**
- `/api/v1/resume/import` и `/api/v1/resume/vacancies/*` → **interview-service** (исторически)
- `/api/billing/*` → admin-service
- `/api/softskills/*` → softskills-service

### 2. **user-service** (Go, :8080)

Аутентификация. БД `user_service`.

- `POST /api/v1/auth/register` — регистрация (bcrypt cost 10, JWT issue)
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh` — refresh access-token
- `GET /api/v1/users/me`, `PATCH /api/v1/users/me`
- `POST /api/v1/users/me/password` — смена пароля
- `DELETE /api/v1/users/me`

JWT HS256 с shared secret `JWT_SECRET`.

### 3. **interview-service** (Go, :8082) ⭐ главный сервис

Самый большой сервис (~5500 LOC в `handlers.go`). Управляет lifecycle интервью. БД `interview_service`.

**Главные endpoints:**
- `POST /api/v1/interviews/sessions` — создание сессии
- `GET /api/v1/interviews/sessions/{id}` — получение
- `GET/POST /api/v1/interviews/sessions/{id}/messages` — история / отправка
- `POST /api/v1/interviews/sessions/{id}/finish` — завершение → отчёт
- `GET /api/v1/interviews/sessions/{id}/report` — итоговый отчёт
- `GET /api/v1/interviews/sessions/{id}/ws` — WebSocket real-time

**Дополнительные:**
- `POST /api/v1/github/import` — импорт GitHub-профиля
- `POST /api/v1/resume/import` — парсинг резюме PDF/DOCX/TXT
- `GET /api/v1/resume/history`, `GET /api/v1/resume/history/{id}`
- `GET /api/v1/resume/vacancies/{report_id}?area=16` — HH.ru подбор вакансий
- `GET /api/v1/interviews/my/report` — пользовательская аналитика

**Внутренние особенности:**

#### LLM-роутер
`callAIWithFailover()` — fallback между двумя ai-service URL (primary + secondary).

#### Per-turn AI-вердикты
Каждый ответ → LLM с запросом вердикта (`correct`/`partial`/`wrong`/`skipped`/`off_topic`/`none`) + reason. Сохраняется в `interview_messages.verdict`, транслируется WS для UI-бейджа.

#### Семантическая дедупликация
- `rememberAskedQuestion()` → embedding в Redis с TTL
- `isQuestionRepeated()` → косинусное сходство
- Порог зависит от роли (Backend строже, Management — мягче)

#### Force topic rotation
`TopicStats[topic]++` на каждое AI-сообщение. При `>= 2` → принудительный `nextTopic(role, cursor)`, чтобы AI не разгонял один топик 5 раз.

#### Practice-mode валидация
`isPracticeTaskQuestion()` ловит theory-style формулировки и подменяет на coding-задачу из `practiceFallbackTask()` — bank гарантированных задач по ролям (Backend, Frontend, Data, ML, DevOps, Mobile, Fullstack).

#### Soft-skills bridge
Когда `interview_mode == "softskills"` — `requestNextQuestion()` делает early-return в `buildSoftSkillsNextQuestion()`:
1. Скорит предыдущий ответ через `softskills-service:/api/v1/score`
2. Берёт следующий вопрос из bank'а через `/api/v1/questions?n=8`
3. Избегает повторов

#### Pause timer during AI thinking
Перед запросом к LLM — broadcast `session.timer.adjusted` с pause=true. После ответа — добавляет `aiCallDuration` к `session.ExpiresAt`. Кандидат не теряет время на ожидание AI.

### 4. **ai-service** (Python FastAPI, :8001)

**Brain** платформы. Не хранит данные.

**Endpoints:**
- `POST /api/v1/interviewer/next-question` — главный: вопрос + verdict + reason + topic + difficulty_delta
- `POST /api/v1/interviewer/validate-output` — guardrail (off-topic, leak, repeats)
- `POST /api/v1/interviewer/post-analysis` — финальный разбор
- `POST /api/v1/resume/insights` — анализ резюме
- `POST /api/v1/developer/insights` — анализ GitHub
- `POST /api/v1/analysis/answer` — рубричная оценка ответа
- `POST /api/v1/analysis/similarity` — семантическое сходство
- `POST /api/v1/transcription` — Whisper (заготовка)
- `POST /embeddings/question`, `POST /embeddings/compare`

**Промпт-инжиниринг:**
- Базовый шаблон `INTERVIEWER_TURN_TEMPLATE` со ~30 placeholder'ами
- Mode-specific guidance — жёсткие правила для practice (только coding), theory (только концепции, макс 2 follow-up)
- Style profile per role — backend строгий по архитектуре, frontend — по производительности
- `turn_nonce` UUID в каждом запросе — ломает кеши LLM

### 5. **softskills-service** (Python FastAPI + PyTorch, :8090)

См. раздел [Soft-skills ML модель](#-soft-skills-ml-модель-собственная-pytorch-разработка) выше.

### 6. **scoring-service** (Go, :8080)

Считает финальный отчёт. БД `scoring_service`.

**Endpoint:** `POST /api/v1/scoring/generate`

**Логика:**
- Фильтр: учитываются только substantive ответы (≥6 символов или с verdict от AI)
- Per-answer score: correct→90, partial→55, wrong→20, skipped→0, off_topic→10
- **Correctness** = avg(base) по всем answers
- **Clarity** = avg(clarity_score)
- **Completeness** = clamp(correctness − penalty / n)
- **Relevance** = (n − skipped − off_topic) / n × 100 — доля содержательных ответов
- **Overall** = avg(correctness, clarity, completeness, relevance)

Если нет содержательных ответов — все score=0 + recommendation «попробуйте снова, отвечайте 1-2 предложениями». Никаких фейковых оценок.

### 7. **report-service** (Go, :8080)

История отчётов и аналитика. БД `report_service`. Получает события из Kafka.

### 8. **admin-service** (Go, :8080)

Биллинг + админ-панель. БД `admin_service`.

**Endpoints:**
- `GET /api/v1/admin/users` (с пагинацией, фильтром)
- `POST /api/v1/admin/users/{id}/suspend|activate|ban|role`
- `GET /api/v1/admin/subscriptions`
- `GET /api/v1/admin/audit-logs`
- `GET /api/v1/admin/dashboard/stats` — данные для 6 диаграмм
- `GET /api/v1/billing/me` / `POST /api/v1/billing` / `DELETE /api/v1/billing`

**Тарифы (BYN с новым знаком ₣):**
- Стартовый — **29 Br/мес**
- Профи — **65 Br/мес** (highlighted)
- Команда — **159 Br/мес**

### 9. **github-service** (Go, :8082)
Заглушка. Реальный импорт через interview-service.

### 10. **resume-service** (Go, :8080)
Заглушка. Реальный парсинг через interview-service.

### 11. **analytics-service** (Go, :8080)
ETL: Kafka → ClickHouse. Подписки на dashboards.

### 12. **notification-service** (Go, :8080)
Email + SMS через RabbitMQ. SMTP / SendGrid / Twilio.

### 13. **code-executor-service** (Go)
Заготовка для исполнения кода в sandbox.

---

## 💻 Frontend: страницы и архитектура

### Структура `frontend/src/`

```
app/
  providers/      ThemeProvider, AppProviders
  router/         AppRouter, AppShell, ProtectedRoute
  store/          authStore, userStore, uiStore, subscriptionStore, preferencesStore
features/
  auth/           AuthForm
  github-connect/ GithubConnectCard (полная аналитика)
  interview-module/
    api.ts        Session CRUD
    components/   ChatWindow, MessageList, PracticeCodeWorkspace, ...
    stores/       chatStore, sessionStore, timerStore, networkStore
  upload-resume/  ResumeUploader (dashed dropzone + drag&drop)
pages/
  Auth/           login/register с editorial-дизайном
  Home/           лендинг с parallax «Я», magnetic CTA, datastream
  Workspace/      parent layout с рейлом, Cmd+K палитра команд
    Overview      метрики, недавние интервью, рекомендации, GH grid
  CareerCenter/   AI Career Copilot, Career Radar, Resume Lab
  InterviewSetup/ Vacancy + Mode + Level + Duration, popup-flow
  InterviewSession/ Real-time chat + practice editor + softskills mode
  InterviewResult/ Итоговый отчёт
  Reports/        История + рапорт + PDF/JSON экспорт
  Resume/         Загрузка + анализ + HH-вакансии + треки интервью
  Profile/        Профиль + GitHub + подписка + danger zone
  Admin/          Пользователи + подписки + аудит + 6 диаграмм
  Billing/        External checkout-страница
shared/
  api/            apiClient (axios 90s timeout), per-service modules
  config/         env, queryKeys
  i18n/           ru.ts, en.ts, useTranslation, useLanguageStore
  lib/            currency (formatBYN), cn
  ui/             RsIcon, Sparkline, Counter, Track, Tape, BynSign (SVG)
styles/
  realsync/
    tokens.css    дизайн-токены: цвета oklch, шрифты, шкалы
    v3.css        тренды: giant-letter, brutal, glitch, expr-headline
    pages.css     стили страниц, IDE-редактор, адаптив
widgets/
  navbar/         Topbar
```

### Дизайн-система RealSync

**Шрифты:**
- **Bricolage Grotesque** (12-96 opsz) — заголовки
- **Instrument Serif italic** — акцентные слова
- **JetBrains Mono** — sysbar, моно-лейблы, код
- **Geist** — body

**Цвета (oklch):**
- `--bg: oklch(0.974 0.006 85)` — тёплая бумага (light)
- `--ink: oklch(0.18 0.012 60)` — ink-чёрный (light) / бумага (dark)
- `--accent: oklch(0.84 0.18 130)` — лайм
- `--accent-2: oklch(0.78 0.14 50)` — амбер
- `--signal: oklch(0.65 0.14 25)` — коралл

**Компоненты:**
- `.btn` варианты с hover через `color-mix(in oklch, ...)` — корректный контраст на обеих темах
- `.profile-card` / `.gh-card` — карточки с rounded
- `.sysbar` — pill из моно-чипов k/v с pulse-dot
- `.eyebrow` — моно-капс с лаймовой точкой
- `.expr-headline` — микс bold + italic + light + underline + pill
- `.brutal` — 2px ink-рамка + 5px жёсткая тень
- `.metric-row` — три равные колонки с Counter
- `.dash-rail` — sticky sidebar

**Адаптив:** breakpoints 1024 / 880 / 640 / 380, touch-pointer для 44px+ tap targets.

**Command palette (Cmd+K):** в Workspace, 8 команд (Обзор / Карьерный / Профиль / Резюме / Админ / Новое интервью / Отчёты / Главная).

---

## 🗄 База данных

PostgreSQL 16, **отдельная БД на сервис** — изоляция, никаких cross-service JOIN'ов. 8 БД на одной инстанции.

### Ключевые таблицы interview_service

#### `interview_sessions`
- `id` UUID PK, `user_id` UUID
- `role` VARCHAR (Backend/Frontend/.../SoftSkills)
- `level` VARCHAR (Junior/Middle/Senior)
- `status` (active/finished/expired)
- `metadata` JSONB (interview_mode, vacancy_title, focus_areas, primary_skills)
- `started_at`, `ends_at`, `created_at`

#### `interview_messages`
- `id` UUID, `session_id` UUID FK
- `sender` (ai/user), `content` TEXT
- `topic` VARCHAR, `difficulty` INT
- `verdict` (correct/partial/wrong/skipped/off_topic/none)
- `verdict_reason` TEXT

#### `interview_reports`
- `session_id` UUID UNIQUE
- `correctness`, `clarity`, `completeness`, `relevance`, `overall_score` FLOAT
- `strengths`, `weaknesses`, `recommendations` JSONB[]

#### `resume_imports`
- `report_id`, `user_id` UUID
- `file_name`, `file_size`, `content_type`
- `stats` JSONB, `extracted_skills` TEXT[]
- `ai_insights` JSONB (summary, strong/improvement_points, action_plan, language_insights, interview_tracks, recommended_positions)

### Кеши Redis

- `interview:embedding:cache:{question}` — embedding для дедупликации
- `interview:asked:{user}:{role}` — заданные вопросы
- `hh:vacancies:{area}:{queryhash}` — HH.ru (TTL 1h)
- `auth:refresh:{token}` — refresh tokens

---

## 🔄 Поток данных: типовые сценарии

### Сценарий 1: Запуск интервью (practice mode)

```
1. UI · POST /api/interviews/sessions   { role, level, mode: "practice", ... }
2. interview-service создаёт сессию, возвращает sessionId + wsUrl
3. UI открывает popup /interview/session/{id}
4. Popup · GET .../sessions/{id} + /messages
5. Popup · WS /ws/sessions/{id}
6. interview-service · generateInitialQuestion async
   ├─ buildIntroQuestion(session)
   ├─ если mode=practice → buildPracticeTaskQuestion
   │   └─ requestInterviewQuestionFromAI → ai-service
   │      └─ LLMRouter.generate_json
   │         ├─ Tier 1 Groq → 200 OK или fail-fast 429
   │         ├─ Tier 2 OpenRouter
   │         └─ Tier 3/4 ...
   ├─ isPracticeTaskQuestion(out) → true/false
   │   └─ if false: practiceFallbackTask(session)
   └─ broadcast "ai.typing.stopped" + "message.ai"
7. UI рендерит первое сообщение
```

### Сценарий 2: Ход интервью (user → AI с verdict)

```
1. UI · POST /api/interviews/sessions/{id}/messages  { content }
2. interview-service:
   a) broadcast "message.user" + "ai.typing.started"
   b) processUserMessage:
      ├─ classifyAnswer → "strong"/"weak"/"empty"
      ├─ detectCandidateIntent → "skip"/"clarify"/"answer"
      ├─ requestNextQuestion(session, content)
      │   ├─ если softskills → buildSoftSkillsNextQuestion (короткий путь)
      │   └─ иначе → callAIWithFailover → ai-service
      ├─ TopicStats[topic] += 1
      ├─ если TopicStats[topic] >= 2 → force-rotate через nextTopic()
      └─ applyAnswerSignalToResponse (только non-softskills)
   c) broadcast "verdict.applied" + "message.ai"
   d) broadcast "session.timer.adjusted" (+aiCallDuration)
3. UI ставит verdict-бейдж + новый AI-вопрос
```

### Сценарий 3: Завершение и отчёт

```
1. UI · POST /api/interviews/sessions/{id}/finish
2. interview-service:
   a) status = "finished"
   b) requestScoringReport → scoring-service
      └─ вычисляет 4 метрики + strengths/weaknesses/recommendations
   c) сохраняет в interview_reports
   d) broadcast "session.finished" + report
3. UI редиректит на /interview/result/{sessionId}
```

### Сценарий 4: Парсинг резюме

```
1. UI · POST /api/resume/import (multipart)
2. interview-service:
   a) ParseResume(PDF/DOCX/TXT) → text + metadata
   b) Извлечь навыки регулярками + AI
   c) POST ai-service /api/v1/resume/insights
      └─ LLMRouter.generate_json(RESUME_INSIGHTS_SCHEMA)
   d) Defensive parsing (coerce кривого JSON)
   e) Save to resume_imports
3. UI рендерит:
   ├─ readiness 0-100%
   ├─ score by факторы
   ├─ языки программирования (фильтр: только реальные ЯП)
   ├─ skills bars (55-95% нормализация)
   ├─ action plan
   └─ HH-вакансии (отдельный запрос)
```

### Сценарий 5: HH.ru вакансии

```
1. UI · GET /api/resume/vacancies/{report_id}?area=16
2. interview-service:
   a) buildHHQuery(role, skills) → «Backend AND (Go OR PostgreSQL OR gRPC)»
   b) Redis cache check (TTL 1h)
   c) miss → GET https://api.hh.ru/vacancies?text=...&area=16
      └─ User-Agent: "RealSync-Interview-Platform/1.0 (...)" обязателен
   d) Нормализация (40+ полей → 12 нужных)
   e) Save to Redis
3. UI рендерит карточки с salary, employer, snippet, match-score
```

### Сценарий 6: Soft-skills интервью

```
1. UI · POST /api/interviews/sessions (mode="softskills")
2. interview-service сохраняет: role="SoftSkills", mode="softskills"
3. UI popup → WS connect
4. generateInitialQuestion:
   └─ buildSoftSkillsNextQuestion (LLM не вызывается)
       ├─ GET softskills-service:/api/v1/questions?n=8
       │   ├─ Скип уже заданных (по session.Messages)
       │   └─ Берёт первый свободный
       └─ Возвращает чистый вопрос
5. Каждый ответ:
   a) UI · POST /messages
   b) buildSoftSkillsNextQuestion:
      ├─ POST softskills-service:/api/v1/score { question, answer }
      │   ├─ rubert-tiny2 embedding
      │   ├─ Scaler transform
      │   ├─ SoftSkillRegressor.forward
      │   └─ → { score 0-100, feedback, verdict }
      └─ Берёт следующий вопрос
```

---

## 🎨 Промпт-инжиниринг и режимы интервью

### Режим THEORY (строгие правила)
- Только концептуальные вопросы (trade-offs, архитектура, принципы)
- **ЗАПРЕЩЕНЫ** coding-задачи
- Максимум **1-2 follow-up** на тему, потом обязательная смена
- После 2 follow-up'ов backend принудительно ротирует topic

### Режим PRACTICE (live-coding)
- **ОБЯЗАТЕЛЬНО** конкретные coding-задачи: «Напишите функцию X», «SQL-запрос», «bash-скрипт», «middleware», «handler»
- **ЗАПРЕЩЕНЫ** theory-style: «Какой подход бы использовали?», «Расскажите как...»
- Validator `isPracticeTaskQuestion`:
  - Reject prefix'ы: «Какой подход», «Расскажите как», «Опишите архитектуру», «В чём разница», «Что такое»
  - Accept markers: «напишите», «реализ», «sql-запрос», «bash-скрипт», «regex», «curl», «middleware», «handler», «сигнатур»
- Если LLM упорно возвращает theory — fallback на bank coding-задач (`practiceFallbackTask`)
- После прислания кода — **макс 1-2 follow-up** про конкретный код

### Режим SOFTSKILLS
- LLM **не используется** вообще
- Вопросы из bank'а 1420 вопросов (softskills-service)
- Оценка — ML-модель (rubert-tiny2 + регрессор)
- Никаких follow-up'ов — каждый ход новый вопрос

---

## 🔐 Безопасность и аутентификация

- **JWT HS256** с `JWT_SECRET` (24h access, 7d refresh)
- **bcrypt cost 10** для паролей
- **OAuth Google + GitHub** (заготовлены)
- **Rate-limiting** через token-bucket
- **CORS** для `http://localhost:3000`
- **XSS-защита** в PDF-экспорте через `escapeHtml`
- **Bearer tokens** (CSRF не нужен)
- **Censorship filter** на запросы кандидата
- **Guardrail** AI-ответов:
  - leak-detection
  - off-topic detection
  - повторов
  - запрещённой лексики

---

## 🚀 Запуск проекта

### Требования
- **Docker Desktop** или Docker Engine 24+
- **Docker Compose v2**
- **Make**
- **8GB+ свободной RAM**

### Быстрый старт

```bash
# 1. Клонировать
git clone https://github.com/BogdanSadovski/saou_nic.git
cd saou_nic

# 2. Настроить .env
# Минимум LLM_API_KEY (Groq) — обязателен, остальные опционально
# Подробно см. docs/LLM_SETUP.md

# 3. Поднять всё
make dev-up

# 4. Дождаться первичной тренировки softskills (5-8 мин)
docker logs -f platform-softskills-service-1
# Должно появиться: "Application startup complete"

# 5. Frontend
cd frontend
npm install
npm run dev   # → http://localhost:3000
```

### Полное переобучение softskills модели

```bash
docker compose -f infrastructure/docker/docker-compose.yml down softskills-service
docker volume rm platform_softskills_weights
docker compose -f infrastructure/docker/docker-compose.yml up -d softskills-service
docker logs -f platform-softskills-service-1
```

### Доступы по умолчанию

- Frontend: **http://localhost:3000**
- API Gateway: **http://localhost:8000**
- AI-service: **http://localhost:3006**
- Soft-skills service: **http://localhost:3012**
- Postgres: **localhost:5433**
- Redis: **localhost:6379**
- RabbitMQ Management: **http://localhost:15672**
- Grafana: **http://localhost:3000** (admin/admin_change_me)
- Prometheus: **http://localhost:9090**

---

## ⚙ Конфигурация (.env)

Полный пример `infrastructure/docker/.env`:

```bash
# General
NODE_ENV=development
LOG_LEVEL=info

# PostgreSQL
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres_secret

# Redis
REDIS_PASSWORD=your_redis_secret

# JWT
JWT_SECRET=your-very-long-random-secret-min-64-chars
JWT_EXPIRES_IN=24h
JWT_REFRESH_EXPIRES_IN=7d

# OAuth (опционально)
OAUTH_GOOGLE_CLIENT_ID=...
OAUTH_GITHUB_CLIENT_ID=...
GITHUB_TOKEN=ghp_...

# LLM Tier 1 — Groq (обязательно)
LLM_API_KEY=gsk_...
LLM_BASE_URL=https://api.groq.com/openai/v1
LLM_MODEL=llama-3.3-70b-versatile

# LLM Tier 2 — OpenRouter (опционально)
LLM_SECONDARY_API_KEY=sk-or-v1-...
LLM_SECONDARY_BASE_URL=https://openrouter.ai/api/v1
LLM_SECONDARY_MODEL=meta-llama/llama-3.3-70b-instruct:free

# LLM Tier 3 — пул из 5 ключей (опционально)
LLM_TERTIARY_API_KEYS=key1,key2,key3,key4,key5
LLM_TERTIARY_BASE_URL=https://openrouter.ai/api/v1
LLM_TERTIARY_MODEL=deepseek/deepseek-r1:free

# LLM Tier 4 — Cerebras (опционально)
LLM_QUATERNARY_API_KEY=csk-...
LLM_QUATERNARY_BASE_URL=https://api.cerebras.ai/v1
LLM_QUATERNARY_MODEL=llama3.1-8b

# RabbitMQ / ClickHouse / S3 / SMTP / Grafana — см. infrastructure/docker/.env
```

### Где получить LLM-ключи

- **Groq:** https://console.groq.com/keys (бесплатно, ~100k TPD)
- **OpenRouter:** https://openrouter.ai/keys (можно создать 5+ ключей в одном аккаунте)
- **DeepSeek:** https://platform.deepseek.com/api_keys ($5 free credits)
- **Cerebras:** https://cloud.cerebras.ai/platform → API Keys (бесплатно, 14400 RPD)
- **Google Gemini:** https://aistudio.google.com/apikey (бесплатно, 1500 RPD)

---

## 🧪 Тестирование и отладка

### Health-checks

```bash
curl http://localhost:8000/api/health
curl http://localhost:3006/health
curl http://localhost:3012/health
```

### Прямой тест AI-каскада

```bash
docker exec platform-ai-service-1 curl -s -X POST \
  http://localhost:8001/api/v1/interviewer/next-question \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test","role":"Backend","level":"Middle","interview_mode":"theory","topic":"databases","difficulty":5,"pressure":3,"time_left_sec":600,"questions_left":5,"turn_nonce":"test1","messages":[]}'
```

### Проверка собранного каскада

```bash
docker exec platform-ai-service-1 python -c "
from src.api.dependencies import DIContainer
c = DIContainer()
client = c.get_llm_client()
print('TYPE:', type(client).__name__)
if hasattr(client, '_clients'):
    for i, t in enumerate(client._clients, 1):
        size = getattr(t, 'size', 1)
        print(f'  Tier {i}: {t.model} (size={size})')
"
```

### Live-логи

```bash
docker logs -f platform-interview-service-1 | grep -E "softskills|cascade|next-question"
docker logs -f platform-ai-service-1 | grep -E "Tier|cascade|rate-limit"
docker logs -f platform-softskills-service-1
```

### Прямой тест soft-skills

```bash
curl -X POST http://localhost:3012/api/v1/score \
  -H "Content-Type: application/json" \
  -d '{"question":"Как вы справляетесь с дедлайнами?","answer":"На прошлом проекте ввёл daily-стендапы по 15 минут — выпустили в срок и сократили блокеры на 40%."}'
# Ожидаемое: score ~85+, verdict=correct
```

### Frontend TypeScript check

```bash
cd frontend
npx tsc --noEmit
```

---

## 📁 Структура репозитория

```
real_ass/
├── README.md                          ← этот файл
├── LICENSE
├── Makefile                           ← dev-up / dev-down / build-all
├── docs/
│   ├── LLM_SETUP.md                   ← подробная инструкция по LLM-ключам
│   ├── PROJECT_FULL_DOCUMENTATION.md
│   ├── api/
│   ├── architecture/
│   ├── deployment/
│   └── development/
├── scripts/
│   ├── run-migrations.sh              ← с фильтром ClickHouse-файлов
│   └── set-llm-key.sh
├── infrastructure/
│   ├── docker/
│   │   ├── docker-compose.yml         ← главный production-stack
│   │   ├── docker-compose.monitoring.yml
│   │   ├── .env                       ← gitignored, см. пример выше
│   │   └── init-scripts/
│   ├── kubernetes/                    ← манифесты (опционально)
│   └── terraform/                     ← IaC (опционально)
├── services/
│   ├── api-gateway/                   ← Go, единый ingress
│   ├── user-service/                  ← Go, auth+JWT
│   ├── interview-service/             ← Go ⭐ главный (handlers.go ~5500 LOC)
│   ├── scoring-service/               ← Go, финальный отчёт
│   ├── report-service/                ← Go, история сессий
│   ├── admin-service/                 ← Go, биллинг + админка
│   ├── github-service/                ← Go, заглушка
│   ├── resume-service/                ← Go, заглушка
│   ├── analytics-service/             ← Go, Kafka → ClickHouse ETL
│   ├── notification-service/          ← Go, email + SMS
│   ├── code-executor-service/         ← Go, sandbox (заготовка)
│   ├── ai-service/                    ← Python FastAPI ⭐ LLM-каскад
│   │   ├── src/
│   │   │   ├── main.py
│   │   │   ├── config.py              ← LLMTier* env-vars
│   │   │   ├── api/
│   │   │   │   ├── routes.py          ← 9 endpoint'ов
│   │   │   │   └── dependencies.py    ← DI с LLMRouter/LLMPool
│   │   │   ├── core/
│   │   │   │   ├── llm_client.py      ← LLMClient + LLMPool + LLMRouter
│   │   │   │   ├── embeddings.py
│   │   │   │   └── prompt_templates.py
│   │   │   └── services/
│   │   ├── requirements.txt
│   │   └── Dockerfile
│   └── softskills-service/            ← Python FastAPI + PyTorch ⭐ собственная ML
│       ├── app/
│       │   ├── main.py                ← FastAPI с /score, /questions
│       │   ├── model.py               ← SoftSkillRegressor
│       │   ├── dataset.py             ← seed + augmentation
│       │   ├── train.py               ← 200 эпох, early stopping
│       │   └── predict.py             ← lazy load
│       ├── data/
│       │   ├── dataset_seed.json      ← 644 примеров от автора
│       │   └── questions_pool.json    ← 1420 вопросов
│       ├── weights/                   ← persisted Docker volume
│       ├── entrypoint.sh
│       ├── requirements.txt
│       └── Dockerfile
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── app/                       ← providers, router, stores
│   │   ├── features/                  ← auth, github, interview-module, upload
│   │   ├── pages/                     ← 15 страниц
│   │   ├── shared/                    ← api, i18n, lib, ui
│   │   ├── styles/realsync/           ← tokens.css, v3.css, pages.css
│   │   └── widgets/                   ← navbar, sidebar
│   └── public/
├── shared/                            ← protobuf-схемы, общие types
└── tests/                             ← e2e (Playwright) + unit
```

---

## 🤝 Лицензия

MIT — см. [`LICENSE`](LICENSE).

---

## 👤 Автор

**Богдан Садовский** — выпускная квалификационная работа, 2026.

Платформа разработана как часть дипломного проекта. Включает:
- Полноценную микросервисную backend архитектуру на Go + Python
- Frontend с собственной дизайн-системой RealSync
- Инновационный 4-уровневый LLM-каскад с failover и round-robin пулом
- Собственную обученную ML-модель для soft-skills оценки
- Интеграцию с реальными внешними API (GitHub, HH.ru, 5+ LLM-провайдеров)

Если вы используете этот код или часть его — ссылка обязательна.

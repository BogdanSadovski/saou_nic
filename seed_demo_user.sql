-- =====================================================================
-- Сидер для demo_user_test@realsnc.com (синтетика для демо).
-- Запускать на сервере:
--   docker compose exec -T postgres psql -U postgres < seed_demo_user.sql
--
-- Все вставки идемпотентные (ON CONFLICT DO NOTHING / IF NOT EXISTS).
-- Безопасно гонять несколько раз — дубликаты не появятся.
-- =====================================================================

\set ON_ERROR_STOP off

-- ──────────────────────────────────────────────────────────────────────
-- 1) USER PROFILE
-- ──────────────────────────────────────────────────────────────────────
\c user_service

DO $$
DECLARE
  v_user_id UUID;
BEGIN
  SELECT id INTO v_user_id FROM users WHERE email = 'demo_user_test@realsnc.com';
  IF v_user_id IS NULL THEN
    RAISE NOTICE 'demo_user_test@realsnc.com not found — skipping seed';
    RETURN;
  END IF;

  UPDATE users SET
    first_name     = 'Богдан',
    last_name      = 'Демо',
    avatar_url     = 'https://i.pravatar.cc/300?u=demo',
    role           = 'admin',
    status         = 'active',
    email_verified = true,
    phone          = '+375 29 555-12-34',
    github_url     = 'https://github.com/BogdanSadovski',
    github_profile = 'BogdanSadovski',
    bio            = 'Backend-инженер с фокусом на Go, распределённые системы и AI-интеграции. Готовлюсь к собеседованиям через RealSync.',
    location       = 'Минск, Беларусь',
    last_login_at  = NOW()
  WHERE id = v_user_id;

  RAISE NOTICE 'User profile updated: %', v_user_id;
END $$;

-- ──────────────────────────────────────────────────────────────────────
-- 2) TELEGRAM LINK (без живого chat_id, чтобы push не уходил вживую)
-- ──────────────────────────────────────────────────────────────────────
INSERT INTO telegram_links (user_id, chat_id, tg_username, notifications_paused, daily_push_hour_utc, linked_at)
SELECT u.id, 100000000 + (random()*1000000)::bigint, '@bsadovski_demo', true, 6, NOW() - interval '14 days'
FROM users u WHERE u.email = 'demo_user_test@realsnc.com'
ON CONFLICT (user_id) DO UPDATE
SET tg_username = EXCLUDED.tg_username,
    linked_at   = EXCLUDED.linked_at;

-- ──────────────────────────────────────────────────────────────────────
-- 3) SUBSCRIPTION — уже была создана через UI; добиваем «исторические» поля
-- ──────────────────────────────────────────────────────────────────────
UPDATE subscriptions s
SET start_date    = NOW() - interval '20 days',
    end_date      = NOW() + interval '345 days',
    auto_renew    = true,
    is_active     = true,
    amount_cents  = 9990,
    currency      = 'BYN',
    card_last4    = '4242',
    metadata      = jsonb_build_object('source','demo_seed','tier','platinum_demo')
FROM users u
WHERE s.user_id = u.id AND u.email = 'demo_user_test@realsnc.com';

-- ──────────────────────────────────────────────────────────────────────
-- 4) PAYMENT TRANSACTIONS (история биллинга)
-- ──────────────────────────────────────────────────────────────────────
INSERT INTO payment_transactions (id, subscription_id, amount_cents, currency, status, created_at)
SELECT gen_random_uuid(), s.id, 9990, 'BYN', 'succeeded', NOW() - interval '20 days'
FROM subscriptions s
JOIN users u ON u.id = s.user_id
WHERE u.email = 'demo_user_test@realsnc.com'
  AND NOT EXISTS (SELECT 1 FROM payment_transactions pt WHERE pt.subscription_id = s.id);

-- ──────────────────────────────────────────────────────────────────────
-- 5) RESUME (резюме + распарсенные данные)
-- ──────────────────────────────────────────────────────────────────────
\c resume_service

INSERT INTO resumes (user_id, file_name, file_url, content_type, status,
                    first_name, last_name, email, phone,
                    summary, skills, experience, education, languages, certifications, metadata)
SELECT
  'cf42ec6c-949f-46f0-bf62-1c73b000e083',
  'Bogdan_Sadovski_Backend.pdf',
  '/storage/resumes/demo/Bogdan_Sadovski_Backend.pdf',
  'application/pdf',
  'completed',
  'Богдан', 'Садовский',
  'demo_user_test@realsnc.com',
  '+375 29 555-12-34',
  'Backend-разработчик с 3+ годами опыта в Go и Python. Строил микросервисы под нагрузкой 10k+ RPS, проектировал event-driven архитектуры на Kafka, оптимизировал PostgreSQL для аналитических запросов.',
  '["Go", "Python", "PostgreSQL", "Kafka", "Redis", "Docker", "Kubernetes", "gRPC", "REST", "WebSocket", "AWS", "Terraform", "CI/CD"]'::jsonb,
  '[
    {"company":"FinTech Lab","position":"Backend Engineer","start":"2024-02","end":"present","description":"Микросервисы оплаты на Go (gRPC + Kafka), нагрузка 8k RPS, p99 < 80ms. Лидил миграцию платёжного шлюза с REST на gRPC, снизив latency на 35%."},
    {"company":"GreenWave","position":"Backend Engineer","start":"2022-06","end":"2024-01","description":"Service-mesh на 14 микросервисов, Postgres + Redis cluster, наблюдаемость через Prometheus/Grafana/Loki. Поднял uptime с 99.5% до 99.95%."},
    {"company":"StartHub","position":"Junior Backend","start":"2021-09","end":"2022-05","description":"Python/Django backend для маркетплейса, integration with Stripe, REST API."}
  ]'::jsonb,
  '[
    {"degree":"Высшее","university":"Минский колледж предпринимательства","specialty":"Прикладная информатика","years":"2022–2026","gpa":"4.6"}
  ]'::jsonb,
  '["Русский (родной)", "Английский (B2)", "Польский (A2)"]'::jsonb,
  '[
    {"name":"AWS Certified Solutions Architect Associate","issuer":"Amazon","year":2024},
    {"name":"Kubernetes Application Developer (CKAD)","issuer":"CNCF","year":2024}
  ]'::jsonb,
  jsonb_build_object('source','demo_seed','word_count', 716, 'pages', 2)
WHERE NOT EXISTS (
  SELECT 1 FROM resumes WHERE user_id='cf42ec6c-949f-46f0-bf62-1c73b000e083'
                       AND file_name='Bogdan_Sadovski_Backend.pdf'
);

-- ──────────────────────────────────────────────────────────────────────
-- 6) INTERVIEW SESSIONS + MESSAGES + REPORTS
-- ──────────────────────────────────────────────────────────────────────
\c interview_service

DO $$
DECLARE
  v_uid UUID := 'cf42ec6c-949f-46f0-bf62-1c73b000e083';
  v_s1 UUID; v_s2 UUID; v_s3 UUID;
BEGIN

  -- === Сессия 1: theory backend middle, finished, отчёт 76 ===
  IF NOT EXISTS (SELECT 1 FROM interview_sessions
                 WHERE user_id=v_uid AND metadata->>'demo_tag'='theory_backend_1') THEN
    v_s1 := gen_random_uuid();
    INSERT INTO interview_sessions (id, user_id, role, level, status, current_topic,
      difficulty_score, pressure_level, question_count, question_limit,
      started_at, ended_at, duration_seconds, metadata)
    VALUES (v_s1, v_uid, 'backend', 'middle', 'finished', 'caching',
      6, 2, 8, 10,
      NOW() - interval '12 days',
      NOW() - interval '12 days' + interval '32 minutes',
      1920,
      jsonb_build_object('demo_tag','theory_backend_1','mode','theory','vacancy_title','Backend Engineer (Go)'));

    INSERT INTO interview_messages (session_id, sender, content, topic, difficulty, created_at) VALUES
      (v_s1, 'ai',   'Расскажите, как вы спроектировали бы кеширующий слой для высоконагруженного API. Какие техники инвалидации применили бы?', 'caching', 6, NOW() - interval '12 days' + interval '1 min'),
      (v_s1, 'user', 'Использовал бы двухуровневый кеш: in-process LRU + Redis cluster. Инвалидация — through-write для горячих ключей и TTL для остального.', 'caching', 6, NOW() - interval '12 days' + interval '4 min'),
      (v_s1, 'ai',   'Хороший ответ. А если у вас сильная throughput-нагрузка на инвалидацию (мутации в секунду больше чем чтения)? Как избежите cache stampede?', 'caching', 7, NOW() - interval '12 days' + interval '5 min'),
      (v_s1, 'user', 'Использовал бы singleflight на бэкенде, чтобы только один воркер обновлял ключ. Плюс jitter в TTL и probabilistic refresh за N% до истечения.', 'caching', 7, NOW() - interval '12 days' + interval '8 min'),
      (v_s1, 'ai',   'Отлично. Перейдём к транзакциям. Объясните разницу между Read Committed и Repeatable Read в PostgreSQL.', 'transactions', 6, NOW() - interval '12 days' + interval '10 min'),
      (v_s1, 'user', 'RC видит данные, закоммиченные на момент каждого запроса (могут быть non-repeatable reads). RR делает snapshot на начало транзакции — повторные чтения дают тот же результат.', 'transactions', 6, NOW() - interval '12 days' + interval '14 min'),
      (v_s1, 'ai',   'А как Postgres защищается от phantom reads на уровне RR?', 'transactions', 7, NOW() - interval '12 days' + interval '15 min'),
      (v_s1, 'user', 'В Postgres RR использует SSI-подобный snapshot, который защищает от phantom через MVCC. Полноценный SI достигается через Serializable.', 'transactions', 7, NOW() - interval '12 days' + interval '19 min');

    INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score,
      strengths, weaknesses, recommendations, generated_at)
    VALUES (v_s1, 84.0, 78.0, 72.0, 81.0, 78.75,
      ARRAY[
        jsonb_build_object('title','Сильное знание кеширования','detail','Назвал singleflight, jitter, probabilistic refresh — это уровень senior.'),
        jsonb_build_object('title','Чёткая структура ответа','detail','Идёт от high-level описания к конкретным техникам.')
      ]::jsonb[],
      ARRAY[
        jsonb_build_object('title','Поверхностно про SSI','detail','Снял ответ на phantom reads общими фразами, не объяснил детально predicate locks.')
      ]::jsonb[],
      ARRAY[
        jsonb_build_object('title','Углубить isolation levels','detail','Прочитать «PostgreSQL Internals», главы про MVCC и SSI.'),
        jsonb_build_object('title','Подготовиться к performance-вопросам','detail','Решить 5–7 задач на explain analyze в PostgreSQL.')
      ]::jsonb[],
      NOW() - interval '12 days' + interval '33 minutes');
    RAISE NOTICE 'Created theory session %', v_s1;
  END IF;

  -- === Сессия 2: practice backend senior, finished, отчёт 86 ===
  IF NOT EXISTS (SELECT 1 FROM interview_sessions
                 WHERE user_id=v_uid AND metadata->>'demo_tag'='practice_go_1') THEN
    v_s2 := gen_random_uuid();
    INSERT INTO interview_sessions (id, user_id, role, level, status, current_topic,
      difficulty_score, pressure_level, question_count, question_limit,
      started_at, ended_at, duration_seconds, metadata)
    VALUES (v_s2, v_uid, 'backend', 'senior', 'finished', 'live_coding',
      8, 3, 4, 5,
      NOW() - interval '6 days',
      NOW() - interval '6 days' + interval '46 minutes',
      2760,
      jsonb_build_object('demo_tag','practice_go_1','mode','practice','vacancy_title','Senior Go Backend'));

    INSERT INTO interview_messages (session_id, sender, content, topic, difficulty, created_at) VALUES
      (v_s2, 'ai',   '[FEEDBACK]\nНачинаем live-coding. Покажете уровень владения Go.\n[HINT]\nДумайте об инвариантах, не о тестах.\n[NEXT_TASK]\nРеализуйте на Go функцию, которая принимает список задач и возвращает порядок выполнения, удовлетворяющий зависимостям и отсортированный по приоритету.', 'live_coding', 7, NOW() - interval '6 days' + interval '1 min'),
      (v_s2, 'user', 'package main\n\nimport "sort"\n\ntype Task struct { ID string; Priority int; Dependencies []string }\n\nfunc Schedule(tasks []Task) ([]string, error) { /* topo sort + priority */ }', 'live_coding', 7, NOW() - interval '6 days' + interval '14 min'),
      (v_s2, 'ai',   '[FEEDBACK]\nКаркас правильный, но в текущем виде вы рискуете нарушить детерминизм при равных приоритетах.\n[HINT]\nДобавьте stable order по lex(ID).\n[NEXT_TASK]\nДопишите детект циклов и верните типизированный ErrCycle.', 'live_coding', 8, NOW() - interval '6 days' + interval '16 min'),
      (v_s2, 'user', 'Добавил detectCycle через DFS с marker-state. Возвращаю errors.New("cycle") — заменяю на var ErrCycle = errors.New("topological cycle").', 'live_coding', 8, NOW() - interval '6 days' + interval '36 min');

    INSERT INTO code_submissions (session_id, user_id, language, code, submission_sequence) VALUES
      (v_s2, v_uid, 'go',
       'package main\n\nimport (\n  "errors"\n  "sort"\n)\n\nvar ErrCycle = errors.New("topological cycle")\n\ntype Task struct {\n  ID           string\n  Priority     int\n  Dependencies []string\n}\n\nfunc Schedule(tasks []Task) ([]string, error) {\n  byID := make(map[string]Task, len(tasks))\n  for _, t := range tasks { byID[t.ID] = t }\n\n  color := map[string]int{}\n  order := []string{}\n  var dfs func(id string) error\n  dfs = func(id string) error {\n    switch color[id] {\n    case 1: return ErrCycle\n    case 2: return nil\n    }\n    color[id] = 1\n    for _, d := range byID[id].Dependencies {\n      if err := dfs(d); err != nil { return err }\n    }\n    color[id] = 2\n    order = append(order, id)\n    return nil\n  }\n  ids := make([]string, 0, len(tasks))\n  for id := range byID { ids = append(ids, id) }\n  sort.Strings(ids)\n  for _, id := range ids { if err := dfs(id); err != nil { return nil, err } }\n  sort.SliceStable(order, func(i, j int) bool { return byID[order[i]].Priority < byID[order[j]].Priority })\n  return order, nil\n}',
       1);

    INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score,
      strengths, weaknesses, recommendations, generated_at)
    VALUES (v_s2, 90.0, 84.0, 82.0, 90.0, 86.5,
      ARRAY[
        jsonb_build_object('title','Чистый Go-стиль','detail','Идиоматичный DFS, типизированная ошибка ErrCycle, stable sort.'),
        jsonb_build_object('title','Корректный детект циклов','detail','Использовал color-states (0/1/2), правильная семантика.')
      ]::jsonb[],
      ARRAY[
        jsonb_build_object('title','Большие рекурсивные стеки','detail','Не упомянул возможность iterative-варианта для очень глубоких DAG.')
      ]::jsonb[],
      ARRAY[
        jsonb_build_object('title','Покрыть edge cases','detail','Добавить unit-тесты на: пустой список, self-loop, дубли ID.')
      ]::jsonb[],
      NOW() - interval '6 days' + interval '47 minutes');
    RAISE NOTICE 'Created practice session %', v_s2;
  END IF;

  -- === Сессия 3: theory active (в процессе) ===
  IF NOT EXISTS (SELECT 1 FROM interview_sessions
                 WHERE user_id=v_uid AND metadata->>'demo_tag'='active_session') THEN
    v_s3 := gen_random_uuid();
    INSERT INTO interview_sessions (id, user_id, role, level, status, current_topic,
      difficulty_score, pressure_level, question_count, question_limit,
      started_at, duration_seconds, metadata)
    VALUES (v_s3, v_uid, 'backend', 'middle', 'active', 'observability',
      5, 2, 3, 10,
      NOW() - interval '2 hours',
      1800,
      jsonb_build_object('demo_tag','active_session','mode','theory','vacancy_title','Backend Engineer'));

    INSERT INTO interview_messages (session_id, sender, content, topic, difficulty, created_at) VALUES
      (v_s3, 'ai',   'Расскажите, как организовали бы наблюдаемость для микросервиса на Go: метрики, логи, трейсы.', 'observability', 5, NOW() - interval '2 hours' + interval '1 min'),
      (v_s3, 'user', 'Метрики через Prometheus client, логи в JSON через zap, трейсы через OpenTelemetry с экспортом в Jaeger.', 'observability', 5, NOW() - interval '2 hours' + interval '5 min'),
      (v_s3, 'ai',   'А как сэмплируете трейсы при высокой нагрузке?', 'observability', 6, NOW() - interval '2 hours' + interval '6 min');
    RAISE NOTICE 'Created active session %', v_s3;
  END IF;

END $$;

-- ──────────────────────────────────────────────────────────────────────
-- 7) НА ВСЯКИЙ — кэш github для профиля
-- ──────────────────────────────────────────────────────────────────────
INSERT INTO github_profile_cache (user_id, payload, updated_at)
SELECT 'cf42ec6c-949f-46f0-bf62-1c73b000e083',
       jsonb_build_object(
         'username','BogdanSadovski',
         'profile_name','Bogdan Sadovski',
         'profile_url','https://github.com/BogdanSadovski',
         'public_repos', 23,
         'followers', 18,
         'stats', jsonb_build_object(
            'total_stars', 47,
            'total_forks', 12,
            'top_languages', jsonb_build_array('Go','Python','TypeScript','SQL')
         ),
         'top_repositories', jsonb_build_array(
            jsonb_build_object('name','BogdanSadovski.github.io','language','HTML','stars',1,'forks',0),
            jsonb_build_object('name','realsync-platform','language','Go','stars',24,'forks',6),
            jsonb_build_object('name','go-csv-streamer','language','Go','stars',12,'forks',3),
            jsonb_build_object('name','postgres-explain','language','TypeScript','stars',9,'forks',2)
         ),
         'charts', jsonb_build_object(
            'language_distribution', jsonb_build_array(
              jsonb_build_object('label','Go','value',42),
              jsonb_build_object('label','TypeScript','value',26),
              jsonb_build_object('label','Python','value',18),
              jsonb_build_object('label','SQL','value',14)
            )
         )
       ),
       NOW()
ON CONFLICT (user_id) DO UPDATE SET payload=EXCLUDED.payload, updated_at=NOW();

\echo ''
\echo '✓ Seed completed for demo_user_test@realsnc.com'
\echo '  • Профиль обновлён (имя, аватар, github, bio, телефон, локация)'
\echo '  • Подписка platinum активна (history + transaction)'
\echo '  • Резюме «Bogdan_Sadovski_Backend.pdf» добавлено'
\echo '  • 3 интервью-сессии (theory finished, practice finished, theory active)'
\echo '  • GitHub-кеш заполнен'

-- Per-session message threads + final reports for the demo sessions
-- seeded in 008. Each session gets a believable conversation:
--   AI question (verdict NULL) -> user answer (verdict set)
-- repeating 4-5 times. Verdicts feed a coherent rubric score that
-- matches what the report shows.

-- Session #1 — Backend Middle, theory, strong performer (score ~78)
INSERT INTO interview_messages (id, session_id, sender, content, topic, difficulty, created_at, verdict, verdict_reason) VALUES
  ('55550001-0000-0001-0001-000000000001','44444444-1111-0001-0000-000000000001','ai','Что такое eventual consistency и в каких сценариях она допустима?','consistency',6, NOW() - INTERVAL '80 days', NULL, NULL),
  ('55550001-0000-0001-0001-000000000002','44444444-1111-0001-0000-000000000001','user','Это когда узлы могут временно расходиться, но в итоге сходятся. Допустима для лайков/счётчиков, недопустима для денежных балансов где нужна strong.','consistency',6, NOW() - INTERVAL '80 days' + INTERVAL '2 minutes', 'correct','Чёткое определение + правильные сценарии'),
  ('55550001-0000-0001-0001-000000000003','44444444-1111-0001-0000-000000000001','ai','Как реализовать quorum-чтение в Cassandra?','consistency',7, NOW() - INTERVAL '80 days' + INTERVAL '4 minutes', NULL, NULL),
  ('55550001-0000-0001-0001-000000000004','44444444-1111-0001-0000-000000000001','user','QUORUM = (replication_factor / 2) + 1 узлов должны ответить. Например для RF=3 нужно 2.','consistency',7, NOW() - INTERVAL '80 days' + INTERVAL '6 minutes', 'correct','Правильная формула + пример'),
  ('55550001-0000-0001-0001-000000000005','44444444-1111-0001-0000-000000000001','ai','Какие trade-offs CAP-теоремы выбирает PostgreSQL?','consistency',7, NOW() - INTERVAL '80 days' + INTERVAL '10 minutes', NULL, NULL),
  ('55550001-0000-0001-0001-000000000006','44444444-1111-0001-0000-000000000001','user','По умолчанию CP — при потере соединения с master узел не отвечает. Можно сделать AP через async-репликацию и приёмом stale-чтений на репликах.','consistency',7, NOW() - INTERVAL '80 days' + INTERVAL '13 minutes', 'partial','Верно в целом, но не упомянут split-brain'),
  ('55550001-0000-0001-0001-000000000007','44444444-1111-0001-0000-000000000001','ai','Что такое read-your-writes consistency?','consistency',6, NOW() - INTERVAL '80 days' + INTERVAL '18 minutes', NULL, NULL),
  ('55550001-0000-0001-0001-000000000008','44444444-1111-0001-0000-000000000001','user','Гарантия что после моей записи я прочитаю её сам, даже если другие читатели ещё не видят.','consistency',6, NOW() - INTERVAL '80 days' + INTERVAL '21 minutes', 'correct','Краткое и точное определение')
ON CONFLICT (id) DO NOTHING;

INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score, strengths, weaknesses, recommendations, generated_at) VALUES
  ('44444444-1111-0001-0000-000000000001', 78.50, 82.00, 75.00, 90.00, 81.38,
    ARRAY['{"text":"Чёткое понимание eventual vs strong consistency"}'::jsonb, '{"text":"Знание quorum-формулы Cassandra"}'::jsonb],
    ARRAY['{"text":"Не раскрыт split-brain в контексте PostgreSQL"}'::jsonb],
    ARRAY['{"text":"Подтянуть тему consensus-алгоритмов (Raft, Paxos)"}'::jsonb, '{"text":"Прочитать про conflict-free replicated data types (CRDT)"}'::jsonb],
    NOW() - INTERVAL '80 days' + INTERVAL '35 minutes')
ON CONFLICT (session_id) DO NOTHING;

-- Session #2 — Backend practice, mid (score ~65)
INSERT INTO interview_messages (id, session_id, sender, content, topic, difficulty, created_at, verdict, verdict_reason) VALUES
  ('55550002-0000-0001-0001-000000000001','44444444-1111-0001-0000-000000000002','ai','Напишите на Go функцию TransferMoney(from, to *Account, amount int64) error с защитой от race condition.','transactions',6, NOW() - INTERVAL '70 days', NULL, NULL),
  ('55550002-0000-0001-0001-000000000002','44444444-1111-0001-0000-000000000002','user','```go
func TransferMoney(from, to *Account, amount int64) error {
  from.mu.Lock(); defer from.mu.Unlock()
  to.mu.Lock(); defer to.mu.Unlock()
  if from.Balance < amount { return errors.New("insufficient") }
  from.Balance -= amount; to.Balance += amount; return nil
}
```','transactions',6, NOW() - INTERVAL '70 days' + INTERVAL '5 minutes', 'partial','Решение работает, но есть классический deadlock: если параллельно вызовут TransferMoney(A,B) и TransferMoney(B,A), оба ждут друг друга.'),
  ('55550002-0000-0001-0001-000000000003','44444444-1111-0001-0000-000000000002','ai','Хорошо. Как исправить deadlock между двумя одновременными переводами A→B и B→A?','transactions',7, NOW() - INTERVAL '70 days' + INTERVAL '8 minutes', NULL, NULL),
  ('55550002-0000-0001-0001-000000000004','44444444-1111-0001-0000-000000000002','user','Сортировать lock по ID: всегда сначала lock того аккаунта чей ID меньше. Тогда оба перевода берут lock в одном порядке.','transactions',7, NOW() - INTERVAL '70 days' + INTERVAL '11 minutes', 'correct','Lock ordering — стандартное решение'),
  ('55550002-0000-0001-0001-000000000005','44444444-1111-0001-0000-000000000002','ai','Напишите SQL-запрос, который возвращает топ-3 пользователей по сумме переводов за последние 30 дней.','sql',6, NOW() - INTERVAL '70 days' + INTERVAL '14 minutes', NULL, NULL),
  ('55550002-0000-0001-0001-000000000006','44444444-1111-0001-0000-000000000002','user','```sql
SELECT from_id, SUM(amount) AS total
FROM transfers WHERE created_at > NOW() - INTERVAL ''30 days''
GROUP BY from_id ORDER BY total DESC LIMIT 3;
```','sql',6, NOW() - INTERVAL '70 days' + INTERVAL '16 minutes', 'correct','Правильно, можно добавить INDEX hint на created_at')
ON CONFLICT (id) DO NOTHING;

INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score, strengths, weaknesses, recommendations, generated_at) VALUES
  ('44444444-1111-0001-0000-000000000002', 68.00, 70.00, 65.00, 95.00, 74.50,
    ARRAY['{"text":"Знание lock ordering для предотвращения deadlock"}'::jsonb, '{"text":"Хорошие SQL-навыки"}'::jsonb],
    ARRAY['{"text":"Первое решение содержало deadlock-баг"}'::jsonb],
    ARRAY['{"text":"Изучить distributed locking patterns"}'::jsonb, '{"text":"Подтянуть тему индексов в больших таблицах"}'::jsonb],
    NOW() - INTERVAL '70 days' + INTERVAL '30 minutes')
ON CONFLICT (session_id) DO NOTHING;

-- Session #3 — Frontend Senior, theory, top performer (score ~90)
INSERT INTO interview_messages (id, session_id, sender, content, topic, difficulty, created_at, verdict, verdict_reason) VALUES
  ('55550003-0000-0001-0001-000000000001','44444444-1111-0001-0000-000000000003','ai','Объясните разницу между Core Web Vitals: LCP, FID, CLS.','performance',7, NOW() - INTERVAL '65 days', NULL, NULL),
  ('55550003-0000-0001-0001-000000000002','44444444-1111-0001-0000-000000000003','user','LCP — Largest Contentful Paint, время появления самого крупного элемента в viewport. Цель < 2.5s. FID — First Input Delay, задержка между первым взаимодействием и реакцией браузера. < 100ms. CLS — Cumulative Layout Shift, сумма visual instability scores. < 0.1.','performance',7, NOW() - INTERVAL '65 days' + INTERVAL '3 minutes', 'correct','Полное и точное объяснение всех 3 метрик'),
  ('55550003-0000-0001-0001-000000000003','44444444-1111-0001-0000-000000000003','ai','Как уменьшить LCP в React-приложении на 50%?','performance',8, NOW() - INTERVAL '65 days' + INTERVAL '6 minutes', NULL, NULL),
  ('55550003-0000-0001-0001-000000000004','44444444-1111-0001-0000-000000000003','user','1) Preload критичного hero-image через <link rel="preload">. 2) SSR/SSG для first paint, hydration после. 3) Code-splitting через React.lazy для не-критичных компонентов. 4) Оптимизировать самое тяжёлое — обычно картинки: WebP/AVIF + responsive srcset.','performance',8, NOW() - INTERVAL '65 days' + INTERVAL '10 minutes', 'correct','Системный ответ с приоритизацией'),
  ('55550003-0000-0001-0001-000000000005','44444444-1111-0001-0000-000000000003','ai','Что такое React Concurrent Mode и в чём его смысл?','react',8, NOW() - INTERVAL '65 days' + INTERVAL '14 minutes', NULL, NULL),
  ('55550003-0000-0001-0001-000000000006','44444444-1111-0001-0000-000000000003','user','Это режим где React может прерывать рендер если поступает высокоприоритетное обновление (например ввод текста). Реализовано через scheduler с приоритетами + Suspense для async-границ. Доступно через useTransition / useDeferredValue.','react',8, NOW() - INTERVAL '65 days' + INTERVAL '17 minutes', 'correct','Архитектурное понимание плюс конкретные API')
ON CONFLICT (id) DO NOTHING;

INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score, strengths, weaknesses, recommendations, generated_at) VALUES
  ('44444444-1111-0001-0000-000000000003', 92.00, 90.00, 88.00, 95.00, 91.25,
    ARRAY['{"text":"Глубокое знание Core Web Vitals"}'::jsonb, '{"text":"Системный подход к performance-оптимизации"}'::jsonb, '{"text":"Понимание архитектуры React Concurrent"}'::jsonb],
    ARRAY['{"text":"Можно глубже про accessibility и a11y-метрики"}'::jsonb],
    ARRAY['{"text":"Готов к Senior+ позициям"}'::jsonb, '{"text":"Можно идти на staff-уровень — нужно подтянуть архитектурный сторителлинг"}'::jsonb],
    NOW() - INTERVAL '65 days' + INTERVAL '42 minutes')
ON CONFLICT (session_id) DO NOTHING;

-- Session #9 — SoftSkills mode (score ~72)
INSERT INTO interview_messages (id, session_id, sender, content, topic, difficulty, created_at, verdict, verdict_reason) VALUES
  ('55550009-0000-0001-0001-000000000001','44444444-1111-0001-0000-000000000009','ai','Расскажите о ситуации, когда вам пришлось разрешить конфликт в команде.','soft_skills',5, NOW() - INTERVAL '30 days', NULL, NULL),
  ('55550009-0000-0001-0001-000000000002','44444444-1111-0001-0000-000000000009','user','На прошлом проекте фронт и бэк спорили про формат API. Я организовал встречу, собрал требования с обеих сторон, нарисовал OpenAPI-спеку и зафиксировал ADR. Конфликт разрешился за 2 часа, потом договорились всегда фиксировать решения в ADR.','soft_skills',5, NOW() - INTERVAL '30 days' + INTERVAL '3 minutes', 'correct','Структурированный STAR-ответ с конкретным результатом'),
  ('55550009-0000-0001-0001-000000000003','44444444-1111-0001-0000-000000000009','ai','Как вы реагируете на критику вашего кода?','soft_skills',4, NOW() - INTERVAL '30 days' + INTERVAL '6 minutes', NULL, NULL),
  ('55550009-0000-0001-0001-000000000004','44444444-1111-0001-0000-000000000009','user','Стараюсь не воспринимать лично. Если критика по делу — благодарю и переписываю. Если не согласен — прошу конкретный пример или ссылку на гайд. Регулярно делаю PR-review-сессии чтобы калибровать стандарты команды.','soft_skills',4, NOW() - INTERVAL '30 days' + INTERVAL '9 minutes', 'partial','Хороший подход, но не хватает конкретного примера из опыта'),
  ('55550009-0000-0001-0001-000000000005','44444444-1111-0001-0000-000000000009','ai','Как вы мотивируете младшего коллегу?','soft_skills',5, NOW() - INTERVAL '30 days' + INTERVAL '12 minutes', NULL, NULL),
  ('55550009-0000-0001-0001-000000000006','44444444-1111-0001-0000-000000000009','user','Стараюсь дать ему задачу немного выше его текущего уровня, провожу 1:1 раз в две недели где обсуждаем что получилось/что блокирует. Главное — давать пространство для ошибок но регулярный feedback.','soft_skills',5, NOW() - INTERVAL '30 days' + INTERVAL '16 minutes', 'correct','Конкретный подход с регулярностью и измеримостью')
ON CONFLICT (id) DO NOTHING;

INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score, strengths, weaknesses, recommendations, generated_at) VALUES
  ('44444444-1111-0001-0000-000000000009', 75.00, 80.00, 70.00, 95.00, 80.00,
    ARRAY['{"text":"Хорошее владение STAR-форматом"}'::jsonb, '{"text":"Конкретные методы менторства"}'::jsonb],
    ARRAY['{"text":"Не всегда приводит примеры из реального опыта"}'::jsonb],
    ARRAY['{"text":"Подготовить 3-5 готовых STAR-историй на типовые soft-skills вопросы"}'::jsonb],
    NOW() - INTERVAL '30 days' + INTERVAL '20 minutes')
ON CONFLICT (session_id) DO NOTHING;

-- Sessions #4-8, #10-19 — bulk insert with simpler 4-message pattern
-- and varied scores so dashboard charts look natural.
INSERT INTO interview_reports (session_id, correctness, clarity, completeness, relevance, overall_score, strengths, weaknesses, recommendations, generated_at) VALUES
  ('44444444-1111-0001-0000-000000000004', 85.00, 88.00, 80.00, 90.00, 85.75,
    ARRAY['{"text":"Хорошее владение React hooks"}'::jsonb, '{"text":"Внимание к accessibility"}'::jsonb],
    ARRAY['{"text":"Не хватило времени на edge cases"}'::jsonb],
    ARRAY['{"text":"Подготовить решение custom-hooks из готовых паттернов"}'::jsonb],
    NOW() - INTERVAL '50 days' + INTERVAL '38 minutes'),
  ('44444444-1111-0001-0000-000000000005', 70.00, 72.00, 65.00, 90.00, 74.25,
    ARRAY['{"text":"Знание базовых k8s-объектов"}'::jsonb],
    ARRAY['{"text":"Слабо в Helm и templating"}'::jsonb],
    ARRAY['{"text":"Пройти курс по Helm + ArgoCD"}'::jsonb],
    NOW() - INTERVAL '55 days' + INTERVAL '30 minutes'),
  ('44444444-1111-0001-0000-000000000006', 58.00, 60.00, 55.00, 85.00, 64.50,
    ARRAY['{"text":"Базовое понимание PyTorch"}'::jsonb],
    ARRAY['{"text":"Слабо в data engineering часть pipeline"}'::jsonb, '{"text":"Не упомянуты metrics для классификации"}'::jsonb],
    ARRAY['{"text":"Junior — ещё рано на ML-роль. Подтянуть data prep и evaluation"}'::jsonb],
    NOW() - INTERVAL '45 days' + INTERVAL '24 minutes'),
  ('44444444-1111-0001-0000-000000000007', 80.00, 78.00, 82.00, 95.00, 83.75,
    ARRAY['{"text":"Отличные SQL-навыки"}'::jsonb, '{"text":"Понимание window functions"}'::jsonb],
    ARRAY['{"text":"Спутал DELETE и TRUNCATE по поведению индексов"}'::jsonb],
    ARRAY['{"text":"Подтянуть тему partitioning + sharding"}'::jsonb],
    NOW() - INTERVAL '40 days' + INTERVAL '32 minutes'),
  ('44444444-1111-0001-0000-000000000008', 88.00, 85.00, 87.00, 95.00, 88.75,
    ARRAY['{"text":"Глубокое знание MVVM и Composable архитектур"}'::jsonb, '{"text":"Опыт с offline-first паттернами"}'::jsonb],
    ARRAY['{"text":"Можно глубже про SwiftUI vs UIKit trade-offs"}'::jsonb],
    ARRAY['{"text":"Готов к Senior Mobile позициям"}'::jsonb],
    NOW() - INTERVAL '35 days' + INTERVAL '40 minutes'),
  ('44444444-1111-0001-0000-000000000010', 95.00, 92.00, 90.00, 100.00, 94.25,
    ARRAY['{"text":"Глубокое понимание Raft consensus"}'::jsonb, '{"text":"Опыт с реальным sharding (Vitess, Cassandra)"}'::jsonb],
    ARRAY[]::jsonb[],
    ARRAY['{"text":"Готов к Staff/Principal позициям"}'::jsonb],
    NOW() - INTERVAL '28 days' + INTERVAL '50 minutes'),
  ('44444444-1111-0001-0000-000000000011', 45.00, 50.00, 40.00, 90.00, 56.25,
    ARRAY['{"text":"Знание базового синтаксиса Go"}'::jsonb],
    ARRAY['{"text":"Не понимает разницу между slice и array"}'::jsonb, '{"text":"Не знает что такое interface"}'::jsonb],
    ARRAY['{"text":"Пройти Go Tour полностью"}'::jsonb, '{"text":"Прочитать Effective Go"}'::jsonb],
    NOW() - INTERVAL '25 days' + INTERVAL '15 minutes'),
  ('44444444-1111-0001-0000-000000000012', 78.00, 80.00, 75.00, 95.00, 82.00,
    ARRAY['{"text":"Реализовал debounce за 2 минуты"}'::jsonb, '{"text":"Знание useEffect-deps"}'::jsonb],
    ARRAY['{"text":"Не учёл cancellation при umount"}'::jsonb],
    ARRAY['{"text":"Подтянуть тему AbortController"}'::jsonb],
    NOW() - INTERVAL '22 days' + INTERVAL '35 minutes'),
  ('44444444-1111-0001-0000-000000000013', 68.00, 75.00, 65.00, 95.00, 75.75,
    ARRAY['{"text":"Открытый стиль общения"}'::jsonb],
    ARRAY['{"text":"Мало конкретных примеров"}'::jsonb, '{"text":"Не использует STAR"}'::jsonb],
    ARRAY['{"text":"Заучить 5 STAR-историй на типовые ситуации"}'::jsonb],
    NOW() - INTERVAL '20 days' + INTERVAL '22 minutes'),
  ('44444444-1111-0001-0000-000000000014', 82.00, 85.00, 78.00, 90.00, 83.75,
    ARRAY['{"text":"Знание Redis для caching"}'::jsonb, '{"text":"Понимание cache invalidation patterns"}'::jsonb],
    ARRAY['{"text":"Не упомянул cache stampede"}'::jsonb],
    ARRAY['{"text":"Изучить write-through vs write-back trade-offs"}'::jsonb],
    NOW() - INTERVAL '18 days' + INTERVAL '30 minutes'),
  ('44444444-1111-0001-0000-000000000015', 86.00, 88.00, 82.00, 95.00, 87.75,
    ARRAY['{"text":"Production-опыт с GitOps и Helm"}'::jsonb, '{"text":"Понимание K8s networking"}'::jsonb],
    ARRAY['{"text":"Не упомянуты service mesh (Istio, Linkerd)"}'::jsonb],
    ARRAY['{"text":"Готов к Senior DevOps, лидерским ролям"}'::jsonb],
    NOW() - INTERVAL '15 days' + INTERVAL '42 minutes'),
  ('44444444-1111-0001-0000-000000000016', 91.00, 90.00, 88.00, 100.00, 92.25,
    ARRAY['{"text":"Глубокое понимание low-latency"}'::jsonb, '{"text":"Опыт с FPGA, kernel bypass"}'::jsonb],
    ARRAY[]::jsonb[],
    ARRAY['{"text":"Готов к старшим позициям в HFT"}'::jsonb],
    NOW() - INTERVAL '12 days' + INTERVAL '48 minutes'),
  ('44444444-1111-0001-0000-000000000017', 40.00, 45.00, 35.00, 85.00, 51.25,
    ARRAY['{"text":"Базовое знание HTML/CSS"}'::jsonb],
    ARRAY['{"text":"Слабо в JS-основах (closures, this)"}'::jsonb, '{"text":"Не знает разницы var/let/const"}'::jsonb],
    ARRAY['{"text":"Пройти JS-курс на freeCodeCamp"}'::jsonb, '{"text":"Прочитать You Dont Know JS"}'::jsonb],
    NOW() - INTERVAL '10 days' + INTERVAL '18 minutes'),
  ('44444444-1111-0001-0000-000000000018', 72.00, 78.00, 70.00, 95.00, 78.75,
    ARRAY['{"text":"Хорошие коммуникативные навыки"}'::jsonb, '{"text":"Умеет слушать"}'::jsonb],
    ARRAY['{"text":"Иногда отвечает слишком обобщённо"}'::jsonb],
    ARRAY['{"text":"Готовить конкретные числа в ответах (сроки, размер команды, метрики)"}'::jsonb],
    NOW() - INTERVAL '8 days' + INTERVAL '25 minutes'),
  ('44444444-1111-0001-0000-000000000019', 87.00, 85.00, 88.00, 95.00, 88.75,
    ARRAY['{"text":"Production-опыт с PySpark"}'::jsonb, '{"text":"Понимание Airflow DAG patterns"}'::jsonb],
    ARRAY['{"text":"Не упомянуты Delta Lake / Iceberg"}'::jsonb],
    ARRAY['{"text":"Подтянуть data lakehouse patterns"}'::jsonb],
    NOW() - INTERVAL '6 days' + INTERVAL '40 minutes')
ON CONFLICT (session_id) DO NOTHING;

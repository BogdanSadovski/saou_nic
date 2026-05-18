import { useNavigate } from "react-router-dom";

import { Counter, RsIcon as Icon, Track } from "@/shared/ui/realsync";

export default function CareerCenterPage() {
  const navigate = useNavigate();

  const radar = [
    { role: "Backend", val: 92, hint: "Приоритетный сценарий для следующего интервью" },
    { role: "Frontend", val: 64, hint: "Хорошая зона для развития" },
    { role: "DevOps", val: 41, hint: "Глубоко добрать знания и практику" },
    { role: "ML", val: 32, hint: "Глубоко добрать знания и практику" },
    { role: "Mobile", val: 18, hint: "Базовое знакомство" },
  ];

  const sims = [
    { name: "Backend", fit: 92 },
    { name: "Frontend", fit: 64 },
    { name: "DevOps", fit: 41 },
  ];

  const plan = [
    "Разобрать слабую зону: distributed consensus (Raft, Paxos)",
    "Добавить практику по теме: проектирование очередей и backpressure",
    "Закрепить сильную сторону: trade-off диалоги по выбору БД",
    "Использовать рекомендацию: добавить метрики в каждое решение",
  ];

  return (
    <>
      <section className="career-hero">
        <div>
          <span className="eyebrow">Карьерный центр</span>
          <h1 className="expr-headline" style={{ fontSize: "clamp(44px, 5.6vw, 80px)", margin: "20px 0 24px" }}>
            <span className="bold">Единое</span> <span className="ital">рабочее</span><br />
            <span className="light">пространство</span> <span className="ital underline">роста</span>.
          </h1>
          <p className="lede">
            Симулятор, трек развития, анализ пробелов, экспорт публичного профиля и быстрые переходы в существующие сценарии — собраны на одной странице без визуального шума.
          </p>
          <div className="career-actions">
            <button className="btn btn--primary" onClick={() => navigate("/resume")} type="button">Открыть резюме <Icon name="arrow" /></button>
            <button className="btn btn--ghost" onClick={() => navigate("/reports")} type="button">Открыть отчёты</button>
            <button className="btn btn--ghost" onClick={() => navigate("/profile")} type="button">Открыть профиль</button>
          </div>
        </div>

        <aside className="career-pulse scanline">
          <span className="eyebrow" style={{ color: "oklch(0.84 0.18 130)" }}>Career Pulse · live</span>
          <h3>Серия 6 дней,<br />momentum растёт.</h3>
          <div className="pulse-rows">
            <div className="pulse-row">
              <span className="pulse-label">Серия</span>
              <span className="pulse-value mono">6d</span>
            </div>
            <div className="pulse-row">
              <span className="pulse-label">Динамика</span>
              <span className="pulse-value mono">+12</span>
            </div>
            <div className="pulse-row">
              <span className="pulse-label">Стабильность</span>
              <span className="pulse-value mono">87%</span>
            </div>
          </div>
        </aside>
      </section>

      <section className="metric-row" style={{ marginTop: 32 }}>
        <div className="metric reveal reveal-1">
          <div className="metric-label">Средняя оценка</div>
          <div className="metric-value mono"><Counter target={82} suffix="%" /></div>
        </div>
        <div className="metric reveal reveal-2">
          <div className="metric-label">Завершение</div>
          <div className="metric-value mono"><Counter target={94} suffix="%" /></div>
        </div>
        <div className="metric reveal reveal-3">
          <div className="metric-label">Пройдено</div>
          <div className="metric-value mono"><Counter target={28} /></div>
        </div>
      </section>

      <section className="career-modules">
        <div className="card card--hover wide reveal reveal-1">
          <span className="eyebrow">AI Career Copilot</span>
          <h3>Что делать дальше</h3>
          <p className="body" style={{ marginTop: 12 }}>
            Добавьте измеримые критерии в ответы о системном дизайне — задавайте границы, числа, ограничения. Это закроет основной паттерн потери баллов в последних 4 сессиях.
          </p>
          <div className="recs-list" style={{ marginTop: 20 }}>
            {["Добавлять измеримые критерии", "Явно проговаривать trade-offs", "Закрепить структуру STAR в ответах", "Сократить latency ответа до 75 сек"].map((r, i) => (
              <div className="rec-item" key={i}>
                <div className="rec-bullet">{String(i + 1).padStart(2, "0")}</div>
                <div className="rec-text">{r}</div>
              </div>
            ))}
          </div>
          <div className="row" style={{ marginTop: 16 }}>
            <button className="btn btn--accent" onClick={() => navigate("/interview")} type="button">Запустить лучший симулятор <Icon name="arrow" /></button>
            <button className="btn btn--ghost" type="button">Сохранить публичный профиль</button>
          </div>
        </div>

        <div className="card card--hover reveal reveal-2">
          <span className="eyebrow">Карьерный радар</span>
          <h3>Сильные направления</h3>
          <div className="radar-list">
            {radar.map((r, i) => (
              <div className="radar-item reveal" style={{ animationDelay: `${i * 80}ms` }} key={r.role}>
                <div className="radar-head">
                  <strong>{r.role}</strong>
                  <em className="mono">{r.val}%</em>
                </div>
                <Track value={r.val} />
                <p className="muted" style={{ fontSize: 12 }}>{r.hint}</p>
              </div>
            ))}
          </div>
        </div>

        <div className="card card--hover reveal reveal-3">
          <span className="eyebrow">Симулятор интервью</span>
          <h3>Быстрый старт</h3>
          <p className="body" style={{ marginTop: 8 }}>Лучшие роли и темы из вашей аналитики.</p>
          <div className="sim-list">
            {sims.map((s) => (
              <button key={s.name} className="sim-item" onClick={() => navigate("/interview")} type="button">
                <strong>{s.name}</strong>
                <span className="fit mono">{s.fit}%</span>
                <Icon name="arrow" size={14} />
              </button>
            ))}
          </div>
        </div>

        <div className="card card--hover reveal reveal-4">
          <span className="eyebrow">Лаборатория резюме</span>
          <h3>Что улучшить в резюме</h3>
          <div className="recs-list" style={{ marginTop: 16 }}>
            {["Конкретизировать impact-метрики", "Добавить пример системного дизайна", "Сократить experience section"].map((r, i) => (
              <div className="rec-item" key={i}>
                <div className="rec-bullet">·</div>
                <div className="rec-text">{r}</div>
              </div>
            ))}
          </div>
          <button className="btn btn--ghost btn--sm" style={{ marginTop: 16 }} onClick={() => navigate("/resume")} type="button">Перейти в лабораторию</button>
        </div>

        <div className="card card--hover reveal reveal-5">
          <span className="eyebrow">План обучения</span>
          <h3>План на 7 дней</h3>
          <ol style={{ marginTop: 16, display: "grid", gap: 12 }}>
            {plan.map((p, i) => (
              <li key={i} className="row" style={{ alignItems: "baseline", gap: 14 }}>
                <span className="mono" style={{ color: "var(--muted)", fontSize: 11 }}>{String(i + 1).padStart(2, "0")}</span>
                <span style={{ fontSize: 14, color: "var(--ink-2)", lineHeight: 1.55 }}>{p}</span>
              </li>
            ))}
          </ol>
        </div>

        <div className="card card--hover reveal reveal-6">
          <span className="eyebrow">Публичный профиль</span>
          <h3>Публикация результата</h3>
          <p className="body" style={{ marginTop: 8 }}>Сохраните карточку кандидата и поделитесь ссылкой без ручной вёрстки.</p>
          <div style={{ marginTop: 16, padding: 16, border: "1px solid var(--line)", borderRadius: "var(--r-1)", display: "grid", gap: 4 }}>
            <strong>Садовский Богдан Дм.</strong>
            <span className="muted" style={{ fontSize: 13 }}>Backend · 82% readiness</span>
            <span className="mono" style={{ fontSize: 11, color: "var(--muted)" }}>realsync.io/p/bsadovski</span>
          </div>
          <div className="row" style={{ marginTop: 16 }}>
            <button className="btn btn--primary btn--sm" type="button">Сохранить снапшот</button>
            <button className="btn btn--ghost btn--sm" type="button">Скопировать ссылку</button>
          </div>
        </div>
      </section>
    </>
  );
}

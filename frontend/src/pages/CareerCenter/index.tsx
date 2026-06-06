import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { Counter, RsIcon as Icon, Track } from "@/shared/ui/realsync";

const radarHint = (val: number): string => {
  if (val >= 80) return "Приоритетный сценарий для следующего интервью";
  if (val >= 50) return "Хорошая зона для развития";
  if (val >= 30) return "Глубоко добрать знания и практику";
  return "Базовое знакомство";
};

export default function CareerCenterPage() {
  const navigate = useNavigate();
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await reportsApi.getMyInterviewReport();
        if (!cancelled) setReport(data);
      } catch (e) {
        const status = (e as { response?: { status?: number } })?.response?.status;
        if (!cancelled && status === 404) setReport(reportsApi.emptyReport());
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const totalInterviews = report?.totals.total_interviews ?? 0;
  const avgScore = Math.round(report?.performance.average_score ?? 0);
  const completionRate = Math.round(report?.totals.completion_rate ?? 0);
  const hasData = totalInterviews > 0;

  const radar = useMemo(() => {
    const items = (report?.role_distribution || []).slice(0, 5);
    return items.map((r) => ({ role: r.label, val: Math.round(r.value), hint: radarHint(Math.round(r.value)) }));
  }, [report]);

  const sims = useMemo(() => radar.slice(0, 3).map((r) => ({ name: r.role, fit: r.val })), [radar]);
  const recs = useMemo(() => (report?.top_recommendations || []).slice(0, 4), [report]);

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
          {hasData ? (
            <>
              <h3>Аналитика по<br />{totalInterviews} интервью.</h3>
              <div className="pulse-rows">
                <div className="pulse-row">
                  <span className="pulse-label">Средняя оценка</span>
                  <span className="pulse-value mono">{avgScore}</span>
                </div>
                <div className="pulse-row">
                  <span className="pulse-label">Завершение</span>
                  <span className="pulse-value mono">{completionRate}%</span>
                </div>
                <div className="pulse-row">
                  <span className="pulse-label">Пройдено</span>
                  <span className="pulse-value mono">{totalInterviews}</span>
                </div>
              </div>
            </>
          ) : (
            <>
              <h3>Здесь появится<br />ваша аналитика.</h3>
              <p className="muted" style={{ fontSize: 13, marginTop: 12 }}>
                Пройдите первое интервью — и панель заполнится реальными показателями.
              </p>
            </>
          )}
        </aside>
      </section>

      <section className="metric-row" style={{ marginTop: 32 }}>
        <div className="metric reveal reveal-1">
          <div className="metric-label">Средняя оценка</div>
          <div className="metric-value mono"><Counter target={avgScore} suffix="%" /></div>
        </div>
        <div className="metric reveal reveal-2">
          <div className="metric-label">Завершение</div>
          <div className="metric-value mono"><Counter target={completionRate} suffix="%" /></div>
        </div>
        <div className="metric reveal reveal-3">
          <div className="metric-label">Пройдено</div>
          <div className="metric-value mono"><Counter target={totalInterviews} /></div>
        </div>
      </section>

      <section className="career-modules">
        <div className="card card--hover wide reveal reveal-1">
          <span className="eyebrow">AI Career Copilot</span>
          <h3>Что делать дальше</h3>
          {recs.length > 0 ? (
            <>
              <p className="body" style={{ marginTop: 12 }}>
                Персональные рекомендации на основе ваших последних интервью.
              </p>
              <div className="recs-list" style={{ marginTop: 20 }}>
                {recs.map((r, i) => (
                  <div className="rec-item" key={i}>
                    <div className="rec-bullet">{String(i + 1).padStart(2, "0")}</div>
                    <div className="rec-text">{r}</div>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <p className="body" style={{ marginTop: 12 }}>
              Рекомендации появятся после первого пройденного интервью: система определит частые
              ошибки и подскажет, что улучшить.
            </p>
          )}
          <div className="row" style={{ marginTop: 16 }}>
            <button className="btn btn--accent" onClick={() => navigate("/interview")} type="button">Запустить симулятор <Icon name="arrow" /></button>
          </div>
        </div>

        <div className="card card--hover reveal reveal-2">
          <span className="eyebrow">Карьерный радар</span>
          <h3>Сильные направления</h3>
          {radar.length > 0 ? (
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
          ) : (
            <p className="muted" style={{ fontSize: 14, marginTop: 12 }}>
              Радар направлений построится по мере прохождения интервью в разных областях.
            </p>
          )}
        </div>

        <div className="card card--hover reveal reveal-3">
          <span className="eyebrow">Симулятор интервью</span>
          <h3>Быстрый старт</h3>
          {sims.length > 0 ? (
            <>
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
            </>
          ) : (
            <>
              <p className="body" style={{ marginTop: 8 }}>Выберите направление и начните первое интервью.</p>
              <button className="btn btn--ghost btn--sm" style={{ marginTop: 16 }} onClick={() => navigate("/interview")} type="button">Выбрать направление</button>
            </>
          )}
        </div>

        <div className="card card--hover reveal reveal-4">
          <span className="eyebrow">Лаборатория резюме</span>
          <h3>Анализ резюме</h3>
          <p className="body" style={{ marginTop: 12 }}>
            Загрузите резюме — система выделит навыки, языки и подскажет, что улучшить для целевой роли.
          </p>
          <button className="btn btn--ghost btn--sm" style={{ marginTop: 16 }} onClick={() => navigate("/resume")} type="button">Перейти в лабораторию</button>
        </div>
      </section>
    </>
  );
}

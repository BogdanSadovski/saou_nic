import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { Counter, RsIcon as Icon, Sparkline } from "@/shared/ui/realsync";

export default function WorkspaceOverview() {
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

  const interviews = useMemo(() => {
    const items = report?.recent_interviews?.slice(0, 5) || [];
    if (items.length === 0) {
      return [
        { role: "Backend", vac: "Backend-инженер (Go, платформа)", mode: "theory", score: 91, date: "12 мая" },
        { role: "Backend", vac: "Backend-инженер (Go, платформа)", mode: "theory", score: 83, date: "10 мая" },
        { role: "Backend", vac: "Backend-инженер (Go, платформа)", mode: "practice", score: 72, date: "08 мая" },
        { role: "Backend", vac: "Backend-инженер (Go, платформа)", mode: "theory", score: 88, date: "05 мая" },
        { role: "Backend", vac: "Backend-инженер (Go, платформа)", mode: "theory", score: 79, date: "02 мая" },
      ];
    }
    return items.map((it) => ({
      role: it.role,
      vac: it.vacancy_title || it.role,
      mode: it.interview_mode || "theory",
      score: it.overall_score ?? 0,
      date: it.started_at ? new Date(it.started_at).toLocaleDateString("ru-RU", { day: "2-digit", month: "short" }) : "",
    }));
  }, [report]);

  const recs = useMemo(() => {
    const items = report?.top_recommendations || [];
    if (items.length > 0) return items.slice(0, 3);
    return [
      "Добавлять измеримые критерии — числа, диапазоны, ограничения",
      "Явно проговаривать trade-offs до выбора решения",
      "Попробуйте пройти интервью заново и отвечайте на каждый вопрос хотя бы 1–2 предложениями",
    ];
  }, [report]);

  const totalInterviews = report?.totals.total_interviews ?? 28;
  const avgScore = Math.round(report?.performance.average_score ?? 82);
  const completionRate = Math.round(report?.totals.completion_rate ?? 89);

  return (
    <>
      <header className="dash-head">
        <div>
          <span className="eyebrow">Рабочее пространство</span>
          <h1 className="expr-headline">
            <span className="ital">Панель</span> <span className="bold">управления</span>
          </h1>
        </div>
        <button className="btn btn--ghost" onClick={() => navigate("/reports")} type="button">
          <Icon name="chart" size={14} /> Открыть отчёты
        </button>
      </header>

      <div className="metric-row">
        <div className="metric reveal reveal-1">
          <div className="metric-label">Интервью</div>
          <div className="metric-value mono"><Counter target={totalInterviews} /></div>
          <div className="metric-delta">↑ 4 за неделю</div>
          <div className="metric-spark">
            <Sparkline data={[2, 3, 1, 4, 2, 5, 3, 6, 4, 5]} width={220} height={32} />
          </div>
        </div>
        <div className="metric reveal reveal-2">
          <div className="metric-label">Средняя оценка</div>
          <div className="metric-value mono"><Counter target={avgScore} /><em>/100</em></div>
          <div className="metric-delta">↑ +3.2 за месяц</div>
          <div className="metric-spark">
            <Sparkline data={[68, 70, 72, 74, 71, 76, 78, 80, 81, 82]} width={220} height={32} />
          </div>
        </div>
        <div className="metric reveal reveal-3">
          <div className="metric-label">Совпадение резюме</div>
          <div className="metric-value mono"><Counter target={completionRate} suffix="%" /></div>
          <div className="metric-delta">стабильно</div>
          <div className="metric-spark">
            <Sparkline data={[85, 86, 84, 87, 89, 88, 89, 90, 89, 89]} width={220} height={32} />
          </div>
        </div>
      </div>

      <section className="dash-section">
        <header className="dash-section-head">
          <h2>Недавние интервью</h2>
          <span className="eyebrow">за 30 дней</span>
        </header>
        <div className="interview-list">
          {interviews.map((it, i) => (
            <div className="interview-item" key={i}>
              <div className="interview-item-num mono">{String(i + 1).padStart(2, "0")}</div>
              <div>
                <div className="interview-item-title">{it.role} · {it.vac}</div>
                <div className="interview-item-sub mono">{it.mode.toUpperCase()}</div>
              </div>
              <div className="tag">{it.mode}</div>
              <div className="interview-item-score">{it.score}</div>
              <div className="interview-item-date mono">{it.date}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="dash-section">
        <div className="grid-2">
          <div>
            <header className="dash-section-head">
              <h2>Рекомендации</h2>
            </header>
            <div className="recs-list">
              {recs.map((r, i) => (
                <div className="rec-item" key={i}>
                  <div className="rec-bullet">{String(i + 1).padStart(2, "0")}</div>
                  <div className="rec-text">{r}</div>
                </div>
              ))}
            </div>
          </div>

          <div>
            <header className="dash-section-head">
              <h2>Активность</h2>
            </header>
            <div className="gh-card" style={{ padding: 0, border: "none", background: "transparent" }}>
              <div className="gh-grid">
                {Array.from({ length: 49 }).map((_, i) => {
                  const r = ((i * 31 + 13) % 7) / 7 + ((i * 17) % 5) / 10;
                  const lv = r > 1.1 ? "l4" : r > 0.85 ? "l3" : r > 0.55 ? "l2" : r > 0.3 ? "l1" : "";
                  return <div key={i} className={`gh-cell ${lv}`} style={{ animationDelay: `${i * 8}ms` }} />;
                })}
              </div>
              <div className="row-between" style={{ marginTop: 8, fontFamily: "var(--f-mono)", fontSize: 11, color: "var(--muted)" }}>
                <span>7 недель</span>
                <span>сегодня</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}

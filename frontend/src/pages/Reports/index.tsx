import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { Counter, RsIcon as Icon, Sparkline } from "@/shared/ui/realsync";
import { renderAndPrintReport } from "./pdfExport";

export default function ReportsPage() {
  const navigate = useNavigate();
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<"all" | "finished" | "active" | "expired">("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadReport = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const userReport = await reportsApi.getMyInterviewReport();
      setReport(userReport);
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      const message = e instanceof Error ? e.message : "Не удалось загрузить отчет";
      if (status === 404) {
        setReport(reportsApi.emptyReport());
        setError(null);
      } else {
        setError(message);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadReport();
  }, [loadReport]);

  const filtered = useMemo(() => {
    if (!report) return [];
    const q = search.trim().toLowerCase();
    return report.recent_interviews.filter((item) => {
      const okStatus = filter === "all" || item.status === filter;
      const okSearch =
        !q ||
        item.role.toLowerCase().includes(q) ||
        (item.vacancy_title || "").toLowerCase().includes(q) ||
        item.session_id.toLowerCase().includes(q);
      return okStatus && okSearch;
    });
  }, [report, search, filter]);

  const exportJson = () => {
    if (!report) return;
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `user-interview-report-${report.user_id}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const exportPdf = () => {
    if (!report) return;
    void renderAndPrintReport(report);
  };

  if (loading) {
    return (
      <main className="page">
        <h1 className="expr-headline"><span className="ital">Загрузка</span> отчёта</h1>
      </main>
    );
  }

  if (error) {
    return (
      <main className="page">
        <h1 className="expr-headline"><span className="ital">Не удалось</span> загрузить отчёт</h1>
        <p className="muted">{error}</p>
        <button className="btn btn--primary" onClick={() => void loadReport()} type="button">Повторить</button>
      </main>
    );
  }

  if (!report) return null;

  const totals = report.totals;
  const perf = report.performance;
  const completionRate = Math.round(totals.completion_rate);
  const avg = Math.round(perf.average_score);
  const best = Math.round(perf.best_score);
  const incomplete = totals.in_progress_interviews + totals.expired_interviews;

  const stats = [
    { l: "Всего", v: totals.total_interviews },
    { l: "Завершено", v: totals.completed_interviews },
    { l: "Не завершено", v: incomplete },
    { l: "Завершение", v: completionRate, s: "%" as const },
    { l: "Среднее", v: avg },
    { l: "Лучший", v: best },
  ];

  const timelineCompleted = [...report.timeline].sort((a, b) => a.date.localeCompare(b.date)).map((p) => p.completed);
  const sparkData = timelineCompleted.length ? timelineCompleted.slice(-10) : [2, 3, 2, 4, 3, 5, 4, 6, 5, 7];

  return (
    <main className="page" data-screen-label="06 Reports">
      <div className="sysbar reveal" style={{ marginBottom: 24 }}>
        <span><span className="dot"></span><span className="k">report</span><span className="v">v3.2</span></span>
        <span><span className="k">user_id</span><span className="v">{report.user_id.slice(0, 8)}</span></span>
        <span><span className="k">generated</span><span className="v">{new Date(report.generated_at).toLocaleString("ru-RU")}</span></span>
        <span><span className="k">data points</span><span className="v">{totals.total_interviews}</span></span>
        <span><span className="k">format</span><span className="v">analytics.v2</span></span>
      </div>

      <header className="reports-head">
        <div>
          <span className="eyebrow">Отчёт пользователя</span>
          <h1 className="expr-headline" style={{ fontSize: 72 }}>
            <span className="bold">Отчёт</span><br />
            <span className="ital">по пользователю</span>.
          </h1>
          <div className="id mono">user_id: {report.user_id.slice(0, 12)} · report v3.2</div>
        </div>
        <div className="row">
          <button className="btn btn--ghost" onClick={exportJson} type="button"><Icon name="download" size={14} /> Экспорт JSON</button>
          <button className="btn btn--primary" onClick={exportPdf} type="button"><Icon name="download" size={14} /> Экспорт PDF</button>
        </div>
      </header>

      <section className="report-stats">
        {stats.map((s, i) => (
          <div className="report-stat reveal" style={{ animationDelay: `${i * 60}ms` }} key={s.l}>
            <div className="report-stat-label">{s.l}</div>
            <div className="report-stat-value mono">
              <Counter target={s.v} />{s.s ? <span className="report-stat-suffix">{s.s}</span> : null}
            </div>
          </div>
        ))}
      </section>

      <section className="metric-row" style={{ marginTop: 0, borderTop: "none" }}>
        <div className="metric">
          <div className="metric-label">Карьерный пульс</div>
          <div className="metric-value mono">+{Math.max(0, Math.round((perf.latest_score || 0) - perf.average_score))}</div>
          <div className="metric-delta">Динамика завершения за последние сессии</div>
          <div className="metric-spark"><Sparkline data={sparkData} width={220} height={32} /></div>
        </div>
        <div className="metric">
          <div className="metric-label">Серия дней</div>
          <div className="metric-value mono"><Counter target={Math.min(7, timelineCompleted.filter((c) => c > 0).length)} /></div>
          <div className="metric-delta">Дней подряд с завершёнными интервью</div>
        </div>
        <div className="metric">
          <div className="metric-label">Индекс надёжности</div>
          <div className="metric-value mono"><Counter target={completionRate} suffix="%" /></div>
          <div className="metric-delta">Consistency · {completionRate}%</div>
        </div>
      </section>

      <section className="lists-grid">
        <div className="list-col">
          <h3>Сильные стороны</h3>
          <ul>
            {report.top_strengths.length === 0 ? <li>Недостаточно данных</li> : report.top_strengths.map((s) => <li key={s}>{s}</li>)}
          </ul>
        </div>
        <div className="list-col">
          <h3>Слабые стороны</h3>
          <ul>
            {report.top_weaknesses.length === 0 ? <li>Недостаточно данных</li> : report.top_weaknesses.map((s) => <li key={s}>{s}</li>)}
          </ul>
        </div>
      </section>

      <section style={{ marginTop: 40 }}>
        <header className="dash-section-head">
          <h2>Спринт развития</h2>
          <span className="eyebrow">adaptive · v2</span>
        </header>
        <div className="card scanline" style={{ background: "var(--ink)", color: "var(--bg)", borderColor: "var(--ink)" }}>
          <p style={{ fontFamily: "var(--f-display)", fontSize: 28, lineHeight: 1.1, maxWidth: "50ch" }}>
            <em style={{ color: "var(--accent)", fontStyle: "italic" }}>Challenge дня.</em>{" "}
            25 минут практический раунд с фокусом на trade-offs и метрики.
          </p>
          <ul style={{ marginTop: 20, display: "grid", gap: 10 }}>
            {[
              "Dual Mode Sprint: 15 мин theory + 15 мин practice",
              "Replay Debrief: пересоберите 1 неуспешный ответ в STAR",
              "Pressure Test: ограничьте ответ 90 сек + 2 fallback-стратегии",
            ].map((it, i) => (
              <li key={it} style={{ display: "grid", gridTemplateColumns: "24px 1fr", gap: 12, color: "oklch(0.82 0.01 60)", fontSize: 14 }}>
                <span className="mono" style={{ color: "var(--accent)" }}>{String(i + 1).padStart(2, "0")}</span>
                <span>{it}</span>
              </li>
            ))}
          </ul>
          <div className="row" style={{ marginTop: 22 }}>
            <button className="btn btn--accent" onClick={() => navigate("/interview")} type="button">Запустить спринт-интервью <Icon name="arrow" /></button>
            <button className="btn" style={{ border: "1px solid oklch(0.35 0.01 60)", color: "var(--bg)" }} type="button">Скопировать challenge</button>
          </div>
        </div>
      </section>

      <section style={{ marginTop: 40 }}>
        <header className="dash-section-head">
          <h2>Все интервью</h2>
          <div className="row">
            <div className="vacancy-search" style={{ padding: "8px 14px" }}>
              <Icon name="search" size={14} />
              <input
                placeholder="Поиск…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={{ width: 160 }}
              />
            </div>
            <div className="segmented">
              {(["all", "finished", "active", "expired"] as const).map((s) => (
                <button key={s} className={filter === s ? "is-active" : ""} onClick={() => setFilter(s)} type="button">
                  {s === "all" ? "Все" : s === "finished" ? "Завершённые" : s === "active" ? "Активные" : "Истёкшие"}
                </button>
              ))}
            </div>
          </div>
        </header>

        <div className="report-table">
          <div className="report-row head">
            <span></span>
            <span>Сессия</span>
            <span>Режим</span>
            <span>Статус</span>
            <span>Баллы</span>
            <span style={{ textAlign: "right" }}>Дата</span>
          </div>
          {filtered.map((r, i) => (
            <div className="report-row" key={r.session_id}>
              <span className="num">{String(i + 1).padStart(2, "0")}</span>
              <div>
                <strong>{r.role} · #{r.session_id.slice(0, 6)}</strong>
                <div className="mono" style={{ fontSize: 11, color: "var(--muted)", marginTop: 2 }}>{r.messages_total} сообщений</div>
              </div>
              <span className="mono" style={{ fontSize: 12, color: "var(--muted)" }}>{r.interview_mode}</span>
              <span><span className={`status ${r.status}`}>{r.status}</span></span>
              <span className="score mono">{r.overall_score ?? "—"}</span>
              <span className="mono" style={{ fontSize: 12, color: "var(--muted)", textAlign: "right" }}>
                {r.started_at ? new Date(r.started_at).toLocaleDateString("ru-RU", { day: "2-digit", month: "short" }) : ""}
              </span>
            </div>
          ))}
          {!filtered.length && (
            <div style={{ padding: 40, textAlign: "center", color: "var(--muted)" }}>Ничего не найдено. Сбросьте фильтры.</div>
          )}
        </div>
      </section>
    </main>
  );
}

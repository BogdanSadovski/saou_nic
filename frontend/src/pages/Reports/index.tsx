import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport, UserInterviewEntry } from "@/shared/api/reports";
import { EmptyState, FloatingInput, GlassButton, GlassCard, Skeleton } from "@/shared/ui";
import { ReportsCharts } from "./charts";
import { renderAndPrintReport } from "./pdfExport";

export default function ReportsPage() {
  const navigate = useNavigate();
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "finished" | "active" | "expired">("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [authIssue, setAuthIssue] = useState(false);

  const loadReport = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      setAuthIssue(false);
      const userReport = await reportsApi.getMyInterviewReport();
      setReport(userReport);
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      const message = e instanceof Error ? e.message : "Не удалось загрузить отчет";
      // 404 means the user doesn't have any interview data yet — render a
      // friendly empty state instead of a hard error so the page still
      // works (search, exports, recommendations) without a backend report.
      if (status === 404) {
        setReport(reportsApi.emptyReport());
        setError(null);
        setAuthIssue(false);
      } else {
        setError(message);
        setAuthIssue(status === 401 || /401|auth|authorization|token/i.test(message));
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadReport();
  }, [loadReport]);

  const filtered = useMemo(() => {
    if (!report) {
      return [];
    }
    const q = search.trim().toLowerCase();
    return report.recent_interviews.filter((item) => {
      const byStatus = statusFilter === "all" ? true : item.status === statusFilter;
      const bySearch =
        !q ||
        item.role.toLowerCase().includes(q) ||
        (item.vacancy_title || "").toLowerCase().includes(q) ||
        (item.current_topic || "").toLowerCase().includes(q);
      return byStatus && bySearch;
    });
  }, [report, search, statusFilter]);

  const innovationInsights = useMemo(() => {
    if (!report) {
      return null;
    }

    const timeline = [...report.timeline].sort((a, b) => a.date.localeCompare(b.date));
    const completedSeries = timeline.map((point) => point.completed);
    const startedSeries = timeline.map((point) => point.started);

    const tail = completedSeries.slice(-3);
    const prev = completedSeries.slice(-6, -3);
    const tailAvg = tail.length ? tail.reduce((acc, value) => acc + value, 0) / tail.length : 0;
    const prevAvg = prev.length ? prev.reduce((acc, value) => acc + value, 0) / prev.length : 0;
    const momentum = Math.round((tailAvg - prevAvg) * 20);

    let streakDays = 0;
    for (let i = completedSeries.length - 1; i >= 0; i -= 1) {
      if (completedSeries[i] > 0) {
        streakDays += 1;
      } else {
        break;
      }
    }

    const startedTotal = startedSeries.reduce((acc, value) => acc + value, 0);
    const completedTotal = completedSeries.reduce((acc, value) => acc + value, 0);
    const consistency = startedTotal > 0 ? Math.round((completedTotal / startedTotal) * 100) : 0;
    const reliability = Math.round((consistency * 0.6 + report.totals.completion_rate * 0.4));

    const primaryWeakness = report.top_weaknesses[0] || "системное мышление";
    const challenge = `Challenge дня: 25 минут на практический раунд по теме "${primaryWeakness}" с фокусом на trade-offs и метрики.`;

    const experiments = [
      `Dual Mode Sprint: 15 минут theory + 15 минут practice по теме ${primaryWeakness}.`,
      "Replay Debrief: пересоберите 1 неуспешный ответ в формате STAR и сравните с исходным.",
      "Pressure Test: ограничьте ответ 90 секундами и добавьте 2 fallback-стратегии.",
    ];

    return {
      momentum,
      streakDays,
      consistency,
      reliability,
      challenge,
      experiments,
    };
  }, [report]);

  const exportJson = () => {
    if (!report) {
      return;
    }
    const blob = new Blob([JSON.stringify(report, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `user-interview-report-${report.user_id}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const exportPdf = () => {
    if (!report) {
      return;
    }
    void renderAndPrintReport(report);
  };

  const interviewRow = (item: UserInterviewEntry) => {
    return (
      <div className="report-interview-row" key={item.session_id}>
        <div>
          <strong>{item.role}</strong>
          <small>{item.vacancy_title || "Без вакансии"}</small>
        </div>
        <div>
          <span className={`report-status report-status-${item.status}`}>{item.status}</span>
          <small>{item.interview_mode}</small>
        </div>
        <div>
          <strong>{item.overall_score ?? "-"}</strong>
          <small>score</small>
        </div>
        <div>
          <strong>{item.messages_total}</strong>
          <small>messages</small>
        </div>
      </div>
    );
  };

  if (loading) {
    return (
      <section className="page" aria-busy="true">
        <div className="section-header">
          <Skeleton width={260} height={28} />
          <Skeleton width={180} height={36} />
        </div>
        <GlassCard>
          <div className="report-metrics-grid">
            <Skeleton variant="card" />
            <Skeleton variant="card" />
            <Skeleton variant="card" />
          </div>
        </GlassCard>
        <GlassCard>
          <Skeleton count={4} />
        </GlassCard>
        <GlassCard>
          <Skeleton variant="card" height={140} />
        </GlassCard>
      </section>
    );
  }

  if (error) {
    return (
      <section className="page">
        <EmptyState
          icon="⚠️"
          title="Не удалось загрузить отчёт"
          hint={error}
          action={
            <GlassButton onClick={() => void loadReport()} type="button" variant="primary">
              Повторить
            </GlassButton>
          }
          secondaryAction={
            authIssue ? (
              <GlassButton onClick={() => navigate("/auth")} type="button" variant="ghost">
                Войти заново
              </GlassButton>
            ) : null
          }
        />
      </section>
    );
  }

  if (!report) {
    return (
      <section className="page">
        <EmptyState
          icon="📊"
          title="Отчёт пока пуст"
          hint="После первого пройденного интервью здесь появится подробная аналитика, графики прогресса и персональные рекомендации."
          action={
            <GlassButton onClick={() => navigate("/interview")} type="button" variant="primary">
              Начать интервью
            </GlassButton>
          }
        />
      </section>
    );
  }

  const hasInterviews = report.totals.total_interviews > 0;

  return (
    <section className="page">
      <div className="section-header">
        <h1>Отчет по пользователю</h1>
        <div className="report-actions">
          <GlassButton onClick={exportJson} type="button" variant="ghost">
            Экспорт JSON
          </GlassButton>
          <GlassButton onClick={exportPdf} type="button" variant="primary">
            Экспорт PDF
          </GlassButton>
        </div>
      </div>

      <div className="dashboard-grid report-metrics-grid">
        <GlassCard>
          <p className="muted">Всего интервью</p>
          <h2>{report.totals.total_interviews}</h2>
        </GlassCard>
        <GlassCard>
          <p className="muted">Завершено</p>
          <h2>{report.totals.completed_interviews}</h2>
        </GlassCard>
        <GlassCard>
          <p className="muted">Не завершено</p>
          <h2>{report.totals.in_progress_interviews + report.totals.expired_interviews}</h2>
        </GlassCard>
        <GlassCard>
          <p className="muted">Completion rate</p>
          <h2>{report.totals.completion_rate}%</h2>
        </GlassCard>
        <GlassCard>
          <p className="muted">Средний балл</p>
          <h2>{report.performance.average_score}</h2>
        </GlassCard>
        <GlassCard>
          <p className="muted">Лучший балл</p>
          <h2>{report.performance.best_score}</h2>
        </GlassCard>
      </div>

      {innovationInsights ? (
        <div className="dashboard-grid report-metrics-grid">
          <GlassCard>
            <p className="muted">Карьерный пульс</p>
            <h2>{innovationInsights.momentum > 0 ? `+${innovationInsights.momentum}` : innovationInsights.momentum}</h2>
            <p className="muted">Динамика завершения интервью за последние сессии</p>
          </GlassCard>
          <GlassCard>
            <p className="muted">Серия дней</p>
            <h2>{innovationInsights.streakDays}</h2>
            <p className="muted">Дней подряд с завершенными интервью</p>
          </GlassCard>
          <GlassCard>
            <p className="muted">Индекс надежности</p>
            <h2>{innovationInsights.reliability}%</h2>
            <p className="muted">Consistency: {innovationInsights.consistency}%</p>
          </GlassCard>
        </div>
      ) : null}

      {hasInterviews ? (
        <GlassCard>
          <h3>Графики прогресса</h3>
          <ReportsCharts report={report} />
        </GlassCard>
      ) : null}

      <div className="filters two-col">
        <FloatingInput label="Поиск по роли/теме/вакансии" onChange={(e) => setSearch(e.target.value)} value={search} />
        <div className="interview-field">
          <label htmlFor="status-filter">Статус интервью</label>
          <select
            id="status-filter"
            onChange={(event) => setStatusFilter(event.target.value as "all" | "finished" | "active" | "expired")}
            value={statusFilter}
          >
            <option value="all">Все</option>
            <option value="finished">Завершенные</option>
            <option value="active">В процессе</option>
            <option value="expired">Истекшие</option>
          </select>
        </div>
      </div>

      <div className="home-grid">
        <GlassCard>
          <h3>Сильные стороны</h3>
          <ul className="report-bullet-list">
            {report.top_strengths.map((item) => (
              <li key={item}>{item}</li>
            ))}
            {report.top_strengths.length === 0 ? <li>Недостаточно данных</li> : null}
          </ul>
        </GlassCard>
        <GlassCard>
          <h3>Слабые стороны</h3>
          <ul className="report-bullet-list">
            {report.top_weaknesses.map((item) => (
              <li key={item}>{item}</li>
            ))}
            {report.top_weaknesses.length === 0 ? <li>Недостаточно данных</li> : null}
          </ul>
        </GlassCard>
      </div>

      <GlassCard>
        <h3>Рекомендации к развитию</h3>
        <ul className="report-bullet-list">
          {report.top_recommendations.map((item) => (
            <li key={item}>{item}</li>
          ))}
          {report.top_recommendations.length === 0 ? <li>Недостаточно данных</li> : null}
        </ul>
      </GlassCard>

      {innovationInsights ? (
        <GlassCard className="report-innovation-card">
          <h3>Innovation Sprint</h3>
          <p className="muted">{innovationInsights.challenge}</p>
          <ul className="report-bullet-list">
            {innovationInsights.experiments.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
          <div className="report-actions">
            <GlassButton onClick={() => navigate("/interview")} type="button">
              Запустить спринт-интервью
            </GlassButton>
            <GlassButton
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(innovationInsights.challenge);
                } catch {
                  // noop: clipboard may be unavailable in insecure contexts.
                }
              }}
              type="button"
              variant="ghost"
            >
              Скопировать challenge
            </GlassButton>
          </div>
        </GlassCard>
      ) : null}

      <GlassCard>
        <h3>Завершенные интервью</h3>
        {report.completed_interviews.slice(0, 12).map(interviewRow)}
        {report.completed_interviews.length === 0 ? <p className="muted">Пока нет завершенных интервью</p> : null}
      </GlassCard>

      <GlassCard>
        <h3>Незавершенные интервью</h3>
        {report.incomplete_interviews.slice(0, 12).map(interviewRow)}
        {report.incomplete_interviews.length === 0 ? <p className="muted">Незавершенных интервью нет</p> : null}
      </GlassCard>

      <GlassCard>
        <h3>Последние интервью (фильтруемые)</h3>
        {filtered.map(interviewRow)}
        {filtered.length === 0 && hasInterviews ? (
          <EmptyState
            icon="🔍"
            title="Ничего не найдено"
            hint="По текущим фильтрам ни одно интервью не подошло. Попробуйте сбросить поиск или статус."
            action={
              <GlassButton
                onClick={() => {
                  setSearch("");
                  setStatusFilter("all");
                }}
                type="button"
                variant="ghost"
              >
                Сбросить фильтры
              </GlassButton>
            }
          />
        ) : null}
        {!hasInterviews ? (
          <EmptyState
            icon="🚀"
            title="У вас ещё нет интервью"
            hint="Пройдите первое практическое или теоретическое интервью — и сюда подтянутся метрики, тренд оценок и рекомендации."
            action={
              <GlassButton onClick={() => navigate("/interview")} type="button" variant="primary">
                Начать первое интервью
              </GlassButton>
            }
          />
        ) : null}
      </GlassCard>
    </section>
  );
}

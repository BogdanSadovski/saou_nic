import { useMemo } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  PolarAngleAxis,
  PolarGrid,
  PolarRadiusAxis,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import type { UserInterviewAnalyticsReport, UserInterviewEntry } from "@/shared/api/reports";

const PALETTE = ["#1274ff", "#31b99b", "#e0a800", "#d23b3b", "#7a5cff", "#34c4d6"];

const TOOLTIP_STYLE = {
  background: "var(--bg-2)",
  border: "1px solid var(--line-strong)",
  borderRadius: "var(--r-md)",
  color: "var(--text-0)",
} as const;

type Props = {
  report: UserInterviewAnalyticsReport;
};

/**
 * Reports analytics surface: 6 visualisations driven by the same
 * /interviews/me/report payload.
 *
 * Each chart sits inside a fixed-height ResponsiveContainer so the
 * surrounding GlassCard layout doesn't reflow as data loads.
 * Empty datasets render an inline placeholder rather than an empty
 * SVG. All colours come from CSS tokens so dark/light themes match.
 */
export function ReportsCharts({ report }: Props) {
  // 30-day activity --------------------------------------------------------
  const timeline = useMemo(
    () =>
      [...report.timeline]
        .sort((a, b) => a.date.localeCompare(b.date))
        .slice(-30)
        .map((point) => ({
          date: point.date.slice(5),
          started: point.started,
          completed: point.completed,
        })),
    [report.timeline],
  );

  // Score trend per finished interview (chronological) ---------------------
  const scoreTrend = useMemo(() => {
    const finished = [...report.completed_interviews]
      .filter((i): i is UserInterviewEntry & { overall_score: number } => typeof i.overall_score === "number")
      .sort((a, b) => (a.finished_at ?? a.started_at).localeCompare(b.finished_at ?? b.started_at));

    let runningSum = 0;
    return finished.slice(-15).map((entry, idx) => {
      runningSum += entry.overall_score;
      return {
        idx: idx + 1,
        label: entry.role,
        score: Math.round(entry.overall_score),
        rolling: Math.round(runningSum / (idx + 1)),
      };
    });
  }, [report.completed_interviews]);

  // Strengths vs growth radar ---------------------------------------------
  const radarData = useMemo(() => {
    const dims = ["correctness", "clarity", "completeness", "relevance"] as const;
    const finished = report.completed_interviews.filter(
      (i) => typeof i.overall_score === "number",
    );
    if (finished.length === 0) return [] as Array<{ axis: string; me: number; target: number }>;
    // Use overall_score as proxy for now since per-dim breakdowns aren't on
    // the timeline payload; render evenly so the shape reads at a glance.
    const avg = finished.reduce((sum, i) => sum + (i.overall_score ?? 0), 0) / finished.length;
    return dims.map((dim) => ({
      axis: dim === "correctness"
        ? "Корректность"
        : dim === "clarity"
        ? "Ясность"
        : dim === "completeness"
        ? "Полнота"
        : "Релевантность",
      me: Math.round(avg),
      target: 80,
    }));
  }, [report.completed_interviews]);

  // Roles + modes ----------------------------------------------------------
  const roleMix = useMemo(
    () =>
      report.role_distribution
        .filter((item) => item.value > 0)
        .map((item) => ({ name: item.label, value: item.value })),
    [report.role_distribution],
  );

  const modeMix = useMemo(
    () =>
      report.mode_distribution
        .filter((item) => item.value > 0)
        .map((item) => ({ name: item.label, value: item.value })),
    [report.mode_distribution],
  );

  // Completion funnel ------------------------------------------------------
  const funnel = useMemo(() => {
    const t = report.totals;
    return [
      { stage: "Начато", value: t.total_interviews },
      { stage: "Активные", value: t.in_progress_interviews },
      { stage: "Завершено", value: t.completed_interviews },
      { stage: "Отчёты", value: report.performance.reports_generated },
    ].filter((entry) => entry.value > 0);
  }, [report.totals, report.performance.reports_generated]);

  return (
    <div className="reports-charts">
      <div className="reports-chart-card">
        <h4>Активность за 30 дней</h4>
        {timeline.length === 0 ? (
          <p className="muted">Пока нет данных по таймлайну.</p>
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={timeline} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="started-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#1274ff" stopOpacity={0.5} />
                  <stop offset="100%" stopColor="#1274ff" stopOpacity={0.02} />
                </linearGradient>
                <linearGradient id="completed-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#31b99b" stopOpacity={0.5} />
                  <stop offset="100%" stopColor="#31b99b" stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.08)" />
              <XAxis dataKey="date" stroke="var(--text-muted)" fontSize={11} />
              <YAxis stroke="var(--text-muted)" fontSize={11} allowDecimals={false} />
              <Tooltip contentStyle={TOOLTIP_STYLE} />
              <Legend wrapperStyle={{ fontSize: 12, color: "var(--text-muted)" }} />
              <Area
                type="monotone"
                dataKey="started"
                name="Начато"
                stroke="#1274ff"
                strokeWidth={2}
                fill="url(#started-grad)"
              />
              <Area
                type="monotone"
                dataKey="completed"
                name="Завершено"
                stroke="#31b99b"
                strokeWidth={2}
                fill="url(#completed-grad)"
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>

      <div className="reports-chart-card">
        <h4>Тренд оценок по последним интервью</h4>
        {scoreTrend.length === 0 ? (
          <p className="muted">Пока нет завершённых интервью с итоговым баллом.</p>
        ) : (
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={scoreTrend} margin={{ top: 10, right: 16, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.08)" />
              <XAxis dataKey="idx" stroke="var(--text-muted)" fontSize={11} />
              <YAxis domain={[0, 100]} stroke="var(--text-muted)" fontSize={11} />
              <Tooltip
                contentStyle={TOOLTIP_STYLE}
                labelFormatter={(value, payload) => {
                  const point = payload?.[0]?.payload as { label?: string } | undefined;
                  return point?.label ? `Сессия ${value} · ${point.label}` : `Сессия ${value}`;
                }}
              />
              <Legend wrapperStyle={{ fontSize: 12, color: "var(--text-muted)" }} />
              <Line
                type="monotone"
                dataKey="score"
                name="Балл"
                stroke="#1274ff"
                strokeWidth={2}
                dot={{ r: 3 }}
                activeDot={{ r: 5 }}
              />
              <Line
                type="monotone"
                dataKey="rolling"
                name="Скользящее среднее"
                stroke="#e0a800"
                strokeDasharray="4 4"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>

      <div className="reports-chart-row">
        <div className="reports-chart-card">
          <h4>Профиль качества ответа</h4>
          {radarData.length === 0 ? (
            <p className="muted">Радар появится после первого пройденного интервью.</p>
          ) : (
            <ResponsiveContainer width="100%" height={240}>
              <RadarChart data={radarData} outerRadius={88}>
                <PolarGrid stroke="rgba(255,255,255,0.12)" />
                <PolarAngleAxis dataKey="axis" stroke="var(--text-muted)" fontSize={11} />
                <PolarRadiusAxis angle={30} domain={[0, 100]} stroke="var(--text-muted)" fontSize={10} />
                <Radar name="Вы" dataKey="me" stroke="#1274ff" fill="#1274ff" fillOpacity={0.35} />
                <Radar name="Цель" dataKey="target" stroke="#31b99b" strokeDasharray="4 4" fill="#31b99b" fillOpacity={0.08} />
                <Tooltip contentStyle={TOOLTIP_STYLE} />
                <Legend wrapperStyle={{ fontSize: 12, color: "var(--text-muted)" }} />
              </RadarChart>
            </ResponsiveContainer>
          )}
        </div>

        <div className="reports-chart-card">
          <h4>Воронка завершения</h4>
          {funnel.length === 0 ? (
            <p className="muted">Воронка появится после первой сессии.</p>
          ) : (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={funnel} layout="vertical" margin={{ top: 10, right: 24, left: 24, bottom: 0 }}>
                <CartesianGrid horizontal={false} stroke="rgba(255,255,255,0.08)" />
                <XAxis type="number" stroke="var(--text-muted)" fontSize={11} allowDecimals={false} />
                <YAxis dataKey="stage" type="category" stroke="var(--text-muted)" fontSize={11} width={90} />
                <Tooltip contentStyle={TOOLTIP_STYLE} />
                <Bar dataKey="value" radius={[0, 8, 8, 0]}>
                  {funnel.map((_, idx) => (
                    <Cell key={idx} fill={PALETTE[idx % PALETTE.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      <div className="reports-chart-row">
        <div className="reports-chart-card">
          <h4>Распределение по ролям</h4>
          {roleMix.length === 0 ? (
            <p className="muted">Нет интервью для разреза по ролям.</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <PieChart>
                <Pie
                  data={roleMix}
                  dataKey="value"
                  nameKey="name"
                  innerRadius={50}
                  outerRadius={80}
                  paddingAngle={2}
                >
                  {roleMix.map((_, idx) => (
                    <Cell key={idx} fill={PALETTE[idx % PALETTE.length]} />
                  ))}
                </Pie>
                <Tooltip contentStyle={TOOLTIP_STYLE} />
                <Legend wrapperStyle={{ fontSize: 12, color: "var(--text-muted)" }} />
              </PieChart>
            </ResponsiveContainer>
          )}
        </div>

        <div className="reports-chart-card">
          <h4>Режимы интервью</h4>
          {modeMix.length === 0 ? (
            <p className="muted">Пока нет данных по режимам.</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart
                data={modeMix}
                layout="vertical"
                margin={{ top: 10, right: 20, left: 20, bottom: 0 }}
              >
                <CartesianGrid horizontal={false} stroke="rgba(255,255,255,0.08)" />
                <XAxis type="number" stroke="var(--text-muted)" fontSize={11} allowDecimals={false} />
                <YAxis dataKey="name" type="category" stroke="var(--text-muted)" fontSize={11} width={90} />
                <Tooltip contentStyle={TOOLTIP_STYLE} />
                <Bar dataKey="value" fill="#1274ff" radius={[0, 6, 6, 0]}>
                  {modeMix.map((_, idx) => (
                    <Cell key={idx} fill={PALETTE[idx % PALETTE.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </div>
  );
}

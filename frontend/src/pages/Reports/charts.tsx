import { useMemo } from "react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";

const PALETTE = ["#1274ff", "#31b99b", "#e0a800", "#d23b3b", "#7a5cff", "#34c4d6"];

type Props = {
  report: UserInterviewAnalyticsReport;
};

/**
 * Reports page analytics charts.
 *
 * Three views, computed from the same report payload:
 *   - Activity timeline (daily started vs completed)
 *   - Role mix (pie)
 *   - Mode mix (horizontal bars)
 *
 * Each chart sits inside a fixed-height ResponsiveContainer so the
 * surrounding GlassCard layout doesn't reflow as data loads. Empty
 * datasets render an inline placeholder rather than an empty SVG.
 */
export function ReportsCharts({ report }: Props) {
  const timeline = useMemo(
    () =>
      [...report.timeline]
        .sort((a, b) => a.date.localeCompare(b.date))
        .slice(-30)
        .map((point) => ({
          date: point.date.slice(5), // MM-DD
          started: point.started,
          completed: point.completed,
        })),
    [report.timeline],
  );

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
              <Tooltip
                contentStyle={{
                  background: "var(--bg-2)",
                  border: "1px solid var(--line-strong)",
                  borderRadius: "var(--r-md)",
                  color: "var(--text-0)",
                }}
              />
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
                <Tooltip
                  contentStyle={{
                    background: "var(--bg-2)",
                    border: "1px solid var(--line-strong)",
                    borderRadius: "var(--r-md)",
                    color: "var(--text-0)",
                  }}
                />
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
                <Tooltip
                  contentStyle={{
                    background: "var(--bg-2)",
                    border: "1px solid var(--line-strong)",
                    borderRadius: "var(--r-md)",
                    color: "var(--text-0)",
                  }}
                />
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

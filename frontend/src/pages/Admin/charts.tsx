import { useMemo } from "react";
import {
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

import type { AdminDashboardStats } from "@/shared/api/admin";

const PALETTE = ["#1274ff", "#31b99b", "#e0a800", "#d23b3b", "#7a5cff", "#34c4d6"];

type Props = { stats: AdminDashboardStats };

const distributionToData = (dist: Record<string, number>) =>
  Object.entries(dist)
    .filter(([, value]) => value > 0)
    .map(([name, value]) => ({ name, value }));

/**
 * Two side-by-side charts powered by the admin /dashboard/stats payload:
 * role distribution (donut) + subscription tier breakdown (vertical bars).
 */
export function AdminCharts({ stats }: Props) {
  const roleData = useMemo(() => distributionToData(stats.role_distribution), [stats.role_distribution]);
  const tierData = useMemo(() => distributionToData(stats.subscription_tiers), [stats.subscription_tiers]);

  const tooltipStyle = {
    background: "var(--bg-2)",
    border: "1px solid var(--line-strong)",
    borderRadius: "var(--r-md)",
    color: "var(--text-0)",
  };

  return (
    <div className="reports-chart-row">
      <div className="reports-chart-card">
        <h4>Распределение ролей</h4>
        {roleData.length === 0 ? (
          <p className="muted">Нет данных по ролям.</p>
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <PieChart>
              <Pie
                data={roleData}
                dataKey="value"
                nameKey="name"
                innerRadius={50}
                outerRadius={80}
                paddingAngle={2}
              >
                {roleData.map((_, idx) => (
                  <Cell key={idx} fill={PALETTE[idx % PALETTE.length]} />
                ))}
              </Pie>
              <Tooltip contentStyle={tooltipStyle} />
              <Legend wrapperStyle={{ fontSize: 12, color: "var(--text-muted)" }} />
            </PieChart>
          </ResponsiveContainer>
        )}
      </div>

      <div className="reports-chart-card">
        <h4>Активные подписки по тарифам</h4>
        {tierData.length === 0 ? (
          <p className="muted">Активных подписок ещё нет.</p>
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={tierData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.08)" />
              <XAxis dataKey="name" stroke="var(--text-muted)" fontSize={11} />
              <YAxis stroke="var(--text-muted)" fontSize={11} allowDecimals={false} />
              <Tooltip contentStyle={tooltipStyle} />
              <Bar dataKey="value" radius={[8, 8, 0, 0]}>
                {tierData.map((_, idx) => (
                  <Cell key={idx} fill={PALETTE[idx % PALETTE.length]} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}

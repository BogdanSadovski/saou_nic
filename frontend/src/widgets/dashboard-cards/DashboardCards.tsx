import { useEffect, useState } from "react";

import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { useTranslation } from "@/shared/i18n";
import { GlassCard, Skeleton } from "@/shared/ui";

/**
 * Top-row KPI cards on the dashboard. Pulls real metrics from
 * `/api/interviews/me/report` so the numbers actually reflect the
 * user's activity instead of the previous hard-coded "24/86/91%"
 * triplet that confused new users.
 *
 * While the report is loading, each card swaps its value with a
 * shimmering skeleton — no visible layout shift when the data lands.
 */
export function DashboardCards() {
  const t = useTranslation();
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await reportsApi.getMyInterviewReport();
        if (!cancelled) setReport(data);
      } catch (e) {
        // 404 = first-run user, render zeros via emptyReport. Anything else
        // we just leave as null and show "—" so the dashboard still loads.
        const status = (e as { response?: { status?: number } })?.response?.status;
        if (!cancelled && status === 404) {
          setReport(reportsApi.emptyReport());
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const totalInterviews = report?.totals.total_interviews ?? 0;
  const avgScore = report?.performance.average_score ?? 0;
  // Backend already returns completion_rate as 0..100 (math.Round of
  // completed/total*100 in interview-service), no extra multiply.
  const completionRate = Math.round(report?.totals.completion_rate ?? 0);

  const fmt = (value: number, suffix = "") =>
    value > 0 ? `${Math.round(value)}${suffix}` : "—";

  const stats = [
    { label: t.interviews, value: fmt(totalInterviews) },
    { label: t.avgScore, value: fmt(avgScore) },
    { label: t.resumeMatch, value: completionRate > 0 ? `${completionRate}%` : "—" },
  ];

  return (
    <section className="dashboard-grid">
      {stats.map((stat) => (
        <GlassCard className="stat-card" key={stat.label}>
          <p className="muted">{stat.label}</p>
          {loading ? <Skeleton width={80} height={32} /> : <h3>{stat.value}</h3>}
        </GlassCard>
      ))}
    </section>
  );
}

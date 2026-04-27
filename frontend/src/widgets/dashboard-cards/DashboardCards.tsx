import { useTranslation } from "@/shared/i18n";
import { GlassCard, Skeleton } from "@/shared/ui";

export function DashboardCards() {
  const t = useTranslation();

  const stats = [
    { label: t.interviews, value: "24" },
    { label: t.avgScore, value: "86" },
    { label: t.resumeMatch, value: "91%" },
  ];

  return (
    <section className="dashboard-grid">
      {stats.map((stat) => (
        <GlassCard className="stat-card" key={stat.label}>
          <p className="muted">{stat.label}</p>
          <h3>{stat.value}</h3>
          <Skeleton className="stat-skeleton" />
        </GlassCard>
      ))}
    </section>
  );
}

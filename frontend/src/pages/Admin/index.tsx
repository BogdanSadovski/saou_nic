import { useEffect, useMemo, useState } from "react";

import { adminApi } from "@/shared/api";
import { useTranslation } from "@/shared/i18n";
import { FloatingInput, GlassCard } from "@/shared/ui";

type AdminUser = {
  name: string;
  status: "Active" | "Review" | "Paused";
  score: number;
};

export default function AdminPage() {
  const [rows, setRows] = useState<AdminUser[]>([]);
  const [averageScore, setAverageScore] = useState(0);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("All");
  const t = useTranslation();

  useEffect(() => {
    const load = async () => {
      try {
        const [users, stats] = await Promise.all([
          adminApi.listUsers(),
          adminApi.getDashboardStats(),
        ]);

        setRows(
          users.map((user) => ({
            name: `${user.first_name ?? ""} ${user.last_name ?? ""}`.trim() || user.username || "Пользователь",
            status: user.status === "active" ? "Active" : user.status === "suspended" ? "Paused" : "Review",
            score: user.score ?? 0,
          })),
        );
        setAverageScore(Math.round(stats.average_score ?? 0));
      } catch {
        setRows([]);
        setAverageScore(0);
      }
    };

    void load();
  }, []);

  const filtered = useMemo(() => {
    return rows.filter((row) => {
      const matchName = row.name.toLowerCase().includes(query.trim().toLowerCase());
      const matchStatus = status === "All" ? true : row.status === status;
      return matchName && matchStatus;
    });
  }, [query, status]);

  const getStatusText = (s: string): string => {
    const statusMap: Record<string, string> = {
      "Active": t.active,
      "Review": t.review,
      "Paused": t.paused,
      "All": "Все",
    };
    return statusMap[s] || s;
  };

  return (
    <section className="page">
      <h1>{t.adminAnalytics}</h1>

      <div className="filters two-col">
        <FloatingInput label={t.searchUser} onChange={(event) => setQuery(event.target.value)} value={query} />
        <label className="status-filter glass-card">
          <span>{t.status}</span>
          <select onChange={(event) => setStatus(event.target.value)} value={status}>
            <option value="All">Все</option>
            <option value="Active">Активный</option>
            <option value="Review">На проверке</option>
            <option value="Paused">Приостановлен</option>
          </select>
        </label>
      </div>

      <div className="two-col">
        <GlassCard>
          <h3>{t.users}</h3>
          <table className="admin-table">
            <thead>
              <tr>
                <th>{t.name}</th>
                <th>{t.status}</th>
                <th>{t.score}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((row) => (
                <tr key={row.name}>
                  <td>{row.name}</td>
                  <td>{getStatusText(row.status)}</td>
                  <td>{row.score}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </GlassCard>
        <GlassCard>
          <h3>{t.platformHealth}</h3>
          <p className="muted">{t.frontendDemoMode}</p>
          <p className="muted">{t.activeCandidates} {rows.filter((r) => r.status === "Active").length}</p>
          <p className="muted">{t.averageScore} {averageScore}</p>
        </GlassCard>
      </div>
    </section>
  );
}

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { adminApi } from "@/shared/api";
import type {
  AdminAuditLog,
  AdminDashboardStats,
  AdminSubscription,
  AdminUser,
} from "@/shared/api/admin";
import { formatBYN } from "@/shared/lib/currency";
import { useToast } from "@/shared/ui";

import { AdminCharts } from "./charts";

type Tab = "users" | "subscriptions" | "audit";

const ROLE_OPTIONS = ["user", "admin", "moderator"];
const STATUS_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "", label: "Все" },
  { value: "active", label: "Активные" },
  { value: "suspended", label: "Приостановленные" },
  { value: "banned", label: "Забаненные" },
  { value: "pending", label: "На проверке" },
];

const formatDate = (raw?: string) => {
  if (!raw) return "—";
  const d = new Date(raw);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleDateString("ru-RU");
};

const formatCurrency = (value: number) =>
  value > 0 ? formatBYN(Math.round(value)) : "—";

const tierMonthlyPriceBYN = (tier: string): number => {
  switch (tier) {
    case "starter":
    case "basic":
      return 29;
    case "pro":
      return 65;
    case "team":
    case "enterprise":
      return 159;
    default:
      return 0;
  }
};

const userDisplayName = (u: AdminUser) => {
  const fn = `${u.first_name ?? ""} ${u.last_name ?? ""}`.trim();
  return fn || u.username || u.email || u.id;
};

export default function AdminPage() {
  const navigate = useNavigate();
  const { pushToast } = useToast();

  const [tab, setTab] = useState<Tab>("users");

  const [stats, setStats] = useState<AdminDashboardStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);
  const [statsError, setStatsError] = useState<string | null>(null);

  const [users, setUsers] = useState<AdminUser[]>([]);
  const [usersTotal, setUsersTotal] = useState(0);
  const [usersLoading, setUsersLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("");

  const [subscriptions, setSubscriptions] = useState<AdminSubscription[]>([]);
  const [subsLoading, setSubsLoading] = useState(false);

  const [auditLogs, setAuditLogs] = useState<AdminAuditLog[]>([]);
  const [auditLoading, setAuditLoading] = useState(false);

  const [pendingId, setPendingId] = useState<string | null>(null);

  const reportError = useCallback(
    (e: unknown, fallback: string) => {
      const status = (e as { response?: { status?: number } })?.response?.status;
      if (status === 401 || status === 403) {
        pushToast("Требуются права администратора");
        navigate("/auth");
        return;
      }
      const msg = e instanceof Error ? e.message : fallback;
      pushToast(msg);
    },
    [navigate, pushToast],
  );

  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const data = await adminApi.getDashboardStats();
      setStats(data);
      setStatsError(null);
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      if (status === 401 || status === 403) {
        setStatsError("Доступ к админ-панели только для роли admin.");
      } else {
        setStatsError("Не удалось загрузить статистику.");
      }
    } finally {
      setStatsLoading(false);
    }
  }, []);

  const loadUsers = useCallback(async () => {
    setUsersLoading(true);
    try {
      const resp = await adminApi.listUsers({
        page: 1,
        pageSize: 50,
        search: query.trim() || undefined,
        status: statusFilter || undefined,
      });
      setUsers(resp.items);
      setUsersTotal(resp.total);
    } catch (e) {
      reportError(e, "Не удалось загрузить пользователей");
      setUsers([]);
      setUsersTotal(0);
    } finally {
      setUsersLoading(false);
    }
  }, [query, statusFilter, reportError]);

  const loadSubscriptions = useCallback(async () => {
    setSubsLoading(true);
    try {
      const resp = await adminApi.listSubscriptions({ page: 1, pageSize: 50 });
      setSubscriptions(resp.items);
    } catch (e) {
      reportError(e, "Не удалось загрузить подписки");
      setSubscriptions([]);
    } finally {
      setSubsLoading(false);
    }
  }, [reportError]);

  const loadAuditLogs = useCallback(async () => {
    setAuditLoading(true);
    try {
      const resp = await adminApi.listAuditLogs({ page: 1, pageSize: 100 });
      setAuditLogs(resp.items);
    } catch (e) {
      reportError(e, "Не удалось загрузить журнал");
      setAuditLogs([]);
    } finally {
      setAuditLoading(false);
    }
  }, [reportError]);

  useEffect(() => {
    void loadStats();
    void loadUsers();
  }, [loadStats, loadUsers]);

  useEffect(() => {
    if (tab === "subscriptions" && subscriptions.length === 0) {
      void loadSubscriptions();
    }
    if (tab === "audit" && auditLogs.length === 0) {
      void loadAuditLogs();
    }
  }, [tab, subscriptions.length, auditLogs.length, loadSubscriptions, loadAuditLogs]);

  const filteredUsers = useMemo(() => {
    const q = query.trim().toLowerCase();
    return users.filter((u) => {
      if (statusFilter && u.status !== statusFilter) return false;
      if (!q) return true;
      const haystack = [u.email, u.username, u.first_name, u.last_name, u.id]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(q);
    });
  }, [users, query, statusFilter]);

  const performAction = async (
    user: AdminUser,
    action: "suspend" | "activate" | "ban",
  ) => {
    setPendingId(user.id);
    try {
      if (action === "suspend") await adminApi.suspendUser(user.id);
      if (action === "activate") await adminApi.activateUser(user.id);
      if (action === "ban") await adminApi.banUser(user.id);
      pushToast("Готово");
      await loadUsers();
    } catch (e) {
      reportError(e, "Не удалось выполнить действие");
    } finally {
      setPendingId(null);
    }
  };

  const handleRoleChange = async (user: AdminUser, role: string) => {
    if (!role || role === user.role) return;
    setPendingId(user.id);
    try {
      await adminApi.changeUserRole(user.id, role);
      pushToast(`Роль обновлена: ${role}`);
      await loadUsers();
    } catch (e) {
      reportError(e, "Не удалось сменить роль");
    } finally {
      setPendingId(null);
    }
  };

  return (
    <>
      <span className="eyebrow">Админ · консоль</span>
      <header className="row-between" style={{ alignItems: "end", marginTop: 8 }}>
        <h1 className="expr-headline" style={{ fontSize: 72 }}>
          <span className="ital">Админ</span>.
        </h1>
        <button
          className="btn btn--ghost btn--sm"
          onClick={() => {
            void loadStats();
            void loadUsers();
            if (tab === "subscriptions") void loadSubscriptions();
            if (tab === "audit") void loadAuditLogs();
          }}
          type="button"
        >
          Обновить
        </button>
      </header>

      {/* KPI strip */}
      <div className="sysbar reveal" style={{ marginTop: 20 }}>
        <span>
          <span className="dot"></span>
          <span className="k">всего</span>
          <span className="v">{statsLoading ? "…" : (stats?.total_users ?? "—")}</span>
        </span>
        <span>
          <span className="k">активных</span>
          <span className="v">{statsLoading ? "…" : (stats?.active_users ?? "—")}</span>
        </span>
        <span>
          <span className="k">новых сегодня</span>
          <span className="v">{statsLoading ? "…" : (stats?.new_users_today ?? "—")}</span>
        </span>
        <span>
          <span className="k">подписок</span>
          <span className="v">{statsLoading ? "…" : (stats?.active_subscriptions ?? "—")}</span>
        </span>
        <span>
          <span className="k">доход (мес)</span>
          <span className="v">{statsLoading ? "…" : formatCurrency(stats?.revenue_this_month ?? 0)}</span>
        </span>
      </div>

      {statsError ? (
        <section className="profile-card" style={{ marginTop: 24 }}>
          <h3>Нет доступа к админ-метрикам</h3>
          <p className="muted">{statsError}</p>
          <button className="btn btn--primary btn--sm" onClick={() => void loadStats()} type="button" style={{ marginTop: 12 }}>
            Повторить
          </button>
        </section>
      ) : (
        <section style={{ marginTop: 24 }}>
          <header className="dash-section-head" style={{ marginBottom: 16 }}>
            <h2 style={{ fontSize: 24 }}>Аналитика платформы</h2>
          </header>
          {!statsLoading && stats ? <AdminCharts stats={stats} /> : null}
        </section>
      )}

      {/* Tab nav */}
      <div className="segmented" role="tablist" aria-label="Разделы" style={{ marginTop: 28 }}>
        <button className={tab === "users" ? "is-active" : ""} onClick={() => setTab("users")} type="button">
          Пользователи
        </button>
        <button className={tab === "subscriptions" ? "is-active" : ""} onClick={() => setTab("subscriptions")} type="button">
          Подписки
        </button>
        <button className={tab === "audit" ? "is-active" : ""} onClick={() => setTab("audit")} type="button">
          Журнал
        </button>
      </div>

      {tab === "users" && (
        <section className="profile-card" style={{ marginTop: 18 }}>
          <header className="dash-section-head">
            <h2 style={{ fontSize: 24 }}>Пользователи</h2>
            <span className="eyebrow">{usersTotal} всего</span>
          </header>

          <div className="admin-filters">
            <div className="field">
              <label>Поиск</label>
              <input
                className="input"
                onChange={(e) => setQuery(e.target.value)}
                placeholder="email · имя · username"
                value={query}
              />
            </div>
            <div className="field">
              <label>Статус</label>
              <div className="segmented">
                {STATUS_OPTIONS.map((opt) => (
                  <button
                    key={opt.value || "all"}
                    className={statusFilter === opt.value ? "is-active" : ""}
                    onClick={() => setStatusFilter(opt.value)}
                    type="button"
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="admin-table">
            <div className="admin-row head">
              <span>#</span>
              <span>Пользователь</span>
              <span>Статус</span>
              <span>Создан</span>
              <span>Роль</span>
              <span style={{ textAlign: "right" }}>Действия</span>
            </div>

            {usersLoading ? (
              <div className="muted" style={{ padding: 20 }}>Загружаем…</div>
            ) : filteredUsers.length === 0 ? (
              <div className="muted" style={{ padding: 20 }}>
                {usersTotal === 0 ? "Пока нет пользователей" : "Ничего не найдено"}
              </div>
            ) : (
              filteredUsers.map((u, i) => (
                <div className="admin-row" key={u.id}>
                  <span className="num">{String(i + 1).padStart(2, "0")}</span>
                  <div>
                    <strong>{userDisplayName(u)}</strong>
                    <span className="sub">{u.email ?? "—"}</span>
                  </div>
                  <span>
                    <span className={`status ${u.status ?? "active"}`}>{u.status ?? "active"}</span>
                  </span>
                  <span className="date-mono">{formatDate(u.created_at)}</span>
                  <select
                    className="admin-select"
                    disabled={pendingId === u.id}
                    onChange={(e) => void handleRoleChange(u, e.target.value)}
                    value={u.role ?? "user"}
                  >
                    {ROLE_OPTIONS.map((r) => (
                      <option key={r} value={r}>{r}</option>
                    ))}
                  </select>
                  <div className="admin-actions">
                    {u.status === "suspended" || u.status === "banned" ? (
                      <button
                        className="btn btn--ghost btn--sm"
                        disabled={pendingId === u.id}
                        onClick={() => void performAction(u, "activate")}
                        type="button"
                      >
                        Активировать
                      </button>
                    ) : (
                      <button
                        className="btn btn--ghost btn--sm"
                        disabled={pendingId === u.id}
                        onClick={() => void performAction(u, "suspend")}
                        type="button"
                      >
                        Приостановить
                      </button>
                    )}
                    <button
                      className="btn btn--ghost btn--sm"
                      disabled={pendingId === u.id}
                      onClick={() => void performAction(u, "ban")}
                      type="button"
                    >
                      Забанить
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>
      )}

      {tab === "subscriptions" && (
        <section className="profile-card" style={{ marginTop: 18 }}>
          <header className="dash-section-head">
            <h2 style={{ fontSize: 24 }}>Подписки</h2>
          </header>

          <div className="admin-table">
            <div className="admin-row head" style={{ gridTemplateColumns: "40px 1.4fr 1fr 1fr 1fr 1fr" }}>
              <span>#</span>
              <span>Тариф · юзер</span>
              <span>Статус</span>
              <span>Сумма</span>
              <span>Старт</span>
              <span>До</span>
            </div>
            {subsLoading ? (
              <div className="muted" style={{ padding: 20 }}>Загружаем…</div>
            ) : subscriptions.length === 0 ? (
              <div className="muted" style={{ padding: 20 }}>Подписок пока нет</div>
            ) : (
              subscriptions.map((s, i) => (
                <div className="admin-row" key={s.id} style={{ gridTemplateColumns: "40px 1.4fr 1fr 1fr 1fr 1fr" }}>
                  <span className="num">{String(i + 1).padStart(2, "0")}</span>
                  <div>
                    <strong>{s.tier}</strong>
                    <span className="sub">{s.user_id}</span>
                  </div>
                  <span>
                    <span className={`status ${s.status ?? "active"}`}>{s.status ?? "—"}</span>
                  </span>
                  <span className="date-mono">
                    {s.amount && s.amount > 0
                      ? formatCurrency(s.amount)
                      : formatCurrency(tierMonthlyPriceBYN(s.tier))}
                  </span>
                  <span className="date-mono">{formatDate(s.start_date)}</span>
                  <span className="date-mono">{formatDate(s.end_date)}</span>
                </div>
              ))
            )}
          </div>
        </section>
      )}

      {tab === "audit" && (
        <section className="profile-card" style={{ marginTop: 18 }}>
          <header className="dash-section-head">
            <h2 style={{ fontSize: 24 }}>Журнал действий</h2>
          </header>

          <div className="admin-table">
            <div className="admin-row head" style={{ gridTemplateColumns: "40px 1.4fr 1fr 1fr 1fr 1fr" }}>
              <span>#</span>
              <span>Действие · ресурс</span>
              <span>Статус</span>
              <span>Когда</span>
              <span>Админ</span>
              <span>IP</span>
            </div>
            {auditLoading ? (
              <div className="muted" style={{ padding: 20 }}>Загружаем…</div>
            ) : auditLogs.length === 0 ? (
              <div className="muted" style={{ padding: 20 }}>Журнал пуст</div>
            ) : (
              auditLogs.map((log, i) => (
                <div className="admin-row" key={log.id} style={{ gridTemplateColumns: "40px 1.4fr 1fr 1fr 1fr 1fr" }}>
                  <span className="num">{String(i + 1).padStart(2, "0")}</span>
                  <div>
                    <strong>{log.action ?? "—"}</strong>
                    <span className="sub">
                      {log.resource ?? "—"}
                      {log.resource_id ? ` · ${log.resource_id}` : ""}
                    </span>
                  </div>
                  <span>
                    <span className={`status ${log.status ?? "active"}`}>{log.status ?? "—"}</span>
                  </span>
                  <span className="date-mono">{formatDate(log.created_at)}</span>
                  <span className="date-mono">{log.admin_id ?? "—"}</span>
                  <span className="date-mono">{log.ip_address ?? "—"}</span>
                </div>
              ))
            )}
          </div>
        </section>
      )}
    </>
  );
}

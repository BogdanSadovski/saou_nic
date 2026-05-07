import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { adminApi } from "@/shared/api";
import type {
  AdminAuditLog,
  AdminDashboardStats,
  AdminSubscription,
  AdminUser,
} from "@/shared/api/admin";
import { useTranslation } from "@/shared/i18n";
import {
  EmptyState,
  FloatingInput,
  GlassButton,
  GlassCard,
  Skeleton,
  useToast,
} from "@/shared/ui";

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
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString();
};

const formatCurrency = (value: number) =>
  value > 0 ? `${value.toLocaleString("ru-RU", { maximumFractionDigits: 0 })} ₽` : "—";

const userDisplayName = (u: AdminUser) => {
  const fn = `${u.first_name ?? ""} ${u.last_name ?? ""}`.trim();
  return fn || u.username || u.email || u.id;
};

export default function AdminPage() {
  const t = useTranslation();
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

  // Apply local search/filter on top of the server filter so the user
  // sees instant feedback while typing without an extra round-trip.
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
    <section className="page admin-page">
      <div className="section-header">
        <h1>{t.adminAnalytics}</h1>
        <GlassButton
          onClick={() => {
            void loadStats();
            void loadUsers();
            if (tab === "subscriptions") void loadSubscriptions();
            if (tab === "audit") void loadAuditLogs();
          }}
          type="button"
          variant="ghost"
        >
          Обновить
        </GlassButton>
      </div>

      {/* KPI strip */}
      <div className="dashboard-grid report-metrics-grid">
        <GlassCard className="stat-card">
          <p className="muted">Всего пользователей</p>
          {statsLoading ? <Skeleton width={80} height={32} /> : <h2>{stats?.total_users ?? "—"}</h2>}
        </GlassCard>
        <GlassCard className="stat-card">
          <p className="muted">Активные</p>
          {statsLoading ? <Skeleton width={80} height={32} /> : <h2>{stats?.active_users ?? "—"}</h2>}
        </GlassCard>
        <GlassCard className="stat-card">
          <p className="muted">Новые за сегодня</p>
          {statsLoading ? <Skeleton width={80} height={32} /> : <h2>{stats?.new_users_today ?? "—"}</h2>}
        </GlassCard>
        <GlassCard className="stat-card">
          <p className="muted">Активные подписки</p>
          {statsLoading ? (
            <Skeleton width={80} height={32} />
          ) : (
            <h2>{stats?.active_subscriptions ?? "—"}</h2>
          )}
        </GlassCard>
        <GlassCard className="stat-card">
          <p className="muted">Доход за месяц</p>
          {statsLoading ? (
            <Skeleton width={120} height={32} />
          ) : (
            <h2>{formatCurrency(stats?.revenue_this_month ?? 0)}</h2>
          )}
        </GlassCard>
      </div>

      {statsError ? (
        <EmptyState
          icon="🔒"
          title="Нет доступа к админ-метрикам"
          hint={statsError}
          action={
            <GlassButton onClick={() => void loadStats()} type="button" variant="primary">
              Повторить
            </GlassButton>
          }
        />
      ) : (
        <GlassCard>
          <h3>Распределение и тарифы</h3>
          {statsLoading ? <Skeleton variant="card" height={220} /> : stats ? <AdminCharts stats={stats} /> : null}
        </GlassCard>
      )}

      {/* Tabs */}
      <div className="auth-mode-switch admin-tabs">
        <button
          type="button"
          className={tab === "users" ? "mode-active" : ""}
          onClick={() => setTab("users")}
        >
          Пользователи
        </button>
        <button
          type="button"
          className={tab === "subscriptions" ? "mode-active" : ""}
          onClick={() => setTab("subscriptions")}
        >
          Подписки
        </button>
        <button
          type="button"
          className={tab === "audit" ? "mode-active" : ""}
          onClick={() => setTab("audit")}
        >
          Журнал действий
        </button>
      </div>

      {tab === "users" && (
        <GlassCard>
          <div className="filters two-col">
            <FloatingInput
              label="Поиск (email/имя/username)"
              onChange={(e) => setQuery(e.target.value)}
              value={query}
            />
            <label className="status-filter">
              <span>Статус</span>
              <select onChange={(e) => setStatusFilter(e.target.value)} value={statusFilter}>
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {usersLoading ? (
            <Skeleton count={5} />
          ) : filteredUsers.length === 0 ? (
            <EmptyState
              icon="🧑‍💻"
              title={usersTotal === 0 ? "Пока нет пользователей" : "Ничего не найдено"}
              hint={
                usersTotal === 0
                  ? "Когда появятся регистрации — они подтянутся в эту таблицу автоматически."
                  : "Попробуйте сбросить фильтры."
              }
              action={
                usersTotal !== 0 ? (
                  <GlassButton
                    onClick={() => {
                      setQuery("");
                      setStatusFilter("");
                    }}
                    type="button"
                    variant="ghost"
                  >
                    Сбросить фильтры
                  </GlassButton>
                ) : null
              }
            />
          ) : (
            <div className="admin-table-wrap">
              <table className="admin-table">
                <thead>
                  <tr>
                    <th>Пользователь</th>
                    <th>Email</th>
                    <th>Роль</th>
                    <th>Статус</th>
                    <th>Создан</th>
                    <th>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredUsers.map((u) => (
                    <tr key={u.id}>
                      <td>
                        <strong>{userDisplayName(u)}</strong>
                      </td>
                      <td className="muted">{u.email ?? "—"}</td>
                      <td>
                        <select
                          disabled={pendingId === u.id}
                          onChange={(e) => void handleRoleChange(u, e.target.value)}
                          value={u.role ?? "user"}
                        >
                          {ROLE_OPTIONS.map((r) => (
                            <option key={r} value={r}>
                              {r}
                            </option>
                          ))}
                        </select>
                      </td>
                      <td>
                        <span className={`report-status report-status-${u.status ?? "active"}`}>
                          {u.status ?? "active"}
                        </span>
                      </td>
                      <td className="muted">{formatDate(u.created_at)}</td>
                      <td>
                        <div className="admin-actions">
                          {u.status === "suspended" || u.status === "banned" ? (
                            <button
                              className="practice-secondary-btn"
                              disabled={pendingId === u.id}
                              onClick={() => void performAction(u, "activate")}
                              type="button"
                            >
                              Активировать
                            </button>
                          ) : (
                            <button
                              className="practice-secondary-btn"
                              disabled={pendingId === u.id}
                              onClick={() => void performAction(u, "suspend")}
                              type="button"
                            >
                              Приостановить
                            </button>
                          )}
                          <button
                            className="practice-secondary-btn admin-action-danger"
                            disabled={pendingId === u.id}
                            onClick={() => void performAction(u, "ban")}
                            type="button"
                          >
                            Забанить
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </GlassCard>
      )}

      {tab === "subscriptions" && (
        <GlassCard>
          <h3>Подписки</h3>
          {subsLoading ? (
            <Skeleton count={5} />
          ) : subscriptions.length === 0 ? (
            <EmptyState
              icon="💳"
              title="Подписок пока нет"
              hint="Все активные тарифы пользователей появятся здесь."
            />
          ) : (
            <div className="admin-table-wrap">
              <table className="admin-table">
                <thead>
                  <tr>
                    <th>Пользователь</th>
                    <th>Тариф</th>
                    <th>Статус</th>
                    <th>Сумма</th>
                    <th>Старт</th>
                    <th>Истекает</th>
                  </tr>
                </thead>
                <tbody>
                  {subscriptions.map((s) => (
                    <tr key={s.id}>
                      <td className="muted">{s.user_id}</td>
                      <td>
                        <strong>{s.tier}</strong>
                      </td>
                      <td>
                        <span className={`report-status report-status-${s.status}`}>{s.status}</span>
                      </td>
                      <td>
                        {s.amount && s.amount > 0
                          ? `${s.amount.toLocaleString("ru-RU")} ${s.currency ?? ""}`
                          : "—"}
                      </td>
                      <td className="muted">{formatDate(s.started_at)}</td>
                      <td className="muted">{formatDate(s.expires_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </GlassCard>
      )}

      {tab === "audit" && (
        <GlassCard>
          <h3>Журнал действий</h3>
          {auditLoading ? (
            <Skeleton count={6} />
          ) : auditLogs.length === 0 ? (
            <EmptyState
              icon="📜"
              title="Журнал пуст"
              hint="Любые админ-действия будут логироваться здесь автоматически."
            />
          ) : (
            <div className="admin-table-wrap">
              <table className="admin-table">
                <thead>
                  <tr>
                    <th>Когда</th>
                    <th>Админ</th>
                    <th>Действие</th>
                    <th>Объект</th>
                    <th>Статус</th>
                    <th>IP</th>
                  </tr>
                </thead>
                <tbody>
                  {auditLogs.map((log) => (
                    <tr key={log.id}>
                      <td className="muted">{formatDate(log.created_at)}</td>
                      <td className="muted">{log.admin_id ?? "—"}</td>
                      <td>
                        <strong>{log.action ?? "—"}</strong>
                      </td>
                      <td className="muted">
                        {log.resource ?? "—"}
                        {log.resource_id ? ` · ${log.resource_id}` : ""}
                      </td>
                      <td>
                        <span className={`report-status report-status-${log.status ?? "active"}`}>
                          {log.status ?? "—"}
                        </span>
                      </td>
                      <td className="muted">{log.ip_address ?? "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </GlassCard>
      )}
    </section>
  );
}

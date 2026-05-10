import { apiClient } from "./client";

export type AdminDashboardStats = {
  total_users: number;
  active_users: number;
  new_users_today: number;
  total_subscriptions: number;
  active_subscriptions: number;
  revenue_this_month: number;
  role_distribution: Record<string, number>;
  subscription_tiers: Record<string, number>;
  recent_audit_logs: AdminAuditLog[];
};

export type AdminUser = {
  id: string;
  email?: string;
  username?: string;
  first_name?: string;
  last_name?: string;
  role?: string;
  status?: string;
  score?: number;
  created_at?: string;
  last_login_at?: string;
};

export type AdminUserListResponse = {
  items: AdminUser[];
  total: number;
  page: number;
  page_size: number;
};

export type AdminAuditLog = {
  id: string;
  admin_id?: string;
  action?: string;
  resource?: string;
  resource_id?: string;
  status?: string;
  ip_address?: string;
  created_at?: string;
};

export type AdminSubscription = {
  id: string;
  user_id: string;
  tier: string;
  status: string;
  /** Match admin-service domain.Subscription field names. */
  start_date?: string;
  end_date?: string;
  amount?: number;
  currency?: string;
};

export type AdminSubscriptionListResponse = {
  items: AdminSubscription[];
  total: number;
};

type ListParams = {
  page?: number;
  pageSize?: number;
  search?: string;
  status?: string;
  role?: string;
};

const buildQuery = (params: ListParams): string => {
  const sp = new URLSearchParams();
  if (params.page) sp.set("page", String(params.page));
  if (params.pageSize) sp.set("page_size", String(params.pageSize));
  if (params.search) sp.set("search", params.search);
  if (params.status) sp.set("status", params.status);
  if (params.role) sp.set("role", params.role);
  const q = sp.toString();
  return q ? `?${q}` : "";
};

/**
 * Admin-only client. All endpoints require an admin-role JWT — the
 * gateway forwards 401/403 directly so callers can react. Responses
 * are normalised: empty / missing arrays come back as [], and total
 * counts default to 0 so charts and counters never see undefined.
 */
export const adminApi = {
  async getDashboardStats(): Promise<AdminDashboardStats> {
    const { data } = await apiClient.get<Partial<AdminDashboardStats>>("/admin/dashboard/stats");
    return {
      total_users: data.total_users ?? 0,
      active_users: data.active_users ?? 0,
      new_users_today: data.new_users_today ?? 0,
      total_subscriptions: data.total_subscriptions ?? 0,
      active_subscriptions: data.active_subscriptions ?? 0,
      revenue_this_month: data.revenue_this_month ?? 0,
      role_distribution: data.role_distribution ?? {},
      subscription_tiers: data.subscription_tiers ?? {},
      recent_audit_logs: data.recent_audit_logs ?? [],
    };
  },

  async listUsers(params: ListParams = {}): Promise<AdminUserListResponse> {
    const { data } = await apiClient.get<Partial<AdminUserListResponse>>(
      `/admin/users${buildQuery(params)}`,
    );
    return {
      items: data.items ?? [],
      total: data.total ?? 0,
      page: data.page ?? 1,
      page_size: data.page_size ?? 20,
    };
  },

  async suspendUser(userId: string, reason?: string): Promise<void> {
    await apiClient.post(`/admin/users/${userId}/suspend`, { reason });
  },

  async activateUser(userId: string): Promise<void> {
    await apiClient.post(`/admin/users/${userId}/activate`);
  },

  async banUser(userId: string, reason?: string): Promise<void> {
    await apiClient.post(`/admin/users/${userId}/ban`, { reason });
  },

  async changeUserRole(userId: string, role: string): Promise<void> {
    await apiClient.post(`/admin/users/${userId}/role`, { role });
  },

  async listSubscriptions(params: ListParams = {}): Promise<AdminSubscriptionListResponse> {
    const { data } = await apiClient.get<Partial<AdminSubscriptionListResponse>>(
      `/admin/subscriptions${buildQuery(params)}`,
    );
    return {
      items: data.items ?? [],
      total: data.total ?? 0,
    };
  },

  async listAuditLogs(
    params: ListParams = {},
  ): Promise<{ items: AdminAuditLog[]; total: number }> {
    const { data } = await apiClient.get<{ items?: AdminAuditLog[]; total?: number }>(
      `/admin/audit-logs${buildQuery(params)}`,
    );
    return {
      items: data.items ?? [],
      total: data.total ?? 0,
    };
  },
};

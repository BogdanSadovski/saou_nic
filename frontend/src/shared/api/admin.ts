import { apiClient } from "./client";

type DashboardStats = {
  total_users?: number;
  active_users?: number;
  average_score?: number;
};

type AdminUser = {
  id: string;
  first_name?: string;
  last_name?: string;
  username?: string;
  status?: string;
  score?: number;
};

type PaginationResponse = {
  items?: AdminUser[];
};

export const adminApi = {
  async getDashboardStats(): Promise<DashboardStats> {
    const { data } = await apiClient.get<DashboardStats>("/admin/dashboard/stats");
    return data;
  },

  async listUsers(): Promise<AdminUser[]> {
    const { data } = await apiClient.get<PaginationResponse>("/admin/users");
    return data.items ?? [];
  },
};

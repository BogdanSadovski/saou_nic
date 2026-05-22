import { apiClient } from "./client";

/**
 * Subscription quota — серверный учёт «сколько действий пользователь
 * выполнил в этом календарном месяце». Бэк (interview-service) хранит
 * счётчики в Redis (ключ `quota:<resource>:<user_id>:<YYYY-MM>`),
 * сбрасывает их каждый месяц и enforce'ит при создании интервью /
 * импорте резюме / GitHub-импорте.
 *
 * Фронт показывает «осталось N интервью» в Profile/Billing и
 * предлагает апгрейд при 0 remaining.
 */

export type QuotaStatus = {
  resource: "interview" | "resume" | "github_import";
  tier: string;
  /** -1 означает безлимит. */
  limit: number;
  used: number;
  /** -1 означает безлимит. */
  remaining: number;
  allowed: boolean;
};

export type QuotaSnapshot = {
  tier: string;
  quota: Record<"interview" | "resume" | "github_import", QuotaStatus>;
};

type WrappedResponse<T> = {
  data?: T;
  success?: boolean;
  error?: string;
};

export const quotaApi = {
  async getMine(): Promise<QuotaSnapshot | null> {
    try {
      const { data } = await apiClient.get<WrappedResponse<QuotaSnapshot>>("/quota/me");
      return data?.data ?? null;
    } catch (error) {
      const status = (error as { response?: { status?: number } })?.response?.status;
      if (status === 401) return null;
      // На legacy-роут пробрасываем для совместимости с api-gateway-rewrite.
      try {
        const { data } = await apiClient.get<WrappedResponse<QuotaSnapshot>>("/v1/quota/me");
        return data?.data ?? null;
      } catch {
        return null;
      }
    }
  },
};

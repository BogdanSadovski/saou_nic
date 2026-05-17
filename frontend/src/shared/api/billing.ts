import { apiClient } from "./client";

/**
 * User-facing billing endpoints. Backed by admin-service /billing/me
 * (subscriptions table lives in user_service DB; admin reads/writes
 * the same rows for its dashboard counts).
 *
 * The "checkout" page is a fake provider on the frontend — it POSTs
 * here on success to record the chosen tier so admin stats reflect
 * real subscriptions and the user's tier persists across devices.
 */

export type BackendSubscription = {
  id: string;
  user_id: string;
  tier: "free" | "basic" | "starter" | "pro" | "team" | "enterprise";
  status: string; // "active" | "cancelled" | "expired" ...
  /** Field names match admin-service domain.Subscription. */
  start_date?: string;
  end_date?: string;
  auto_renew?: boolean;
  amount?: number;
  currency?: string;
  created_at?: string;
  updated_at?: string;
};

export const billingApi = {
  /** 200 with subscription payload, or 404 when on the free plan. */
  async getMine(): Promise<BackendSubscription | null> {
    try {
      const { data } = await apiClient.get<BackendSubscription>(
        "/billing/me/subscription",
      );
      return data;
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      if (status === 404) return null;
      throw e;
    }
  },

  /** Records a successful "payment" — the backend creates an active
   *  subscription tied to the current user. */
  async create(input: { tier: string; cardLast4?: string }): Promise<BackendSubscription> {
    const { data } = await apiClient.post<BackendSubscription>(
      "/billing/me/subscription",
      { tier: input.tier, card_last4: input.cardLast4 ?? "" },
    );
    return data;
  },

  /** Downgrades the current user back to free. */
  async cancel(): Promise<void> {
    await apiClient.delete("/billing/me/subscription");
  },
};

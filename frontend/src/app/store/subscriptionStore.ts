import { create } from "zustand";

import { billingApi } from "@/shared/api/billing";
import type { BackendSubscription } from "@/shared/api/billing";

/**
 * Subscription state — server-backed (admin-service /billing/me) with
 * a localStorage cache so the UI renders instantly while the network
 * request resolves.
 *
 * Source of truth: /billing/me/subscription on admin-service.
 *   • hydrate()       fetches the current user's subscription on app boot
 *   • applyPayment()  POST + write-through cache (called by Checkout)
 *   • cancel()        DELETE + write-through cache
 *
 * Everything still falls back to local cache when the network is down
 * so the user never sees a broken billing UI.
 */

const KEY = "realsync_subscription";

export type Tier = "free" | "starter" | "pro" | "team";

export type PaidIntent = {
  tier: Exclude<Tier, "free">;
  /** Amount in USD whole units. Backend stats price subscriptions in
   *  USD too — see admin-service tierMonthlyPriceUSD. */
  amount: number;
  currency: "USD";
  cardLast4: string;
  paidAt: string;
  expiresAt: string;
};

export type Subscription = {
  tier: Tier;
  intent: PaidIntent | null;
};

const DEFAULT: Subscription = {
  tier: "free",
  intent: null,
};

const load = (): Subscription => {
  if (typeof window === "undefined") return DEFAULT;
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return DEFAULT;
    const parsed = JSON.parse(raw) as Subscription;
    if (!parsed || typeof parsed.tier !== "string") return DEFAULT;
    // Auto-downgrade expired subscriptions on hydration.
    if (parsed.intent && new Date(parsed.intent.expiresAt) < new Date()) {
      return { tier: "free", intent: null };
    }
    return parsed;
  } catch {
    return DEFAULT;
  }
};

const persist = (s: Subscription) => {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(KEY, JSON.stringify(s));
  }
};

type State = Subscription & {
  /** Pull current subscription from the backend. No-op if offline. */
  hydrate: () => Promise<void>;
  /** Mark a successful checkout — POSTs to backend, then caches. */
  applyPayment: (intent: PaidIntent) => Promise<void>;
  /** Cancel current paid plan (revert to free). */
  cancel: () => Promise<void>;
  /** Re-read localStorage (used after Checkout pushes to it). */
  refresh: () => void;
};

const fromBackend = (sub: BackendSubscription | null): Subscription => {
  if (!sub) return DEFAULT;
  // Map the wider backend tier set down to the UI's tier vocabulary.
  // Anything not matching a paid tier falls back to free.
  const tier: Tier = (() => {
    if (sub.tier === "starter" || sub.tier === "pro" || sub.tier === "team") return sub.tier;
    if (sub.tier === "basic") return "starter";
    if (sub.tier === "enterprise") return "team";
    return "free";
  })();
  if (tier === "free") return DEFAULT;

  // Pricing isn't stored on the backend Subscription record yet — derive
  // it from the catalog so the Profile renders matching numbers.
  const knownPrice = TIER_CATALOG.find((t) => t.tier === tier)?.price ?? 0;

  return {
    tier,
    intent: {
      tier,
      amount: sub.amount ?? knownPrice,
      currency: "USD",
      cardLast4: "••••",
      paidAt: sub.start_date ?? sub.created_at ?? new Date().toISOString(),
      expiresAt: sub.end_date ?? new Date(Date.now() + 30 * 86_400_000).toISOString(),
    },
  };
};

export const useSubscriptionStore = create<State>((set, get) => ({
  ...load(),
  hydrate: async () => {
    try {
      const sub = await billingApi.getMine();
      const next = fromBackend(sub);
      persist(next);
      set(next);
    } catch {
      // network down — keep the cached value
    }
  },
  applyPayment: async (intent) => {
    // Optimistic write to local cache so the UI updates instantly.
    const optimistic: Subscription = { tier: intent.tier, intent };
    persist(optimistic);
    set(optimistic);
    try {
      await billingApi.create({ tier: intent.tier, cardLast4: intent.cardLast4 });
    } catch {
      // Roll back to whatever the backend says next time we hydrate.
      // Don't strip the optimistic state right now — the user just saw
      // a successful "checkout" page, breaking the UI on a 5xx is worse
      // than a brief mismatch.
    }
  },
  cancel: async () => {
    persist(DEFAULT);
    set(DEFAULT);
    try {
      await billingApi.cancel();
    } catch {
      // ignored — local downgrade is still useful
    }
    void get();
  },
  refresh: () => set(load()),
}));

export const TIER_CATALOG: Array<{
  tier: Exclude<Tier, "free">;
  title: string;
  price: number;
  perks: string[];
  highlight?: boolean;
}> = [
  {
    tier: "starter",
    title: "Starter",
    price: 9, // USD/mo — keep in sync with admin-service tierMonthlyPriceUSD
    perks: [
      "До 5 интервью в месяц",
      "Базовые AI-вопросы",
      "Текстовые отчёты",
    ],
  },
  {
    tier: "pro",
    title: "Pro",
    price: 19,
    highlight: true,
    perks: [
      "До 30 интервью в месяц",
      "Адаптивная сложность и live-coding",
      "Графическая аналитика и тренды",
      "Резюме-инсайты от AI",
    ],
  },
  {
    tier: "team",
    title: "Team",
    price: 49,
    perks: [
      "Без лимита интервью",
      "Все режимы (theory + practice)",
      "Командные отчёты и экспорт",
      "Приоритетная поддержка",
    ],
  },
];

export const getTierTitle = (tier: Tier): string => {
  if (tier === "free") return "Free";
  return TIER_CATALOG.find((t) => t.tier === tier)?.title ?? tier;
};

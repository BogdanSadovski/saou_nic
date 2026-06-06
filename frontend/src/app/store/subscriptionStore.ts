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

// Ровно три тарифа на всей платформе — совпадают с PLAN_CATALOGUE
// в shared/api/billing.ts и с бэкендом (admin-service).
export type Tier = "trial" | "pro" | "platinum";

export type PaidIntent = {
  tier: Exclude<Tier, "trial">;
  /** Amount in BYN whole units. UI displays as "Br" — see
   *  shared/lib/currency formatBYN. Backend historical amounts may
   *  still be denominated in USD; the type is tolerant for that. */
  amount: number;
  currency: "BYN" | "USD";
  cardLast4: string;
  paidAt: string;
  expiresAt: string;
};

export type Subscription = {
  tier: Tier;
  intent: PaidIntent | null;
};

const DEFAULT: Subscription = {
  tier: "trial",
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
      return { tier: "trial", intent: null };
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
  /** Cancel current paid plan (revert to trial). */
  cancel: () => Promise<void>;
  /** Re-read localStorage (used after Checkout pushes to it). */
  refresh: () => void;
};

const fromBackend = (sub: BackendSubscription | null): Subscription => {
  if (!sub) return DEFAULT;
  // Map any tier (incl. legacy free/basic/starter/team/enterprise) down to
  // the canonical three-tier vocabulary.
  const tier: Tier = (() => {
    if (sub.tier === "pro" || sub.tier === "basic" || sub.tier === "starter") return "pro";
    if (sub.tier === "platinum" || sub.tier === "team" || sub.tier === "enterprise") return "platinum";
    return "trial";
  })();
  if (tier === "trial") return DEFAULT;

  // Pricing isn't stored on the backend Subscription record yet — derive
  // it from the catalog so the Profile renders matching numbers.
  const knownPrice = TIER_CATALOG.find((t) => t.tier === tier)?.price ?? 0;

  return {
    tier,
    intent: {
      tier,
      amount: sub.amount ?? knownPrice,
      currency: "BYN",
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
  tier: Exclude<Tier, "trial">;
  title: string;
  price: number;
  perks: string[];
  highlight?: boolean;
}> = [
  {
    tier: "pro",
    title: "Pro",
    price: 65, // BYN/мес — отображается через formatBYN ("Br")
    highlight: true,
    perks: [
      "30 интервью в месяц",
      "Безлимит резюме-анализа",
      "AI-вердикты + PDF-отчёты",
      "Полная GitHub-аналитика",
    ],
  },
  {
    tier: "platinum",
    title: "Platinum",
    price: 159,
    perks: [
      "Безлимит интервью и резюме",
      "Приоритетная очередь LLM",
      "Кастомные сценарии интервью",
      "Personal manager",
    ],
  },
];

export const getTierTitle = (tier: Tier): string => {
  if (tier === "trial") return "Trial";
  return TIER_CATALOG.find((t) => t.tier === tier)?.title ?? tier;
};

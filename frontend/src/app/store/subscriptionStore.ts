import { create } from "zustand";

/**
 * Local subscription state.
 *
 * The platform doesn't yet wire a real billing provider, so we keep
 * the active tier in localStorage and treat the dedicated checkout
 * route as a fake external gateway. When real billing lands, this
 * store should be replaced by /billing/me payload from admin-service.
 */

const KEY = "realsync_subscription";

export type Tier = "free" | "starter" | "pro" | "team";

export type PaidIntent = {
  tier: Exclude<Tier, "free">;
  amount: number; // in RUB
  currency: "RUB";
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
  /** Mark a successful checkout. */
  applyPayment: (intent: PaidIntent) => void;
  /** Cancel current paid plan (revert to free). */
  cancel: () => void;
  /** Reload from storage (used after returning from checkout). */
  refresh: () => void;
};

export const useSubscriptionStore = create<State>((set) => ({
  ...load(),
  applyPayment: (intent) => {
    const next: Subscription = { tier: intent.tier, intent };
    persist(next);
    set(next);
  },
  cancel: () => {
    persist(DEFAULT);
    set(DEFAULT);
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
    price: 490,
    perks: [
      "До 5 интервью в месяц",
      "Базовые AI-вопросы",
      "Текстовые отчёты",
    ],
  },
  {
    tier: "pro",
    title: "Pro",
    price: 1490,
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
    price: 3990,
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

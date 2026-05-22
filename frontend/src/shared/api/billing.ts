import { apiClient } from "./client";

/**
 * User-facing billing endpoints. Backed by admin-service /billing/me
 * (subscriptions table lives in user_service DB; admin reads/writes
 * the same rows for its dashboard counts).
 *
 * Note: backend exposes a minimal CRUD surface (get/create/cancel).
 * The richer Billing-page UX (plans grid, checkout intent, transaction
 * history) is layered on top here:
 *   - getPlans()                 — static catalogue defined client-side
 *   - getSubscription()          — alias for /billing/me/subscription
 *   - listTransactions()         — synthesized from current subscription
 *                                  until the backend adds a real ledger
 *   - createCheckoutIntent()     — UX-side stub (no real PSP)
 *   - confirmCheckoutIntent()    — calls real /billing/me/subscription
 *                                  POST to persist the chosen tier
 *   - cancelSubscription()       — alias for DELETE
 */

export type BackendSubscription = {
  id: string;
  user_id: string;
  tier: "free" | "basic" | "starter" | "pro" | "team" | "enterprise" | "trial" | "platinum";
  status: string; // "active" | "cancelled" | "expired" ...
  start_date?: string;
  end_date?: string;
  auto_renew?: boolean;
  amount?: number;
  currency?: string;
  created_at?: string;
  updated_at?: string;
};

export type BillingPlan = {
  /** Уникальный id (используется в JSX-ключах). Дублируется с tier. */
  id: string;
  tier: "trial" | "pro" | "platinum";
  name: string;
  description: string;
  /** Цена в BYN за период billing_cycle. */
  price: number;
  /** Дубликат price для нового UI. */
  price_byn: number;
  /** Старое название периода — "month" / "year". */
  billing_cycle: "month" | "year";
  period: "month" | "year";
  features: string[];
  limits: {
    interviews_per_month: number | "unlimited";
    resumes_per_month: number | "unlimited";
    github_imports_per_month: number | "unlimited";
  };
  recommended?: boolean;
};

export type SubscriptionSnapshot = {
  tier: string;
  status: string;
  start_date?: string;
  end_date?: string;
  auto_renew?: boolean;
};

export type BillingTransaction = {
  id: string;
  amount_cents: number;
  currency: string;
  status: "succeeded" | "pending" | "failed" | "refunded";
  tier: string;
  created_at: string;
  description?: string;
};

export type CheckoutIntent = {
  id: string;
  tier: "pro" | "platinum";
  amount_cents: number;
  currency: string;
  /** Используем терминологию Stripe для совместимости с UI:
   *  requires_confirmation → ждёт подтверждения; succeeded → оплачено. */
  status: "requires_confirmation" | "succeeded" | "failed" | "canceled";
  payment_method_id?: string;
  expires_at?: string;
};

export type CheckoutConfirmation = {
  intent: CheckoutIntent;
  subscription: SubscriptionSnapshot;
  transaction: BillingTransaction;
};

// Каталог планов хранится на фронте — это «прейскурант», который
// показываем на странице Billing. Реальные лимиты enforce'ятся
// бэкендом (см. middleware подписки).
const PLAN_CATALOGUE: BillingPlan[] = [
  {
    id: "trial",
    tier: "trial",
    name: "Trial",
    description: "Стартовый тариф без оплаты — попробуйте платформу.",
    price: 0,
    price_byn: 0,
    billing_cycle: "month",
    period: "month",
    features: [
      "5 интервью в месяц",
      "3 резюме-анализа",
      "AI-вердикты на каждый ответ",
      "Базовый GitHub-импорт",
    ],
    limits: { interviews_per_month: 5, resumes_per_month: 3, github_imports_per_month: 1 },
  },
  {
    id: "pro",
    tier: "pro",
    name: "Pro",
    description: "Для активной подготовки: безлимит резюме, расширенные интервью.",
    price: 65,
    price_byn: 65,
    billing_cycle: "month",
    period: "month",
    features: [
      "30 интервью в месяц",
      "Безлимит резюме-анализа",
      "AI-вердикты + PDF-отчёты",
      "Soft-skills модель",
      "Полная GitHub-аналитика",
    ],
    limits: { interviews_per_month: 30, resumes_per_month: "unlimited", github_imports_per_month: 10 },
    recommended: true,
  },
  {
    id: "platinum",
    tier: "platinum",
    name: "Platinum",
    description: "Максимум: безлимиты и приоритетный LLM-маршрут.",
    price: 159,
    price_byn: 159,
    billing_cycle: "month",
    period: "month",
    features: [
      "Безлимит интервью",
      "Безлимит резюме и GitHub",
      "Приоритетная очередь LLM",
      "Кастомные сценарии интервью",
      "Personal manager",
    ],
    limits: {
      interviews_per_month: "unlimited",
      resumes_per_month: "unlimited",
      github_imports_per_month: "unlimited",
    },
  },
];

const toSnapshot = (sub: BackendSubscription | null): SubscriptionSnapshot | null => {
  if (!sub) return null;
  return {
    tier: sub.tier,
    status: sub.status,
    start_date: sub.start_date,
    end_date: sub.end_date,
    auto_renew: sub.auto_renew,
  };
};

const tierToCents = (tier: string): number => {
  const plan = PLAN_CATALOGUE.find((p) => p.tier === tier);
  return plan ? Math.round(plan.price_byn * 100) : 0;
};

let mockTransactions: BillingTransaction[] = [];
let mockIntent: CheckoutIntent | null = null;

export const billingApi = {
  /** Сохраняемая совместимость со старым кодом. */
  async getMine(): Promise<BackendSubscription | null> {
    try {
      const { data } = await apiClient.get<BackendSubscription>("/billing/me/subscription");
      return data;
    } catch (e) {
      const status = (e as { response?: { status?: number } })?.response?.status;
      if (status === 404) return null;
      throw e;
    }
  },

  async create(input: { tier: string; cardLast4?: string }): Promise<BackendSubscription> {
    const { data } = await apiClient.post<BackendSubscription>("/billing/me/subscription", {
      tier: input.tier,
      card_last4: input.cardLast4 ?? "",
    });
    return data;
  },

  async cancel(): Promise<void> {
    await apiClient.delete("/billing/me/subscription");
  },

  // ───── Расширенный API для страницы Billing ─────

  async getPlans(): Promise<BillingPlan[]> {
    return PLAN_CATALOGUE;
  },

  async getSubscription(): Promise<SubscriptionSnapshot | null> {
    return toSnapshot(await billingApi.getMine());
  },

  async listTransactions(): Promise<BillingTransaction[]> {
    // Бэкенд ещё не отдаёт ledger — синтезируем последнюю транзакцию из
    // текущей подписки, чтобы блок «История» не был пустым на платных
    // планах. Дополнительно показываем in-memory транзакции, сделанные
    // в этой сессии через confirmCheckoutIntent.
    const sub = await billingApi.getMine();
    const derived: BillingTransaction[] = [];
    if (sub && sub.tier !== "trial" && sub.tier !== "free") {
      derived.push({
        id: sub.id,
        amount_cents: Math.round((sub.amount ?? tierToCents(sub.tier) / 100) * 100),
        currency: sub.currency || "BYN",
        status: "succeeded",
        tier: sub.tier,
        created_at: sub.start_date || sub.created_at || new Date().toISOString(),
        description: `Подписка ${sub.tier.toUpperCase()}`,
      });
    }
    return [...mockTransactions, ...derived];
  },

  async createCheckoutIntent(input: {
    tier: "pro" | "platinum";
    payment_method_id?: string;
  }): Promise<CheckoutIntent> {
    const intent: CheckoutIntent = {
      id: `intent_${Date.now().toString(36)}`,
      tier: input.tier,
      amount_cents: tierToCents(input.tier),
      currency: "BYN",
      status: "requires_confirmation",
      payment_method_id: input.payment_method_id,
      expires_at: new Date(Date.now() + 30 * 60_000).toISOString(),
    };
    mockIntent = intent;
    return intent;
  },

  async confirmCheckoutIntent(intentId: string): Promise<CheckoutConfirmation> {
    if (!mockIntent || mockIntent.id !== intentId) {
      throw new Error("Intent expired or not found");
    }
    const tier = mockIntent.tier;
    const sub = await billingApi.create({ tier, cardLast4: "4242" });
    const confirmedIntent: CheckoutIntent = { ...mockIntent, status: "succeeded" };
    const transaction: BillingTransaction = {
      id: `tx_${Date.now().toString(36)}`,
      amount_cents: mockIntent.amount_cents,
      currency: "BYN",
      status: "succeeded",
      tier,
      created_at: new Date().toISOString(),
      description: `Активация плана ${tier.toUpperCase()}`,
    };
    mockTransactions = [transaction, ...mockTransactions];
    mockIntent = null;
    return {
      intent: confirmedIntent,
      subscription: toSnapshot(sub)!,
      transaction,
    };
  },

  async cancelSubscription(): Promise<void> {
    await billingApi.cancel();
  },
};

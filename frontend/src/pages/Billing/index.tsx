import { useEffect, useState } from "react";

import { billingApi, type BillingPlan, type BillingTransaction, type CheckoutIntent, type SubscriptionSnapshot } from "@/shared/api/billing";
import { quotaApi, type QuotaSnapshot } from "@/shared/api/quota";
import { formatBYN } from "@/shared/lib/currency";
import { BynSign } from "@/shared/ui/BynSign";
import { RsIcon as Icon } from "@/shared/ui/realsync";
import { useToast } from "@/shared/ui";

const formatMoney = (amountCents: number) => {
  // Backend stores cents; display as BYN ("Br").
  const whole = Math.round((amountCents / 100) * 100) / 100;
  return formatBYN(whole);
};

const formatDate = (value?: string) => {
  if (!value) return "—";
  return new Date(value).toLocaleString("ru-RU", {
    day: "2-digit", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
};

const formatLimit = (limit: number | "unlimited"): string =>
  limit === "unlimited" || limit < 0 ? "∞" : String(limit);

const statusLabel = (status: string): string => {
  switch (status) {
    case "active": return "Активна";
    case "cancelled":
    case "canceled": return "Отменена";
    case "expired": return "Истекла";
    case "pending": return "Ожидание";
    default: return status;
  }
};

const txStatusLabel = (status: string): string => {
  switch (status) {
    case "succeeded": return "Оплачено";
    case "pending": return "В обработке";
    case "failed": return "Ошибка";
    case "refunded": return "Возврат";
    default: return status;
  }
};

export default function BillingPage() {
  const { pushToast } = useToast();
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [plans, setPlans] = useState<BillingPlan[]>([]);
  const [subscription, setSubscription] = useState<SubscriptionSnapshot | null>(null);
  const [transactions, setTransactions] = useState<BillingTransaction[]>([]);
  const [activeIntent, setActiveIntent] = useState<CheckoutIntent | null>(null);
  const [quota, setQuota] = useState<QuotaSnapshot | null>(null);

  const currentTier = subscription?.tier ?? "trial";

  const loadAll = async () => {
    setLoading(true);
    try {
      const [plansData, subscriptionData, txData, quotaData] = await Promise.all([
        billingApi.getPlans(),
        billingApi.getSubscription(),
        billingApi.listTransactions(),
        quotaApi.getMine(),
      ]);
      setPlans(plansData);
      setSubscription(subscriptionData);
      setTransactions(txData);
      setQuota(quotaData);
    } catch (error) {
      console.error(error);
      pushToast("Не удалось загрузить платежные данные");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadAll();
  }, []);

  const startTestCheckout = async (tier: "pro" | "platinum") => {
    setProcessing(true);
    try {
      const intent = await billingApi.createCheckoutIntent({
        tier,
        payment_method_id: "pm_test_visa_4242",
      });
      setActiveIntent(intent);
      pushToast("Создан тестовый платёжный intent. Подтвердите ниже.");
    } catch (error) {
      console.error(error);
      pushToast("Не удалось создать тестовый платёж");
    } finally {
      setProcessing(false);
    }
  };

  const confirmTestCheckout = async () => {
    if (!activeIntent) return;
    setProcessing(true);
    try {
      const result = await billingApi.confirmCheckoutIntent(activeIntent.id);
      setActiveIntent(result.intent);
      setSubscription(result.subscription);
      setTransactions((prev) => [result.transaction, ...prev]);
      pushToast(`Тариф ${result.subscription.tier.toUpperCase()} активирован`);
    } catch (error) {
      console.error(error);
      pushToast("Не удалось подтвердить тестовую оплату");
    } finally {
      setProcessing(false);
    }
  };

  const cancelSubscription = async () => {
    setProcessing(true);
    try {
      await billingApi.cancelSubscription();
      await loadAll();
      pushToast("Подписка отменена");
    } catch (error) {
      console.error(error);
      pushToast("Не удалось отменить подписку");
    } finally {
      setProcessing(false);
    }
  };

  return (
    <>
      <span className="eyebrow">Биллинг</span>
      <header className="row-between" style={{ alignItems: "end", marginTop: 8, flexWrap: "wrap", gap: 12 }}>
        <h1 className="expr-headline" style={{ fontSize: 64 }}>
          <span className="bold">Подписка</span> <span className="ital">и оплата</span>.
        </h1>
        <span className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em" }}>
          ● тестовый шлюз · списания не происходит
        </span>
      </header>

      {/* Текущая подписка + использование */}
      <section className="card" style={{ marginTop: 28, padding: 24 }}>
        <header className="dash-section-head">
          <h2 style={{ fontSize: 28 }}>Текущая подписка</h2>
          <span className={`tag ${subscription ? "tag--ink" : ""}`} style={{ fontSize: 10 }}>
            {subscription ? statusLabel(subscription.status).toUpperCase() : "TRIAL"}
          </span>
        </header>

        {loading ? (
          <p className="muted" style={{ marginTop: 12, fontSize: 14 }}>Загрузка…</p>
        ) : (
          <div style={{ display: "grid", gap: 18, marginTop: 14 }}>
            <div className="row" style={{ gap: 24, alignItems: "baseline", flexWrap: "wrap" }}>
              <div>
                <span className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em" }}>ТАРИФ</span>
                <div className="expr-headline" style={{ fontSize: 32, marginTop: 4 }}>
                  <span className="bold">{currentTier.toUpperCase()}</span>
                </div>
              </div>
              {subscription ? (
                <>
                  <div>
                    <span className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em" }}>ДЕЙСТВУЕТ ДО</span>
                    <div style={{ fontSize: 14, marginTop: 6 }}>{formatDate(subscription.end_date)}</div>
                  </div>
                  <div>
                    <span className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em" }}>АВТО-ПРОДЛЕНИЕ</span>
                    <div style={{ fontSize: 14, marginTop: 6 }}>{subscription.auto_renew ? "включено" : "выключено"}</div>
                  </div>
                </>
              ) : (
                <div className="muted" style={{ fontSize: 14 }}>Активной подписки пока нет — вы на бесплатном тарифе Trial.</div>
              )}
            </div>

            {/* Использование лимитов */}
            {quota ? (
              <div style={{ display: "grid", gap: 10, paddingTop: 8, borderTop: "1px solid var(--line)" }}>
                <span className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", marginTop: 8 }}>
                  ИСПОЛЬЗОВАНИЕ В ЭТОМ МЕСЯЦЕ
                </span>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: 14 }}>
                  {(["interview", "resume", "github_import"] as const).map((key) => {
                    const q = quota.quota[key];
                    const labels = {
                      interview: "Интервью",
                      resume: "Резюме-анализ",
                      github_import: "GitHub-импорт",
                    } as const;
                    const isUnlimited = q.limit < 0;
                    const pct = isUnlimited ? 100 : Math.min(100, Math.round((q.used / Math.max(1, q.limit)) * 100));
                    return (
                      <div key={key} style={{ padding: "12px 14px", border: "1px solid var(--line)", borderRadius: "var(--r-1)", background: "var(--paper-2)" }}>
                        <div className="row-between" style={{ alignItems: "baseline" }}>
                          <span style={{ fontSize: 13, fontWeight: 500 }}>{labels[key]}</span>
                          <span className="mono" style={{ fontSize: 11, color: q.allowed ? "var(--muted)" : "var(--signal)" }}>
                            {q.used} / {formatLimit(q.limit)}
                          </span>
                        </div>
                        <div style={{ marginTop: 8, height: 4, background: "var(--line)", borderRadius: 2, overflow: "hidden" }}>
                          <div style={{
                            width: `${pct}%`,
                            height: "100%",
                            background: q.allowed ? "var(--accent)" : "var(--signal)",
                            transition: "width 220ms ease",
                          }}/>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}

            {subscription ? (
              <div className="row" style={{ gap: 10, marginTop: 4 }}>
                <button
                  className="btn btn--ghost btn--sm"
                  type="button"
                  onClick={cancelSubscription}
                  disabled={processing}
                >
                  Отменить подписку
                </button>
              </div>
            ) : null}
          </div>
        )}
      </section>

      {/* Активный intent */}
      {activeIntent ? (
        <section className="card card--hover" style={{ marginTop: 18, padding: 20, borderColor: activeIntent.status === "succeeded" ? "var(--accent)" : "var(--line)" }}>
          <div className="row-between" style={{ alignItems: "center", flexWrap: "wrap", gap: 12 }}>
            <div>
              <span className="eyebrow">Платёжный intent</span>
              <div className="mono" style={{ fontSize: 13, marginTop: 6 }}>{activeIntent.id}</div>
              <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
                Сумма {formatMoney(activeIntent.amount_cents)} · статус «{activeIntent.status}» · до {formatDate(activeIntent.expires_at)}
              </div>
            </div>
            <button
              className="btn btn--primary"
              type="button"
              onClick={confirmTestCheckout}
              disabled={processing || activeIntent.status !== "requires_confirmation"}
            >
              {activeIntent.status === "succeeded" ? "Оплачено" : "Подтвердить оплату"}
            </button>
          </div>
        </section>
      ) : null}

      {/* Тарифы */}
      <section style={{ marginTop: 32 }}>
        <header className="dash-section-head">
          <h2 style={{ fontSize: 28 }}>Тарифы</h2>
          <span className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.06em" }}>
            оплата раз в месяц · BYN
          </span>
        </header>

        <div style={{
          display: "grid",
          gap: 16,
          marginTop: 14,
          gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))",
        }}>
          {plans.map((plan) => {
            const isCurrent = currentTier === plan.tier;
            const isFree = plan.tier === "trial";
            const isHighlight = plan.recommended;
            return (
              <article
                key={plan.id}
                className="card"
                style={{
                  padding: 22,
                  border: isHighlight ? "1px solid var(--ink)" : "1px solid var(--line)",
                  background: isHighlight ? "var(--paper-2)" : "var(--paper)",
                  display: "grid",
                  gap: 12,
                  position: "relative",
                }}
              >
                {isHighlight ? (
                  <span className="tag tag--ink" style={{ position: "absolute", top: -10, left: 18, fontSize: 10 }}>
                    Популярный
                  </span>
                ) : null}
                <div>
                  <h3 className="expr-headline" style={{ fontSize: 32 }}>{plan.name}</h3>
                  <p className="muted" style={{ fontSize: 13, marginTop: 6 }}>{plan.description}</p>
                </div>

                <div className="row" style={{ alignItems: "baseline", gap: 6 }}>
                  <span className="expr-headline" style={{ fontSize: 36 }}>
                    <span className="bold">{plan.price}</span>
                  </span>
                  <BynSign size={20} />
                  <span className="mono muted" style={{ fontSize: 12 }}>/ мес</span>
                </div>

                <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "grid", gap: 6 }}>
                  {plan.features.map((feature) => (
                    <li key={feature} className="row" style={{ alignItems: "baseline", gap: 8, fontSize: 13 }}>
                      <Icon name="check" size={12} />
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>

                <div className="row mono" style={{ gap: 10, flexWrap: "wrap", fontSize: 10, color: "var(--muted)", letterSpacing: "0.04em" }}>
                  <span>интервью: {formatLimit(plan.limits.interviews_per_month)}</span>
                  <span>·</span>
                  <span>резюме: {formatLimit(plan.limits.resumes_per_month)}</span>
                  <span>·</span>
                  <span>github: {formatLimit(plan.limits.github_imports_per_month)}</span>
                </div>

                {isCurrent ? (
                  <button className="btn btn--ghost" type="button" disabled style={{ marginTop: 4 }}>
                    Текущий план
                  </button>
                ) : isFree ? (
                  <button
                    className="btn btn--ghost"
                    type="button"
                    disabled={processing}
                    onClick={cancelSubscription}
                    style={{ marginTop: 4 }}
                  >
                    Перейти на Trial
                  </button>
                ) : (
                  <button
                    className={isHighlight ? "btn btn--accent" : "btn btn--primary"}
                    type="button"
                    disabled={processing}
                    onClick={() => startTestCheckout(plan.tier as "pro" | "platinum")}
                    style={{ marginTop: 4 }}
                  >
                    Тестово оплатить {plan.name}
                  </button>
                )}
              </article>
            );
          })}
        </div>
      </section>

      {/* Транзакции */}
      <section style={{ marginTop: 32, marginBottom: 32 }}>
        <header className="dash-section-head">
          <h2 style={{ fontSize: 28 }}>История транзакций</h2>
        </header>

        {transactions.length === 0 ? (
          <p className="muted" style={{ fontSize: 14, marginTop: 12 }}>
            Транзакций пока нет. После активации платного тарифа здесь появятся записи о платежах.
          </p>
        ) : (
          <div style={{ marginTop: 12, border: "1px solid var(--line)", borderRadius: "var(--r-2)", overflow: "hidden" }}>
            <div className="mono row-between" style={{
              padding: "10px 16px",
              background: "var(--paper-2)",
              fontSize: 10,
              color: "var(--muted)",
              letterSpacing: "0.08em",
              borderBottom: "1px solid var(--line)",
            }}>
              <span style={{ flex: "0 0 180px" }}>ДАТА</span>
              <span style={{ flex: "0 0 110px" }}>СТАТУС</span>
              <span style={{ flex: "0 0 110px" }}>СУММА</span>
              <span style={{ flex: 1 }}>ОПИСАНИЕ</span>
            </div>
            {transactions.map((tx) => (
              <div key={tx.id} className="row-between" style={{
                padding: "12px 16px",
                borderTop: "1px solid var(--line-2)",
                fontSize: 13,
                gap: 12,
              }}>
                <span className="mono" style={{ flex: "0 0 180px", color: "var(--ink-2)" }}>{formatDate(tx.created_at)}</span>
                <span className="mono" style={{
                  flex: "0 0 110px",
                  color: tx.status === "succeeded" ? "var(--accent-ink)" : tx.status === "failed" ? "var(--signal)" : "var(--muted)",
                }}>
                  {txStatusLabel(tx.status)}
                </span>
                <span className="mono" style={{ flex: "0 0 110px" }}>{formatMoney(tx.amount_cents)}</span>
                <span style={{ flex: 1, color: "var(--ink-2)" }}>{tx.description ?? `Тариф ${tx.tier.toUpperCase()}`}</span>
              </div>
            ))}
          </div>
        )}
      </section>
    </>
  );
}

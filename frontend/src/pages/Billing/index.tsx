import { useEffect, useMemo, useState } from "react";

import { billingApi, type BillingPlan, type BillingTransaction, type CheckoutIntent, type SubscriptionSnapshot } from "@/shared/api/billing";
import { GlassButton, GlassCard, useToast } from "@/shared/ui";

import { formatBYN } from "@/shared/lib/currency";

const formatMoney = (amountCents: number, _currency?: string) => {
  // Backend stores cents; display as BYN ("Br") regardless of stored currency.
  const whole = Math.round((amountCents / 100) * 100) / 100;
  return formatBYN(whole);
};

const formatDate = (value?: string) => {
  if (!value) return "-";
  return new Date(value).toLocaleString("ru-RU");
};

export default function BillingPage() {
  const { pushToast } = useToast();
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [plans, setPlans] = useState<BillingPlan[]>([]);
  const [subscription, setSubscription] = useState<SubscriptionSnapshot | null>(null);
  const [transactions, setTransactions] = useState<BillingTransaction[]>([]);
  const [activeIntent, setActiveIntent] = useState<CheckoutIntent | null>(null);

  const paidPlans = useMemo(() => plans.filter((plan) => plan.tier !== "trial"), [plans]);

  const loadAll = async () => {
    setLoading(true);
    try {
      const [plansData, subscriptionData, txData] = await Promise.all([
        billingApi.getPlans(),
        billingApi.getSubscription(),
        billingApi.listTransactions(),
      ]);
      setPlans(plansData);
      setSubscription(subscriptionData);
      setTransactions(txData);
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
      pushToast("Тестовый intent создан. Подтвердите оплату.");
    } catch (error) {
      console.error(error);
      pushToast("Не удалось создать тестовый платеж");
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
      pushToast("Тестовая оплата подтверждена, подписка активирована");
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
    <section className="page">
      <div className="section-header">
        <h1>Тестовая система оплаты</h1>
        <span className="muted">Режим без реального списания средств</span>
      </div>

      <div className="two-col">
        <GlassCard>
          <p className="eyebrow">Текущая подписка</p>
          {loading ? (
            <p className="muted">Загрузка...</p>
          ) : subscription ? (
            <div className="career-list">
              <div className="career-list-item">
                <strong>Тариф: {subscription.tier.toUpperCase()}</strong>
                <p className="muted">Статус: {subscription.status}</p>
                <p className="muted">До: {formatDate(subscription.end_date)}</p>
              </div>
              <GlassButton type="button" variant="ghost" onClick={cancelSubscription} disabled={processing}>
                Отменить подписку
              </GlassButton>
            </div>
          ) : (
            <p className="muted">Активной подписки пока нет</p>
          )}
        </GlassCard>

        <GlassCard>
          <p className="eyebrow">Тестовый checkout intent</p>
          {activeIntent ? (
            <div className="career-list">
              <div className="career-list-item">
                <strong>{activeIntent.id}</strong>
                <p className="muted">Сумма: {formatMoney(activeIntent.amount_cents, activeIntent.currency)}</p>
                <p className="muted">Статус: {activeIntent.status}</p>
                <p className="muted">Истекает: {formatDate(activeIntent.expires_at)}</p>
              </div>
              <GlassButton
                type="button"
                onClick={confirmTestCheckout}
                disabled={processing || activeIntent.status !== "requires_confirmation"}
              >
                Подтвердить тестовую оплату
              </GlassButton>
            </div>
          ) : (
            <p className="muted">Создайте intent, выбрав платный тариф ниже</p>
          )}
        </GlassCard>
      </div>

      <GlassCard>
        <p className="eyebrow">Тарифы</p>
        <div className="career-content-grid">
          {paidPlans.map((plan) => (
            <div key={plan.id} className="career-list-item">
              <h3>{plan.name}</h3>
              <p className="muted">{plan.description}</p>
              <p>
                <strong>{formatBYN(plan.price)}</strong>
                <span className="muted"> / {plan.billing_cycle}</span>
              </p>
              <ul className="simple-list">
                {plan.features.slice(0, 4).map((feature) => (
                  <li key={feature}>{feature}</li>
                ))}
              </ul>
              <GlassButton
                type="button"
                onClick={() => startTestCheckout(plan.tier as "pro" | "platinum")}
                disabled={processing}
              >
                Тестово оплатить {plan.name}
              </GlassButton>
            </div>
          ))}
        </div>
      </GlassCard>

      <GlassCard>
        <p className="eyebrow">История транзакций</p>
        {transactions.length === 0 ? (
          <p className="muted">Транзакций пока нет</p>
        ) : (
          <ul className="simple-list">
            {transactions.map((tx) => (
              <li key={tx.id}>
                {formatDate(tx.created_at)} - {tx.status.toUpperCase()} - {formatMoney(tx.amount_cents, tx.currency)} - {tx.description}
              </li>
            ))}
          </ul>
        )}
      </GlassCard>
    </section>
  );
}

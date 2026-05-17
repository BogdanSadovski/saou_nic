import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import {
  TIER_CATALOG,
  useSubscriptionStore,
  type Tier,
} from "@/app/store";
import { useToast } from "@/shared/ui";

/**
 * Mock external checkout — looks intentionally different from the rest
 * of the app (its own minimal layout, "RealPay" branding, no nav bar)
 * so the user perceives it as a 3rd-party gateway like Stripe Checkout.
 *
 * Flow:
 *   /billing/checkout?tier=pro&amount=1490
 *      ↓ user fills card form
 *   "Processing..." (1.5s)
 *      ↓ on success: applyPayment to subscriptionStore
 *   redirect → /profile?paid=pro
 */
export default function CheckoutPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { pushToast } = useToast();
  const applyPayment = useSubscriptionStore((s) => s.applyPayment);

  const tier = (params.get("tier") ?? "pro") as Exclude<Tier, "free">;
  const fallbackPrice = useMemo(
    () => TIER_CATALOG.find((t) => t.tier === tier)?.price ?? 0,
    [tier],
  );
  const amount = Number(params.get("amount")) || fallbackPrice;
  const tierTitle = TIER_CATALOG.find((t) => t.tier === tier)?.title ?? tier;

  // Card form state -------------------------------------------------------
  const [cardNumber, setCardNumber] = useState("");
  const [cardName, setCardName] = useState("");
  const [expiry, setExpiry] = useState("");
  const [cvc, setCvc] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Block scroll on body so the checkout feels like a real overlay-page
  useEffect(() => {
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, []);

  const formatCardNumber = (raw: string): string => {
    const digits = raw.replace(/\D/g, "").slice(0, 16);
    return digits.replace(/(.{4})/g, "$1 ").trim();
  };

  const formatExpiry = (raw: string): string => {
    const digits = raw.replace(/\D/g, "").slice(0, 4);
    if (digits.length <= 2) return digits;
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  };

  const validate = (): string | null => {
    const cardDigits = cardNumber.replace(/\D/g, "");
    if (cardDigits.length !== 16) return "Номер карты должен содержать 16 цифр";
    if (!cardName.trim()) return "Укажите имя владельца карты";
    if (!/^\d{2}\/\d{2}$/.test(expiry)) return "Срок в формате ММ/ГГ";
    const [mmStr, yyStr] = expiry.split("/");
    const mm = Number(mmStr);
    if (mm < 1 || mm > 12) return "Месяц должен быть 01–12";
    const yy = Number(yyStr);
    const now = new Date();
    const currentYY = now.getFullYear() % 100;
    const currentMM = now.getMonth() + 1;
    if (yy < currentYY || (yy === currentYY && mm < currentMM)) {
      return "Срок действия карты истёк";
    }
    if (!/^\d{3,4}$/.test(cvc)) return "CVC из 3–4 цифр";
    return null;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    const v = validate();
    if (v) {
      setError(v);
      return;
    }

    setSubmitting(true);

    // Simulate gateway round-trip (auth → 3DS → settlement). 1.5s feels
    // closer to a real provider than instantly resolving.
    await new Promise((r) => setTimeout(r, 1500));

    const last4 = cardNumber.replace(/\D/g, "").slice(-4);
    const now = new Date();
    const expires = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);

    await applyPayment({
      tier,
      amount,
      currency: "USD",
      cardLast4: last4,
      paidAt: now.toISOString(),
      expiresAt: expires.toISOString(),
    });

    pushToast(`Тариф ${tierTitle} активирован`);
    navigate(`/profile?paid=${tier}`, { replace: true });
  };

  const handleCancel = () => {
    navigate("/profile?paid=cancelled", { replace: true });
  };

  return (
    <div className="checkout-shell">
      <div className="checkout-card" role="dialog" aria-labelledby="checkout-title">
        <header className="checkout-header">
          <span className="checkout-brand">
            <span className="checkout-brand-mark" aria-hidden="true">
              ⛨
            </span>
            RealPay
          </span>
          <span className="checkout-secure">🔒 Защищено TLS 1.3</span>
        </header>

        <h1 id="checkout-title" className="checkout-title">
          Оплата подписки {tierTitle}
        </h1>
        <p className="checkout-merchant">
          Получатель: <strong>RealSync Interview Platform</strong>
        </p>

        <div className="checkout-amount-row">
          <span className="muted">К оплате</span>
          <strong className="checkout-amount">${amount}/mo</strong>
        </div>

        <form className="checkout-form" onSubmit={handleSubmit} noValidate>
          <label className="checkout-field">
            <span>Номер карты</span>
            <input
              autoComplete="cc-number"
              inputMode="numeric"
              maxLength={19}
              onChange={(e) => setCardNumber(formatCardNumber(e.target.value))}
              placeholder="1234 5678 9012 3456"
              required
              type="text"
              value={cardNumber}
            />
          </label>

          <label className="checkout-field">
            <span>Имя владельца</span>
            <input
              autoComplete="cc-name"
              onChange={(e) => setCardName(e.target.value.toUpperCase())}
              placeholder="IVAN IVANOV"
              required
              type="text"
              value={cardName}
            />
          </label>

          <div className="checkout-row">
            <label className="checkout-field">
              <span>Срок действия</span>
              <input
                autoComplete="cc-exp"
                inputMode="numeric"
                maxLength={5}
                onChange={(e) => setExpiry(formatExpiry(e.target.value))}
                placeholder="ММ/ГГ"
                required
                type="text"
                value={expiry}
              />
            </label>
            <label className="checkout-field">
              <span>CVC</span>
              <input
                autoComplete="cc-csc"
                inputMode="numeric"
                maxLength={4}
                onChange={(e) => setCvc(e.target.value.replace(/\D/g, "").slice(0, 4))}
                placeholder="•••"
                required
                type="password"
                value={cvc}
              />
            </label>
          </div>

          {error ? (
            <p className="checkout-error" role="alert">
              {error}
            </p>
          ) : null}

          <div className="checkout-actions">
            <button
              className="checkout-cancel"
              disabled={submitting}
              onClick={handleCancel}
              type="button"
            >
              Отмена
            </button>
            <button
              className="checkout-submit"
              disabled={submitting}
              type="submit"
            >
              {submitting ? "Обрабатываем платёж..." : `Оплатить $${amount}`}
            </button>
          </div>

          <p className="checkout-disclaimer">
            Это тестовая страница. Реальное списание не происходит — никаких
            данных карты в платёжный сервис не отправляется. Используйте
            любые валидные форматы для проверки потока.
          </p>
        </form>
      </div>
    </div>
  );
}

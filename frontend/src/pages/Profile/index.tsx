import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import {
  TIER_CATALOG,
  getTierTitle,
  useAuthStore,
  usePreferencesStore,
  useSubscriptionStore,
  useUIStore,
  useUserStore,
} from "@/app/store";
import { GithubConnectCard } from "@/features/github-connect/GithubConnectCard";
import { reportsApi } from "@/shared/api";
import type { UserInterviewAnalyticsReport } from "@/shared/api/reports";
import { useTranslation } from "@/shared/i18n";
import {
  EmptyState,
  FloatingInput,
  GlassButton,
  GlassCard,
  Icon,
  Skeleton,
  useToast,
} from "@/shared/ui";

export default function ProfilePage() {
  const t = useTranslation();
  const navigate = useNavigate();
  const { pushToast } = useToast();

  // User identity ----------------------------------------------------------
  const user = useUserStore((s) => s.user);
  const updateProfile = useUserStore((s) => s.updateProfile);
  const hydrate = useUserStore((s) => s.hydrate);
  const logout = useAuthStore((s) => s.logout);

  const [fullName, setFullName] = useState(user.fullName);
  const [email, setEmail] = useState(user.email);
  const [savingProfile, setSavingProfile] = useState(false);

  useEffect(() => {
    void hydrate();
  }, [hydrate]);

  useEffect(() => {
    setFullName(user.fullName);
    setEmail(user.email);
  }, [user.fullName, user.email]);

  // Activity stats (real data) ---------------------------------------------
  const [report, setReport] = useState<UserInterviewAnalyticsReport | null>(null);
  const [reportLoading, setReportLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setReportLoading(true);
      try {
        const data = await reportsApi.getMyInterviewReport();
        if (!cancelled) setReport(data);
      } catch (e) {
        const status = (e as { response?: { status?: number } })?.response?.status;
        if (!cancelled && status === 404) setReport(reportsApi.emptyReport());
      } finally {
        if (!cancelled) setReportLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Preferences ------------------------------------------------------------
  const theme = useUIStore((s) => s.theme);
  const setTheme = useUIStore((s) => s.setTheme);
  const prefs = usePreferencesStore();

  // Subscription -----------------------------------------------------------
  const subscription = useSubscriptionStore();
  const cancelSubscription = useSubscriptionStore((s) => s.cancel);
  const refreshSubscription = useSubscriptionStore((s) => s.refresh);
  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    // Pull fresh state from localStorage in case the user came back
    // from /billing/checkout — the store may already be primed but a
    // full reload during checkout would have wiped in-memory state.
    refreshSubscription();
    const paid = searchParams.get("paid");
    if (paid === "cancelled") {
      pushToast("Платёж отменён");
      const next = new URLSearchParams(searchParams);
      next.delete("paid");
      setSearchParams(next, { replace: true });
    } else if (paid && paid !== "free") {
      pushToast(`Тариф ${getTierTitle(paid as never)} активирован`);
      const next = new URLSearchParams(searchParams);
      next.delete("paid");
      setSearchParams(next, { replace: true });
    }
    // We only want this effect to fire once on mount + when the URL changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSaveProfile = async () => {
    setSavingProfile(true);
    try {
      await updateProfile({ fullName, email });
      pushToast(t.profileUpdated);
    } catch {
      pushToast("Не удалось обновить профиль");
    } finally {
      setSavingProfile(false);
    }
  };

  const handleExportData = () => {
    const payload = {
      exported_at: new Date().toISOString(),
      user,
      preferences: prefs,
      report,
    };
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `realsync-export-${user.id}-${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);
    pushToast("Данные экспортированы");
  };

  const handleLogout = () => {
    logout();
    pushToast("Вы вышли из аккаунта");
    navigate("/auth", { replace: true });
  };

  // Initials for the hero avatar.
  const initials = (() => {
    const src = (user.fullName || user.email || "").trim();
    if (!src) return "·";
    const parts = src.split(/[\s@]+/).filter(Boolean);
    if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
    return (parts[0]![0]! + parts[1]![0]!).toUpperCase();
  })();

  const handleCopyId = async () => {
    if (!user.id) return;
    try {
      await navigator.clipboard.writeText(user.id);
      pushToast("ID скопирован");
    } catch {
      pushToast("Не удалось скопировать");
    }
  };

  return (
    <section className="page profile-page">
      {/* Hero card with avatar, identity, quick stats */}
      <GlassCard className="profile-hero">
        <div className="profile-hero-main">
          <div className="profile-avatar-xl" aria-hidden="true">
            {initials}
          </div>
          <div className="profile-hero-info">
            <span className="eyebrow">Личный профиль</span>
            <h1 className="profile-hero-name">
              {user.fullName || user.email || "Профиль"}
            </h1>
            <p className="muted profile-hero-email">
              <Icon name="user" size={14} />
              <span>{user.email || "—"}</span>
            </p>
            <div className="profile-hero-tags">
              <span className={`report-status report-status-${user.role === "admin" ? "pending" : "active"}`}>
                <Icon name="shield" size={12} /> {user.role}
              </span>
              {subscription.tier !== "free" ? (
                <span className={`report-status report-status-${subscription.tier}`}>
                  <Icon name="sparkles" size={12} /> {getTierTitle(subscription.tier)}
                </span>
              ) : (
                <span className="report-status">
                  <Icon name="credit" size={12} /> Free
                </span>
              )}
              {user.connectedGithub ? (
                <span className="report-status">
                  <Icon name="github" size={12} /> GitHub подключён
                </span>
              ) : null}
            </div>
            {user.id ? (
              <button className="profile-id-pill" onClick={() => void handleCopyId()} type="button" title="Скопировать ID">
                <span className="muted">ID:</span>
                <code>{user.id.slice(0, 8)}…</code>
                <Icon name="resume" size={14} />
              </button>
            ) : null}
          </div>
        </div>
        <div className="profile-hero-quick">
          {reportLoading ? (
            <Skeleton variant="card" height={64} />
          ) : (
            <>
              <div className="profile-hero-quick-item">
                <Icon name="mic" size={16} />
                <span className="muted">Интервью</span>
                <strong>{report?.totals.total_interviews ?? 0}</strong>
              </div>
              <div className="profile-hero-quick-item">
                <Icon name="chart" size={16} />
                <span className="muted">Средний балл</span>
                <strong>{Math.round(report?.performance.average_score ?? 0)}</strong>
              </div>
              <div className="profile-hero-quick-item">
                <Icon name="sparkles" size={16} />
                <span className="muted">Лучший балл</span>
                <strong>{Math.round(report?.performance.best_score ?? 0)}</strong>
              </div>
            </>
          )}
        </div>
      </GlassCard>

      <div className="profile-grid">
        {/* Account info */}
        <GlassCard>
          <h3 className="card-title-with-icon">
            <Icon name="user" size={18} /> Учётная запись
          </h3>
          <p className="muted">{t.manageLocalIdentity}</p>

          <FloatingInput
            label={t.fullName}
            onChange={(e) => setFullName(e.target.value)}
            value={fullName}
          />
          <FloatingInput
            label={t.email}
            onChange={(e) => setEmail(e.target.value)}
            value={email}
          />
          <p className="muted profile-role">
            {t.role}: <strong>{user.role}</strong>
          </p>

          <div className="modal-actions">
            <GlassButton
              disabled={savingProfile}
              onClick={() => void handleSaveProfile()}
              type="button"
            >
              {savingProfile ? "Сохраняем..." : t.saveChanges}
            </GlassButton>
            <GlassButton onClick={handleLogout} type="button" variant="ghost">
              <span className="btn-with-icon">
                <Icon name="logout" size={16} />
                <span>Выйти</span>
              </span>
            </GlassButton>
          </div>
        </GlassCard>

        {/* Activity stats */}
        <GlassCard>
          <h3 className="card-title-with-icon">
            <Icon name="chart" size={18} /> Моя активность
          </h3>
          {reportLoading ? (
            <Skeleton count={4} />
          ) : !report || report.totals.total_interviews === 0 ? (
            <EmptyState
              icon="📊"
              title="Пока нет интервью"
              hint="Пройдите первое — здесь появятся ваши метрики и темп прогресса."
              action={
                <GlassButton onClick={() => navigate("/interview")} type="button" variant="primary">
                  Начать интервью
                </GlassButton>
              }
            />
          ) : (
            <div className="profile-stats-grid">
              <div className="profile-stat">
                <span className="muted">Всего интервью</span>
                <strong>{report.totals.total_interviews}</strong>
              </div>
              <div className="profile-stat">
                <span className="muted">Завершено</span>
                <strong>{report.totals.completed_interviews}</strong>
              </div>
              <div className="profile-stat">
                <span className="muted">Средний балл</span>
                <strong>{Math.round(report.performance.average_score)}</strong>
              </div>
              <div className="profile-stat">
                <span className="muted">Лучший балл</span>
                <strong>{Math.round(report.performance.best_score)}</strong>
              </div>
              <div className="profile-stat">
                <span className="muted">Завершаемость</span>
                <strong>{Math.round(report.totals.completion_rate * 100)}%</strong>
              </div>
              <div className="profile-stat">
                <span className="muted">Отчёты</span>
                <strong>{report.performance.reports_generated}</strong>
              </div>
            </div>
          )}
        </GlassCard>
      </div>

      {/* Settings */}
      <GlassCard>
        <h3 className="card-title-with-icon">
          <Icon name="settings" size={18} /> Настройки
        </h3>
        <div className="settings-grid">
          <div className="settings-row">
            <div>
              <strong>Тема оформления</strong>
              <p className="muted">Светлая, тёмная или подстраивается под систему.</p>
            </div>
            <div className="settings-segmented" role="radiogroup">
              {(["light", "system", "dark"] as const).map((mode) => (
                <button
                  key={mode}
                  className={theme === mode ? "is-active" : ""}
                  onClick={() => setTheme(mode)}
                  role="radio"
                  aria-checked={theme === mode}
                  type="button"
                >
                  {mode === "light" ? "Светлая" : mode === "dark" ? "Тёмная" : "Авто"}
                </button>
              ))}
            </div>
          </div>

          <div className="settings-row">
            <div>
              <strong>Плотный режим</strong>
              <p className="muted">Уплотняет таблицы и карточки.</p>
            </div>
            <Toggle
              checked={prefs.compactDensity}
              onChange={(v) => prefs.setCompactDensity(v)}
              ariaLabel="Плотный режим"
            />
          </div>

          <div className="settings-row">
            <div>
              <strong>Снижение анимаций</strong>
              <p className="muted">Уменьшает скорость переходов и эффектов.</p>
            </div>
            <Toggle
              checked={prefs.reduceMotion}
              onChange={(v) => prefs.setReduceMotion(v)}
              ariaLabel="Снижение анимаций"
            />
          </div>

          <div className="settings-row">
            <div>
              <strong>Звуки во время интервью</strong>
              <p className="muted">Тонкие сигналы таймера и готовности AI.</p>
            </div>
            <Toggle
              checked={prefs.soundEnabled}
              onChange={(v) => prefs.setSoundEnabled(v)}
              ariaLabel="Звуки"
            />
          </div>
        </div>
      </GlassCard>

      {/* Notifications */}
      <GlassCard>
        <h3 className="card-title-with-icon">
          <Icon name="bell" size={18} /> Уведомления
        </h3>
        <p className="muted">Будем присылать только то, что вы сами выбрали.</p>
        <div className="settings-grid">
          {(
            [
              ["interview_reminder", "Напоминания об интервью"],
              ["result_ready", "Готовы результаты"],
              ["weekly_digest", "Еженедельный дайджест прогресса"],
            ] as const
          ).map(([channel, label]) => (
            <div className="settings-row" key={channel}>
              <div>
                <strong>{label}</strong>
              </div>
              <Toggle
                checked={prefs.notifications[channel]}
                onChange={(v) => prefs.setNotification(channel, v)}
                ariaLabel={label}
              />
            </div>
          ))}
        </div>
      </GlassCard>

      {/* Subscription / billing */}
      <GlassCard>
        <div className="section-header">
          <h3 className="card-title-with-icon">
            <Icon name="credit" size={18} /> Подписка
          </h3>
          {subscription.tier !== "free" ? (
            <span className={`report-status report-status-${subscription.tier}`}>
              Активный тариф: {getTierTitle(subscription.tier)}
            </span>
          ) : (
            <span className="report-status">Тариф: Free</span>
          )}
        </div>

        {subscription.intent ? (
          <div className="subscription-summary">
            <div>
              <strong>{getTierTitle(subscription.tier)}</strong>
              <span className="muted"> · продлится до {new Date(subscription.intent.expiresAt).toLocaleDateString("ru-RU")}</span>
            </div>
            <div className="muted">
              Карта •••• {subscription.intent.cardLast4} ·{" "}
              {subscription.intent.amount.toLocaleString("ru-RU")} ₽
            </div>
            <div className="modal-actions">
              <GlassButton
                onClick={() =>
                  navigate(
                    `/billing/checkout?tier=${subscription.tier}&amount=${subscription.intent?.amount ?? 0}`,
                  )
                }
                type="button"
                variant="ghost"
              >
                Продлить
              </GlassButton>
              <GlassButton
                onClick={() => {
                  cancelSubscription();
                  pushToast("Подписка отменена");
                }}
                type="button"
                variant="ghost"
              >
                Отменить
              </GlassButton>
            </div>
          </div>
        ) : null}

        <div className="tier-grid">
          {TIER_CATALOG.map((tier) => {
            const isCurrent = subscription.tier === tier.tier;
            return (
              <div
                className={`tier-card${tier.highlight ? " is-highlight" : ""}${isCurrent ? " is-current" : ""}`}
                key={tier.tier}
              >
                <div className="tier-card-head">
                  <span className="tier-name">{tier.title}</span>
                  {tier.highlight ? <span className="tier-badge">Популярный</span> : null}
                </div>
                <div className="tier-price">
                  <strong>{tier.price.toLocaleString("ru-RU")} ₽</strong>
                  <span className="muted">/мес</span>
                </div>
                <ul className="tier-perks">
                  {tier.perks.map((perk) => (
                    <li key={perk}>{perk}</li>
                  ))}
                </ul>
                <button
                  className={`tier-cta${tier.highlight ? " is-primary" : ""}`}
                  disabled={isCurrent}
                  onClick={() =>
                    navigate(`/billing/checkout?tier=${tier.tier}&amount=${tier.price}`)
                  }
                  type="button"
                >
                  {isCurrent ? "Текущий тариф" : `Перейти на ${tier.title}`}
                </button>
              </div>
            );
          })}
        </div>

        <p className="muted tier-disclaimer">
          Оплата проходит через защищённый платёжный шлюз — данные карты
          не сохраняются на нашей стороне. Подписка продлевается ежемесячно,
          её можно отменить в один клик.
        </p>
      </GlassCard>

      {/* Connected accounts */}
      <GlassCard>
        <h3 className="card-title-with-icon">
          <Icon name="github" size={18} /> Связанные аккаунты
        </h3>
        <GithubConnectCard />
      </GlassCard>

      {/* Data + danger zone */}
      <GlassCard>
        <h3 className="card-title-with-icon">
          <Icon name="shield" size={18} /> Данные и приватность
        </h3>
        <div className="settings-grid">
          <div className="settings-row">
            <div>
              <strong>Экспорт данных</strong>
              <p className="muted">
                Скачать JSON-снимок профиля, настроек и аналитики интервью.
              </p>
            </div>
            <GlassButton onClick={handleExportData} type="button" variant="ghost">
              Скачать .json
            </GlassButton>
          </div>
          <div className="settings-row">
            <div>
              <strong>Сбросить настройки</strong>
              <p className="muted">Возвращает темy, плотность и уведомления к значениям по умолчанию.</p>
            </div>
            <GlassButton
              onClick={() => {
                prefs.reset();
                pushToast("Настройки сброшены");
              }}
              type="button"
              variant="ghost"
            >
              Сбросить
            </GlassButton>
          </div>
        </div>
      </GlassCard>
    </section>
  );
}

type ToggleProps = {
  checked: boolean;
  onChange: (value: boolean) => void;
  ariaLabel: string;
};

/**
 * Accessible iOS-style toggle. Pure CSS, no extra deps.
 */
function Toggle({ checked, onChange, ariaLabel }: ToggleProps) {
  return (
    <button
      aria-checked={checked}
      aria-label={ariaLabel}
      className={`pref-toggle${checked ? " is-on" : ""}`}
      onClick={() => onChange(!checked)}
      role="switch"
      type="button"
    >
      <span className="pref-toggle__knob" />
    </button>
  );
}

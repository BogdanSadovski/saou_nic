import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  useAuthStore,
  usePreferencesStore,
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

  return (
    <section className="page profile-page">
      <h1>{t.profileTitle}</h1>

      <div className="profile-grid">
        {/* Account info */}
        <GlassCard>
          <h3>Учётная запись</h3>
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
              Выйти
            </GlassButton>
          </div>
        </GlassCard>

        {/* Activity stats */}
        <GlassCard>
          <h3>Моя активность</h3>
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
        <h3>Настройки</h3>
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
        <h3>Уведомления</h3>
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

      {/* Connected accounts */}
      <GlassCard>
        <h3>Связанные аккаунты</h3>
        <GithubConnectCard />
      </GlassCard>

      {/* Data + danger zone */}
      <GlassCard>
        <h3>Данные и приватность</h3>
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

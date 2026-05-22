import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import {
  getTierTitle,
  useAuthStore,
  usePreferencesStore,
  useSubscriptionStore,
  useUIStore,
  useUserStore,
} from "@/app/store";
import { TelegramConnectCard } from "@/features/telegram-connect/TelegramConnectCard";
import { githubApi, type GithubImportResponse } from "@/shared/api/github";
import { userApi } from "@/shared/api";
import { formatBYNAmount } from "@/shared/lib/currency";
import { BynSign, UserAvatar, useToast } from "@/shared/ui";
import { RsIcon as Icon } from "@/shared/ui/realsync";

export default function ProfilePage() {
  const navigate = useNavigate();
  const { pushToast } = useToast();

  // Identity ---------------------------------------------------------------
  const user = useUserStore((s) => s.user);
  const updateProfile = useUserStore((s) => s.updateProfile);
  const hydrate = useUserStore((s) => s.hydrate);
  const logout = useAuthStore((s) => s.logout);

  const [name, setName] = useState(user.fullName);
  const [email, setEmail] = useState(user.email);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    void hydrate();
  }, [hydrate]);

  useEffect(() => {
    setName(user.fullName);
    setEmail(user.email);
  }, [user.fullName, user.email]);

  // UI / theme -------------------------------------------------------------
  const theme = useUIStore((s) => s.theme);
  const setTheme = useUIStore((s) => s.setTheme);

  // Preferences (compact/motion/sound) ------------------------------------
  const prefs = usePreferencesStore();

  // GitHub -----------------------------------------------------------------
  const [ghUsername, setGhUsername] = useState("");
  const [ghConnecting, setGhConnecting] = useState(false);
  const [ghReport, setGhReport] = useState<GithubImportResponse | null>(() => {
    try {
      const raw = localStorage.getItem("realsync_github_report");
      return raw ? (JSON.parse(raw) as GithubImportResponse) : null;
    } catch {
      return null;
    }
  });
  const [lastGhUsername, setLastGhUsername] = useState<string>(() => {
    return localStorage.getItem("realsync_github_username") || "";
  });

  const importGithub = async (rawValue: string) => {
    const value = rawValue.trim();
    if (!value) {
      pushToast("Введите GitHub-логин");
      return;
    }
    setGhConnecting(true);
    try {
      const profileUrl = /^https?:\/\//i.test(value)
        ? value
        : `https://github.com/${value.replace(/^@/, "")}`;
      const report = await githubApi.importProfile({ profileUrl });
      setGhReport(report);
      const username = report.username || value.replace(/^@/, "");
      setLastGhUsername(username);
      try {
        localStorage.setItem("realsync_github_report", JSON.stringify(report));
        localStorage.setItem("realsync_github_username", username);
      } catch {/* quota */}
      await updateProfile({ connectedGithub: true });
      pushToast("GitHub-профиль импортирован");
      setGhUsername("");
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Не удалось подключить GitHub";
      pushToast(msg);
    } finally {
      setGhConnecting(false);
    }
  };

  const connectGithub = () => importGithub(ghUsername);
  const resyncGithub = () => importGithub(lastGhUsername || ghUsername);

  const disconnectGithub = async () => {
    try {
      await updateProfile({ connectedGithub: false });
      setGhReport(null);
      setLastGhUsername("");
      localStorage.removeItem("realsync_github_report");
      localStorage.removeItem("realsync_github_username");
      pushToast("GitHub отключён");
    } catch {
      pushToast("Не удалось отключить");
    }
  };

  // Subscription -----------------------------------------------------------
  // Read-only сводка для Profile. Управление тарифом — в /workspace/billing.
  const subscription = useSubscriptionStore();
  const refreshSubscription = useSubscriptionStore((s) => s.refresh);
  const hydrateSubscription = useSubscriptionStore((s) => s.hydrate);
  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    refreshSubscription();
    void hydrateSubscription();
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Save profile -----------------------------------------------------------
  const save = async () => {
    setSaving(true);
    try {
      await updateProfile({ fullName: name, email });
      setSaved(true);
      pushToast("Профиль обновлён");
      setTimeout(() => setSaved(false), 1600);
    } catch {
      pushToast("Не удалось сохранить профиль");
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = () => {
    logout();
    pushToast("Вы вышли из аккаунта");
    navigate("/auth", { replace: true });
  };

  // initials больше не нужны — аватар теперь единый <UserAvatar/>.

  // Deterministic GH-style cells
  const cells = useMemo(
    () =>
      Array.from({ length: 49 }, (_, i) => {
        const r = ((i * 31 + 13) % 7) / 7 + ((i * 17) % 5) / 10;
        return r > 1.1 ? "l4" : r > 0.85 ? "l3" : r > 0.55 ? "l2" : r > 0.3 ? "l1" : "";
      }),
    [],
  );

  const firstName = (name || "").split(" ")[0] || "";
  const secondName = (name || "").split(" ").slice(1).join(" ") || "";

  return (
    <>
      <span className="eyebrow">Профиль · персональное</span>
      <h1 className="expr-headline" style={{ fontSize: 72, marginTop: 8 }}>
        <span className="ital">Профиль</span>.
      </h1>

      <div className="profile-grid">
        {/* LEFT COLUMN */}
        <div style={{ display: "grid", gap: 24 }}>
          {/* Личные данные */}
          <section className="profile-card">
            <div className="row" style={{ gap: 20 }}>
              <UserAvatar size={72} alt={user.fullName || user.email} />
              {/* Старый блок с инициалами {initials} заменён на единую
                  ISO-куб иконку (UserAvatar). Поле `initials` больше
                  не используется здесь, но оставлено как fallback для
                  совместимости с другими компонентами. */}
              <div>
                <h2 style={{ fontSize: 36 }}>
                  {firstName} {secondName}
                </h2>
                <p className="muted">
                  {(user.role || "user").toUpperCase()} · {email}
                </p>
              </div>
            </div>

            <div className="profile-fields">
              <div className="field">
                <label>Имя и фамилия</label>
                <input
                  className="input"
                  onChange={(e) => setName(e.target.value)}
                  value={name}
                />
              </div>
              <div className="field">
                <label>Email</label>
                <input
                  className="input"
                  onChange={(e) => setEmail(e.target.value)}
                  value={email}
                />
              </div>
              <div className="field">
                <label>ID пользователя</label>
                <input className="input" readOnly value={user.id || "—"} />
              </div>
              <div className="field">
                <label>Роль</label>
                <div className="row" style={{ gap: 6 }}>
                  <span className="tag tag--ink">
                    {(user.role || "user").toUpperCase()}
                  </span>
                  {user.connectedGithub ? (
                    <span className="tag tag--lime">GitHub связан</span>
                  ) : null}
                </div>
              </div>
            </div>

            <div className="row" style={{ marginTop: 24, gap: 12 }}>
              <button
                className="btn btn--primary"
                disabled={saving}
                onClick={() => void save()}
                type="button"
              >
                {saved ? (
                  <>
                    <Icon name="check" size={14} /> Сохранено
                  </>
                ) : saving ? (
                  "Сохраняем…"
                ) : (
                  "Сохранить изменения"
                )}
              </button>
            </div>
          </section>

          {/* Безопасность */}
          <SecuritySection />

          {/* Подписка — read-only сводка. Покупка/смена тарифа полностью
              вынесена на /workspace/billing, чтобы не дублировать UI. */}
          <section className="profile-card">
            <header className="row-between" style={{ alignItems: "baseline" }}>
              <div>
                <span className="eyebrow">Подписка</span>
                <h2 style={{ fontSize: 28, marginTop: 4 }}>
                  {getTierTitle(subscription.tier)}
                </h2>
              </div>
              {subscription.tier !== "free" ? (
                <span className="tag tag--lime">ACTIVE</span>
              ) : (
                <span className="tag">FREE</span>
              )}
            </header>

            {subscription.intent ? (
              <div style={{ marginTop: 14, display: "grid", gap: 6 }}>
                <p className="muted" style={{ fontSize: 13 }}>
                  Карта •••• {subscription.intent.cardLast4} ·{" "}
                  {formatBYNAmount(subscription.intent.amount)} <BynSign size={12} />/мес
                </p>
                <p className="muted" style={{ fontSize: 13 }}>
                  Действует до{" "}
                  {new Date(subscription.intent.expiresAt).toLocaleDateString("ru-RU")}
                </p>
              </div>
            ) : (
              <p className="muted" style={{ marginTop: 12, fontSize: 13 }}>
                Сейчас бесплатный тариф. Управление подпиской и оплата —
                в разделе «Подписка» рабочего пространства.
              </p>
            )}

            <div className="row" style={{ marginTop: 18, gap: 12 }}>
              <button
                className="btn btn--primary btn--sm"
                onClick={() => navigate("/workspace/billing")}
                type="button"
              >
                Управлять подпиской
              </button>
            </div>
          </section>
        </div>

        {/* RIGHT COLUMN */}
        <div style={{ display: "grid", gap: 24 }}>
          {/* GitHub card */}
          <section className="gh-card">
            <div className="row-between" style={{ alignItems: "baseline" }}>
              <div>
                <span className="eyebrow">GitHub</span>
                <h2 className="display" style={{ fontSize: 36, marginTop: 6 }}>
                  Подключение.
                </h2>
              </div>
              {user.connectedGithub ? (
                <button className="btn btn--ghost btn--sm" type="button" onClick={() => void disconnectGithub()}>
                  <Icon name="github" size={14} /> Отключить
                </button>
              ) : null}
            </div>

            <div
              style={{
                padding: 18,
                border: "1px solid var(--line)",
                borderRadius: "var(--r-1)",
                background: "var(--paper-2)",
              }}
            >
              <div className="row" style={{ gap: 14 }}>
                <div
                  style={{
                    width: 44,
                    height: 44,
                    borderRadius: 12,
                    background: "var(--ink)",
                    color: "var(--bg)",
                    display: "grid",
                    placeItems: "center",
                  }}
                >
                  <Icon name="github" size={20} />
                </div>
                <div style={{ flex: 1 }}>
                  <strong>
                    {user.connectedGithub ? "Подключён" : "Не подключён"}
                  </strong>
                  <div
                    className="mono"
                    style={{ fontSize: 11, color: "var(--muted)" }}
                  >
                    {user.connectedGithub
                      ? "Контрибуции и языки анализируются"
                      : "Подключите аккаунт, чтобы анализировать контрибуции"}
                  </div>
                </div>
                <span
                  className={`tag ${user.connectedGithub ? "tag--lime" : ""}`}
                >
                  {user.connectedGithub ? "СИНХРОН." : "ВЫКЛ"}
                </span>
              </div>

              {!user.connectedGithub ? (
                <form
                  onSubmit={(e) => { e.preventDefault(); void connectGithub(); }}
                  style={{ marginTop: 14, display: "grid", gap: 10 }}
                >
                  <div className="field">
                    <label htmlFor="gh-username">GitHub-логин</label>
                    <input
                      id="gh-username"
                      className="input"
                      value={ghUsername}
                      onChange={(e) => setGhUsername(e.target.value)}
                      placeholder="octocat"
                      autoComplete="off"
                      disabled={ghConnecting}
                    />
                  </div>
                  <button
                    type="submit"
                    className="btn btn--primary btn--sm"
                    disabled={ghConnecting || !ghUsername.trim()}
                  >
                    {ghConnecting ? "Подключаем…" : "Подключить GitHub"}
                  </button>
                </form>
              ) : (
                <div className="row" style={{ marginTop: 12, gap: 10 }}>
                  <button
                    className="btn btn--ghost btn--sm"
                    type="button"
                    onClick={() => void resyncGithub()}
                    disabled={ghConnecting}
                  >
                    <Icon name="github" size={14} /> Пересинхронизировать
                  </button>
                </div>
              )}
            </div>

            <div>
              <div
                className="row-between mono"
                style={{
                  fontSize: 11,
                  color: "var(--muted)",
                  marginBottom: 10,
                  letterSpacing: "0.08em",
                  textTransform: "uppercase",
                }}
              >
                <span>контрибуции — последние 7 недель</span>
                <span>
                  {ghReport
                    ? `${(ghReport.charts.contribution_days || []).reduce((s, d) => s + (d.count || 0), 0)} коммитов`
                    : "187 коммитов"}
                </span>
              </div>
              <div className="gh-grid">
                {cells.map((lv, i) => (
                  <div
                    key={i}
                    className={`gh-cell ${lv}`}
                    style={{
                      animationDelay: `${i * 6}ms`,
                      animation: `reveal 480ms var(--ease-out) ${i * 6}ms backwards`,
                    }}
                  />
                ))}
              </div>
            </div>

            <div className="hr"></div>

            <div className="grid-2" style={{ gap: 12 }}>
              <div>
                <div
                  className="mono"
                  style={{
                    fontSize: 11,
                    color: "var(--muted)",
                    letterSpacing: "0.08em",
                    textTransform: "uppercase",
                  }}
                >
                  Топ-репо
                </div>
                {(() => {
                  const top = ghReport?.top_repositories?.[0];
                  return top ? (
                    <>
                      <div className="display" style={{ fontSize: 22, marginTop: 4 }}>{top.name}</div>
                      <div className="muted" style={{ fontSize: 12 }}>
                        {[
                          top.language,
                          `${top.stars} ⭐`,
                          `${top.forks} forks`,
                        ].filter(Boolean).join(" · ")}
                      </div>
                    </>
                  ) : (
                    <>
                      <div className="display" style={{ fontSize: 22, marginTop: 4 }}>realsync-core</div>
                      <div className="muted" style={{ fontSize: 12 }}>Go · 142 ⭐ · 12 contributors</div>
                    </>
                  );
                })()}
              </div>
              <div>
                <div
                  className="mono"
                  style={{
                    fontSize: 11,
                    color: "var(--muted)",
                    letterSpacing: "0.08em",
                    textTransform: "uppercase",
                  }}
                >
                  Языки
                </div>
                <div
                  className="row"
                  style={{ marginTop: 8, gap: 6, flexWrap: "wrap" }}
                >
                  {(() => {
                    const dist = ghReport?.charts.language_distribution;
                    if (dist && dist.length) {
                      const total = dist.reduce((s, d) => s + d.value, 0) || 1;
                      return dist.slice(0, 4).map((d, i) => (
                        <span key={d.label} className={`tag ${i === 0 ? "tag--lime" : ""}`}>
                          {d.label} {Math.round((d.value / total) * 100)}%
                        </span>
                      ));
                    }
                    return (
                      <>
                        <span className="tag tag--lime">Go 62%</span>
                        <span className="tag">TS 24%</span>
                        <span className="tag">Py 8%</span>
                        <span className="tag">Other 6%</span>
                      </>
                    );
                  })()}
                </div>
              </div>
            </div>
          </section>

          {/* Telegram — daily push + score через бота */}
          <TelegramConnectCard />

          {/* Полная GitHub-аналитика — только когда есть импорт */}
          {ghReport ? (
            <section className="profile-card">
              <header className="row-between" style={{ alignItems: "baseline" }}>
                <div>
                  <span className="eyebrow">GitHub · аналитика</span>
                  <h2 style={{ fontSize: 24, marginTop: 6 }}>
                    {ghReport.profile_name || ghReport.username}
                  </h2>
                </div>
                <a
                  className="mono"
                  href={ghReport.profile_url}
                  rel="noreferrer"
                  target="_blank"
                  style={{ fontSize: 12, color: "var(--muted)", textDecoration: "none" }}
                >
                  @{ghReport.username} ↗
                </a>
              </header>

              {/* Stats sysbar */}
              <div className="sysbar" style={{ marginTop: 16 }}>
                <span><span className="k">подписчиков</span><span className="v">{ghReport.stats.followers}</span></span>
                <span><span className="k">репо</span><span className="v">{ghReport.stats.public_repos}</span></span>
                <span><span className="k">звёзд</span><span className="v">{ghReport.stats.total_stars}</span></span>
                <span><span className="k">форков</span><span className="v">{ghReport.stats.total_forks}</span></span>
                <span><span className="k">issues</span><span className="v">{ghReport.stats.total_open_issues}</span></span>
              </div>

              {/* AI summary */}
              {ghReport.ai_insights.summary ? (
                <p style={{ marginTop: 16, fontSize: 14, color: "var(--ink-2)", lineHeight: 1.55 }}>
                  {ghReport.ai_insights.summary}
                </p>
              ) : null}

              {/* Strengths + Risks */}
              <div className="grid-2" style={{ gap: 18, marginTop: 18 }}>
                {ghReport.ai_insights.strengths?.length ? (
                  <div>
                    <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 8 }}>
                      Сильные стороны
                    </div>
                    <ul style={{ display: "grid", gap: 6 }}>
                      {ghReport.ai_insights.strengths.slice(0, 4).map((s, i) => (
                        <li key={i} style={{ fontSize: 13, color: "var(--ink-2)", paddingLeft: 14, position: "relative" }}>
                          <span style={{ position: "absolute", left: 0, color: "var(--accent)" }}>·</span>{s}
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}
                {ghReport.ai_insights.risks?.length ? (
                  <div>
                    <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 8 }}>
                      Зоны риска
                    </div>
                    <ul style={{ display: "grid", gap: 6 }}>
                      {ghReport.ai_insights.risks.slice(0, 4).map((s, i) => (
                        <li key={i} style={{ fontSize: 13, color: "var(--ink-2)", paddingLeft: 14, position: "relative" }}>
                          <span style={{ position: "absolute", left: 0, color: "oklch(0.65 0.14 25)" }}>·</span>{s}
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </div>

              {/* Recommended positions */}
              {ghReport.ai_insights.recommended_positions?.length ? (
                <div style={{ marginTop: 18 }}>
                  <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 10 }}>
                    Рекомендованные позиции
                  </div>
                  <div style={{ display: "grid", gap: 8 }}>
                    {ghReport.ai_insights.recommended_positions.slice(0, 3).map((p) => (
                      <div
                        key={p.role}
                        style={{
                          display: "grid",
                          gridTemplateColumns: "1fr 60px",
                          alignItems: "center",
                          gap: 12,
                          padding: "10px 12px",
                          background: "var(--paper-2)",
                          border: "1px solid var(--line)",
                          borderRadius: "var(--r-1)",
                        }}
                      >
                        <div>
                          <strong style={{ fontSize: 14 }}>{p.role}</strong>
                          <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>{p.rationale}</div>
                        </div>
                        <span className="mono" style={{ fontSize: 13, color: "var(--accent-ink, var(--ink))", textAlign: "right" }}>
                          {p.fit_score}%
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}

              {/* Top repos */}
              {ghReport.top_repositories?.length ? (
                <div style={{ marginTop: 18 }}>
                  <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 10 }}>
                    Топ-репозитории
                  </div>
                  <div style={{ display: "grid", gap: 8 }}>
                    {ghReport.top_repositories.slice(0, 4).map((r) => (
                      <a
                        key={r.name}
                        href={r.url}
                        target="_blank"
                        rel="noreferrer"
                        style={{
                          display: "grid",
                          gridTemplateColumns: "1fr auto",
                          gap: 10,
                          alignItems: "baseline",
                          padding: "10px 12px",
                          background: "var(--paper-2)",
                          border: "1px solid var(--line)",
                          borderRadius: "var(--r-1)",
                          textDecoration: "none",
                          color: "var(--ink)",
                        }}
                      >
                        <div>
                          <strong style={{ fontSize: 14 }}>{r.name}</strong>
                          {r.description ? (
                            <div className="muted" style={{ fontSize: 12, marginTop: 2 }}>{r.description}</div>
                          ) : null}
                        </div>
                        <span className="mono" style={{ fontSize: 12, color: "var(--muted)" }}>
                          {r.language || "—"} · {r.stars} ⭐
                        </span>
                      </a>
                    ))}
                  </div>
                </div>
              ) : null}

              {/* Interview tracks */}
              {ghReport.ai_insights.interview_tracks?.length ? (
                <div style={{ marginTop: 18 }}>
                  <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 10 }}>
                    Рекомендованные треки интервью
                  </div>
                  <div style={{ display: "grid", gap: 8 }}>
                    {ghReport.ai_insights.interview_tracks.slice(0, 3).map((tr, i) => (
                      <button
                        key={`${tr.role}-${i}`}
                        type="button"
                        onClick={() => {
                          const sp = new URLSearchParams({
                            role: tr.role,
                            mode: tr.mode,
                            level: tr.level,
                            duration: String(tr.duration_minutes),
                          });
                          navigate(`/interview?${sp.toString()}`);
                        }}
                        style={{
                          display: "grid",
                          gridTemplateColumns: "1fr auto",
                          gap: 10,
                          alignItems: "center",
                          padding: "12px 14px",
                          background: "var(--paper-2)",
                          border: "1px solid var(--line)",
                          borderRadius: "var(--r-1)",
                          cursor: "pointer",
                          textAlign: "left",
                          color: "var(--ink)",
                        }}
                      >
                        <div>
                          <strong style={{ fontSize: 14 }}>{tr.role}</strong>
                          <div className="mono muted" style={{ fontSize: 11, marginTop: 2 }}>
                            {tr.mode === "theory" ? "теория" : "практика"} · {tr.level} · {tr.duration_minutes} мин
                          </div>
                        </div>
                        <Icon name="arrow" size={14} />
                      </button>
                    ))}
                  </div>
                </div>
              ) : null}

              {/* Action plan */}
              {ghReport.ai_insights.action_plan?.length ? (
                <div style={{ marginTop: 18 }}>
                  <div className="mono" style={{ fontSize: 11, color: "var(--muted)", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 10 }}>
                    План действий
                  </div>
                  <ol style={{ display: "grid", gap: 8 }}>
                    {ghReport.ai_insights.action_plan.slice(0, 4).map((p, i) => (
                      <li
                        key={i}
                        style={{
                          display: "grid",
                          gridTemplateColumns: "28px 1fr",
                          gap: 10,
                          fontSize: 13,
                          color: "var(--ink-2)",
                        }}
                      >
                        <span className="mono" style={{ color: "var(--muted)", fontSize: 11 }}>{String(i + 1).padStart(2, "0")}</span>
                        <span>{p}</span>
                      </li>
                    ))}
                  </ol>
                </div>
              ) : null}
            </section>
          ) : null}

          {/* Настройки интерфейса */}
          <section className="profile-card">
            <header className="dash-section-head">
              <h2 style={{ fontSize: 24 }}>Интерфейс</h2>
            </header>
            <div className="profile-fields">
              <div className="field">
                <label>Тема</label>
                <div className="segmented">
                  {(["light", "dark"] as const).map((mode) => (
                    <button
                      key={mode}
                      aria-checked={theme === mode}
                      className={theme === mode ? "is-active" : ""}
                      onClick={() => setTheme(mode)}
                      role="radio"
                      type="button"
                    >
                      {mode === "light" ? "Светлая" : "Тёмная"}
                    </button>
                  ))}
                </div>
              </div>
              <div className="field">
                <label>Плотный режим</label>
                <div className="segmented">
                  <button
                    className={!prefs.compactDensity ? "is-active" : ""}
                    onClick={() => prefs.setCompactDensity(false)}
                    type="button"
                  >
                    Обычный
                  </button>
                  <button
                    className={prefs.compactDensity ? "is-active" : ""}
                    onClick={() => prefs.setCompactDensity(true)}
                    type="button"
                  >
                    Компактный
                  </button>
                </div>
                <small className="muted mono" style={{ fontSize: 11, marginTop: 6, display: "block" }}>
                  Компактный режим уменьшает отступы карточек и таблиц на ~20%.
                </small>
              </div>
              <div className="field">
                <label>Анимации</label>
                <div className="segmented">
                  <button
                    className={!prefs.reduceMotion ? "is-active" : ""}
                    onClick={() => prefs.setReduceMotion(false)}
                    type="button"
                  >
                    Полные
                  </button>
                  <button
                    className={prefs.reduceMotion ? "is-active" : ""}
                    onClick={() => prefs.setReduceMotion(true)}
                    type="button"
                  >
                    Сниженные
                  </button>
                </div>
                <small className="muted mono" style={{ fontSize: 11, marginTop: 6, display: "block" }}>
                  Сниженные анимации убирают transition/animate эффекты — удобно при чувствительности к движению.
                </small>
              </div>
            </div>
          </section>

          {/* Опасная зона */}
          <section
            className="profile-card"
            style={{ borderColor: "var(--line)" }}
          >
            <header className="dash-section-head">
              <h2 style={{ fontSize: 24 }}>Опасная зона</h2>
            </header>
            <p className="muted" style={{ fontSize: 13 }}>
              Выход с устройства и сброс пользовательских настроек до значений
              по умолчанию.
            </p>
            <div className="row" style={{ marginTop: 16, gap: 12, flexWrap: "wrap" }}>
              <button
                className="btn btn--ghost btn--sm"
                onClick={() => {
                  prefs.reset();
                  pushToast("Настройки сброшены");
                }}
                type="button"
              >
                Сбросить настройки
              </button>
              <button
                className="btn btn--ghost btn--sm"
                onClick={handleLogout}
                type="button"
              >
                Выйти
              </button>
            </div>
          </section>
        </div>
      </div>
    </>
  );
}

/**
 * Password-change form. Calls userApi.changePassword and surfaces
 * the validation reason inline.
 */
function SecuritySection() {
  const { pushToast } = useToast();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (next.length < 8) {
      setError("Новый пароль должен быть не короче 8 символов");
      return;
    }
    if (next !== confirm) {
      setError("Пароли не совпадают");
      return;
    }
    setSubmitting(true);
    try {
      await userApi.changePassword(current, next);
      setCurrent("");
      setNext("");
      setConfirm("");
      pushToast("Пароль обновлён");
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      const message =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        (err as Error).message;
      if (status === 401) {
        setError(message || "Неверный текущий пароль");
      } else {
        setError(message || "Не удалось сменить пароль");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="profile-card">
      <header className="dash-section-head">
        <h2 style={{ fontSize: 24 }}>Безопасность</h2>
      </header>
      <p className="muted" style={{ fontSize: 13 }}>
        Минимальная длина пароля — 8 символов. Остальные сессии останутся
        активны до истечения refresh-токена.
      </p>
      <form className="profile-fields" onSubmit={handleSubmit} style={{ marginTop: 14 }}>
        <div className="field">
          <label>Текущий пароль</label>
          <input
            autoComplete="current-password"
            className="input"
            onChange={(e) => setCurrent(e.target.value)}
            type="password"
            value={current}
          />
        </div>
        <div className="field">
          <label>Новый пароль</label>
          <input
            autoComplete="new-password"
            className="input"
            onChange={(e) => setNext(e.target.value)}
            type="password"
            value={next}
          />
        </div>
        <div className="field">
          <label>Повторите новый пароль</label>
          <input
            autoComplete="new-password"
            className="input"
            onChange={(e) => setConfirm(e.target.value)}
            type="password"
            value={confirm}
          />
        </div>
        {error ? (
          <p
            className="mono"
            style={{ fontSize: 12, color: "oklch(0.50 0.18 25)" }}
          >
            {error}
          </p>
        ) : null}
        <div>
          <button
            className="btn btn--primary btn--sm"
            disabled={submitting || !current || !next}
            type="submit"
          >
            {submitting ? "Сохраняем…" : "Сменить пароль"}
          </button>
        </div>
      </form>
    </section>
  );
}

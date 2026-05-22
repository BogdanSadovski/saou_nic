import { useEffect, useMemo, useRef, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";

import { useUserStore } from "@/app/store";
import { RsIcon as Icon } from "@/shared/ui/realsync";

type Command = {
  id: string;
  title: string;
  hint: string;
  icon: Parameters<typeof Icon>[0]["name"];
  action: () => void;
};

type RailItem = {
  to: string;
  icon: string;
  title: string;
  sub: string;
  end?: boolean;
  adminOnly?: boolean;
};

// Полный список разделов рабочего пространства. `adminOnly: true`
// означает, что пункт показывается только пользователям с role=admin
// (см. фильтрацию ниже). Бэкенд всё равно enforces RBAC — это лишь
// чтобы обычные пользователи не видели пункт, который для них
// 403-ит.
const RAIL: RailItem[] = [
  { to: "/workspace", icon: "◎", title: "Обзор", sub: "Метрики и активность", end: true },
  { to: "/workspace/career", icon: "≡", title: "Карьерный центр", sub: "Карьерный профиль" },
  { to: "/workspace/profile", icon: "◐", title: "Профиль", sub: "Личные настройки" },
  { to: "/workspace/resume", icon: "▤", title: "Резюме", sub: "Резюме и инсайты" },
  { to: "/workspace/billing", icon: "▣", title: "Подписка", sub: "Тарифы и оплата" },
  { to: "/workspace/admin", icon: "◇", title: "Админ", sub: "Управление платформой", adminOnly: true },
];

export default function WorkspaceLayout() {
  const navigate = useNavigate();
  const userRole = useUserStore((s) => s.user.role);
  const isAdmin = userRole === "admin";
  const [cmdOpen, setCmdOpen] = useState(false);
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Отфильтрованный rail: admin-only пункты убираются из навигации
  // у обычных пользователей. На бэке RBAC всё равно работает (если
  // юзер угадает URL — получит 403), а здесь UX-only.
  const visibleRail = useMemo(
    () => RAIL.filter((item) => !item.adminOnly || isAdmin),
    [isAdmin],
  );

  const commands = useMemo<Command[]>(
    () => {
      const base: Command[] = [
        { id: "overview", title: "Перейти к обзору", hint: "Метрики, активность, рекомендации", icon: "grid", action: () => navigate("/workspace") },
        { id: "career", title: "Карьерный центр", hint: "Career-радар и модули", icon: "career", action: () => navigate("/workspace/career") },
        { id: "profile", title: "Профиль и настройки", hint: "Аватар, тема, GitHub, подписка", icon: "user", action: () => navigate("/workspace/profile") },
        { id: "resume", title: "Резюме и инсайты", hint: "Загрузка DOCX, анализ", icon: "file", action: () => navigate("/workspace/resume") },
        { id: "billing", title: "Подписка", hint: "Тарифы и история оплат", icon: "spark", action: () => navigate("/workspace/billing") },
      ];
      if (isAdmin) {
        base.push({ id: "admin", title: "Админ-панель", hint: "Пользователи, подписки, аудит", icon: "shield", action: () => navigate("/workspace/admin") });
      }
      base.push(
        { id: "interview", title: "Новое интервью", hint: "Запустить мок-сессию", icon: "play", action: () => navigate("/interview") },
        { id: "reports", title: "Все отчёты", hint: "История интервью + экспорт", icon: "chart", action: () => navigate("/reports") },
        { id: "home", title: "На главную", hint: "Лендинг RealSync", icon: "home", action: () => navigate("/") },
      );
      return base;
    },
    [navigate, isAdmin],
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return commands;
    return commands.filter((c) => `${c.title} ${c.hint}`.toLowerCase().includes(q));
  }, [commands, query]);

  // Cmd+K / Ctrl+K opens; Esc closes
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setCmdOpen((v) => !v);
        setQuery("");
        return;
      }
      if (e.key === "Escape" && cmdOpen) {
        e.preventDefault();
        setCmdOpen(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [cmdOpen]);

  useEffect(() => {
    if (cmdOpen) {
      const t = setTimeout(() => inputRef.current?.focus(), 30);
      return () => clearTimeout(t);
    }
  }, [cmdOpen]);

  const runCommand = (c: Command) => {
    c.action();
    setCmdOpen(false);
    setQuery("");
  };

  return (
    <main className="page" data-screen-label="Workspace">
      <div className="sysbar reveal" style={{ marginBottom: 24 }}>
        <span><span className="dot"></span><span className="k">рабочее пространство</span><span className="v">bsadovski.realsync</span></span>
        <span><span className="k">сессий сегодня</span><span className="v">2</span></span>
        <span><span className="k">последняя синхр.</span><span className="v">14:22 UTC</span></span>
        <span><span className="k">цп</span><span className="v">0.12</span></span>
        <span><span className="k">ws</span><span className="v">подключен</span></span>
      </div>

      <div className="dash-grid">
        <aside className="dash-rail">
          <div className="dash-rail-label">Рабочее пространство</div>
          {visibleRail.map((r) => (
            <NavLink
              key={r.to}
              to={r.to}
              end={r.end}
              className={({ isActive }) => `dash-rail-item ${isActive ? "is-active" : ""}`}
            >
              <div className="dash-rail-icon">{r.icon}</div>
              <div>
                <div className="dash-rail-title">{r.title}</div>
                <div className="dash-rail-sub">{r.sub}</div>
              </div>
              <div className="dash-rail-arrow"><Icon name="arrow" size={14} /></div>
            </NavLink>
          ))}
          <div className="hr"></div>
          <button
            className="btn btn--ghost btn--sm"
            onClick={() => setCmdOpen(true)}
            type="button"
            style={{ justifyContent: "space-between" }}
          >
            <span><Icon name="search" size={14} /> Команды</span>
            <span className="mono" style={{ fontSize: 11, color: "var(--muted)" }}>⌘K</span>
          </button>
          <button
            className="btn btn--primary btn--sm"
            onClick={() => navigate("/interview")}
            type="button"
            style={{ marginTop: 8 }}
          >
            <Icon name="plus" size={14} /> Новое интервью
          </button>
        </aside>

        <section>
          <Outlet />
        </section>
      </div>

      {cmdOpen ? (
        <div
          role="dialog"
          aria-modal="true"
          onClick={() => setCmdOpen(false)}
          style={{
            position: "fixed",
            inset: 0,
            background: "oklch(0 0 0 / 0.55)",
            backdropFilter: "blur(8px)",
            display: "grid",
            placeItems: "start center",
            paddingTop: "10vh",
            zIndex: 200,
          }}
        >
          <div
            onClick={(e) => e.stopPropagation()}
            className="card"
            style={{
              width: "min(640px, 92vw)",
              padding: 0,
              border: "1px solid var(--line)",
              background: "var(--paper)",
              borderRadius: "var(--r-2)",
              overflow: "hidden",
              boxShadow: "0 30px 80px -20px oklch(0 0 0 / 0.45)",
            }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "16px 18px", borderBottom: "1px solid var(--line)" }}>
              <Icon name="search" size={16} />
              <input
                ref={inputRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Найти команду…"
                style={{ flex: 1, background: "transparent", border: "none", outline: "none", fontSize: 16, color: "var(--ink)" }}
              />
              <span className="tag" style={{ fontSize: 10 }}>ESC</span>
            </div>
            <div style={{ maxHeight: "60vh", overflowY: "auto", padding: 6 }}>
              {filtered.length === 0 ? (
                <div className="muted" style={{ padding: 24, textAlign: "center", fontSize: 14 }}>
                  Команды не найдены
                </div>
              ) : (
                filtered.map((c) => (
                  <button
                    key={c.id}
                    type="button"
                    onClick={() => runCommand(c)}
                    style={{
                      width: "100%",
                      display: "grid",
                      gridTemplateColumns: "32px 1fr auto",
                      alignItems: "center",
                      gap: 12,
                      padding: "12px 14px",
                      borderRadius: "var(--r-1)",
                      background: "transparent",
                      border: "none",
                      cursor: "pointer",
                      color: "var(--ink)",
                      textAlign: "left",
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.background = "var(--paper-2)")}
                    onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
                  >
                    <div
                      style={{
                        width: 32,
                        height: 32,
                        borderRadius: 8,
                        background: "var(--paper-2)",
                        display: "grid",
                        placeItems: "center",
                        color: "var(--ink-2)",
                      }}
                    >
                      <Icon name={c.icon} size={14} />
                    </div>
                    <div>
                      <div style={{ fontSize: 14, fontWeight: 500 }}>{c.title}</div>
                      <div className="mono" style={{ fontSize: 11, color: "var(--muted)", marginTop: 2 }}>{c.hint}</div>
                    </div>
                    <Icon name="arrow" size={14} />
                  </button>
                ))
              )}
            </div>
            <div className="row-between mono" style={{ padding: "10px 16px", borderTop: "1px solid var(--line)", fontSize: 11, color: "var(--muted)" }}>
              <span>↑↓ навигация · ↵ выбрать</span>
              <span>⌘K — закрыть</span>
            </div>
          </div>
        </div>
      ) : null}
    </main>
  );
}

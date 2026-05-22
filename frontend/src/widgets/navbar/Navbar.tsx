import { NavLink, useNavigate } from "react-router-dom";

import { useAuthStore, useUIStore, useUserStore } from "@/app/store";
import { UserAvatar } from "@/shared/ui";

const NAV = [
  { to: "/", label: "Главная", end: true },
  { to: "/workspace", label: "Панель", end: true },
  { to: "/workspace/career", label: "Карьерный центр" },
  { to: "/interview", label: "Интервью" },
  { to: "/reports", label: "Отчёты" },
  { to: "/workspace/resume", label: "Резюме" },
];

// initialsOf удалён — теперь у всех юзеров единая ISO-куб иконка
// через <UserAvatar />. См. shared/ui/UserAvatar.tsx.

function ThemeToggle() {
  const resolved = useUIStore((s) => s.resolvedTheme);
  const toggle = useUIStore((s) => s.toggleTheme);
  const dark = resolved === "dark";
  return (
    <button
      className="theme-toggle"
      onClick={() => toggle()}
      title={dark ? "Светлая тема" : "Тёмная тема"}
      type="button"
    >
      <span className={`theme-icon ${dark ? "is-dark" : ""}`}>
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          {dark ? (
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          ) : (
            <>
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
            </>
          )}
        </svg>
      </span>
    </button>
  );
}

export function Navbar() {
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const user = useUserStore((s) => s.user);

  return (
    <header className="topbar">
      <button className="brand" onClick={() => navigate("/")} type="button">
        <span className="brand-dot"></span>
        Real<em>Sync</em>
        <span className="build-chip">v2.4.18</span>
      </button>
      <nav className="nav">
        {NAV.map((n) => (
          <NavLink
            key={n.to}
            to={n.to}
            end={n.end}
            className={({ isActive }) => `nav-link ${isActive ? "is-active" : ""}`}
          >
            {n.label}
          </NavLink>
        ))}
      </nav>
      <div className="topbar-right">
        <ThemeToggle />
        {isAuthenticated ? (
          <button
            className="user"
            onClick={() => navigate("/workspace/profile")}
            type="button"
            title={user.fullName || user.email || "Профиль"}
          >
            <UserAvatar size={32} className="avatar" alt={user.fullName || user.email} />
            <span className="user-meta">
              <span className="user-name">{user.fullName || user.email || "Профиль"}</span>
              <span className="user-role">{(user.role || "user").toUpperCase()}</span>
            </span>
          </button>
        ) : (
          <button className="user" onClick={() => navigate("/auth")} type="button">
            <UserAvatar size={32} className="avatar" alt="Гость" />
            <span className="user-meta">
              <span className="user-name">Войти</span>
              <span className="user-role">GUEST</span>
            </span>
          </button>
        )}
      </div>
    </header>
  );
}

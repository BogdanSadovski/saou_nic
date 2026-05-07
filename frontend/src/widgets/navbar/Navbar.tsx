import { Link, useLocation, useNavigate } from "react-router-dom";

import { useAuthStore, useUserStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { env } from "@/shared/config/env";
import { cn } from "@/shared/lib/cn";
import { Icon, type IconName } from "@/shared/ui";

type NavLink = {
  to: string;
  label: string;
  icon: IconName;
};

const initialsOf = (fullName: string, email: string): string => {
  const source = (fullName || email).trim();
  if (!source) return "·";
  const parts = source.split(/[\s@]+/).filter(Boolean);
  if (parts.length === 0) return "·";
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (parts[0]![0]! + parts[1]![0]!).toUpperCase();
};

/**
 * Top navigation pill.
 *
 * After redesign:
 *   - Theme toggle and Logout were removed; both moved to /profile.
 *   - Authenticated state shows a circular initials avatar that
 *     navigates to /profile (the new "user" entry point).
 *   - Each nav link gets an inline icon for quicker scanning.
 */
export function Navbar() {
  const location = useLocation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useUserStore((state) => state.user);
  const t = useTranslation();

  const mainLinks: NavLink[] = [
    { to: "/", label: t.home, icon: "home" },
    { to: "/dashboard", label: t.dashboard, icon: "dashboard" },
    { to: "/career-center", label: t.careerCenter, icon: "career" },
    { to: "/interview", label: t.interview, icon: "mic" },
    { to: "/reports", label: t.reports, icon: "report" },
  ];

  return (
    <header className="top-nav glass-card">
      <Link className="nav-brand" to="/">
        <span className="brand-dot" />
        <strong>{env.appName}</strong>
      </Link>

      <nav>
        {mainLinks.map((link) => (
          <Link
            className={cn(
              "nav-link",
              location.pathname === link.to && "nav-link-active",
            )}
            key={link.to}
            to={link.to}
          >
            <Icon name={link.icon} size={16} />
            <span>{link.label}</span>
          </Link>
        ))}
      </nav>

      <div className="nav-actions">
        {isAuthenticated ? (
          <button
            aria-label="Профиль"
            className={cn(
              "nav-avatar",
              location.pathname === "/profile" && "is-active",
            )}
            onClick={() => navigate("/profile")}
            title={user.fullName || user.email || "Профиль"}
            type="button"
          >
            <span className="nav-avatar-mark" aria-hidden="true">
              {initialsOf(user.fullName, user.email)}
            </span>
            <span className="nav-avatar-meta">
              <span className="nav-avatar-name">
                {user.fullName || user.email || "Профиль"}
              </span>
              <span className="nav-avatar-role">{user.role}</span>
            </span>
          </button>
        ) : (
          <Link className="nav-link nav-link-cta" to="/auth">
            <Icon name="user" size={16} />
            <span>{t.signIn}</span>
          </Link>
        )}
      </div>
    </header>
  );
}

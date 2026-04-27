import { Link, useLocation } from "react-router-dom";

import { useAuthStore, useUIStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { env } from "@/shared/config/env";
import { cn } from "@/shared/lib/cn";
import { GlassButton } from "@/shared/ui";

export function Navbar() {
  const location = useLocation();
  const theme = useUIStore((state) => state.theme);
  const toggleTheme = useUIStore((state) => state.toggleTheme);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const logout = useAuthStore((state) => state.logout);
  const t = useTranslation();

  const mainLinks = [
    { to: "/", label: t.home },
    { to: "/dashboard", label: t.dashboard },
    { to: "/career-center", label: t.careerCenter },
    { to: "/interview", label: t.interview },
    { to: "/reports", label: t.reports },
  ];

  return (
    <header className="top-nav glass-card">
      <div className="nav-brand">
        <span className="brand-dot" />
        <strong>{env.appName}</strong>
      </div>

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
            {link.label}
          </Link>
        ))}
      </nav>

      <div className="nav-actions">
        <GlassButton onClick={toggleTheme} type="button" variant="ghost">
          {theme === "light" ? t.dark : t.light}
        </GlassButton>
        {isAuthenticated ? (
          <GlassButton onClick={logout} type="button" variant="ghost">
            {t.logout}
          </GlassButton>
        ) : (
          <Link className="nav-link" to="/auth">
            {t.signIn}
          </Link>
        )}
      </div>
    </header>
  );
}

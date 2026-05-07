import { Link, useLocation } from "react-router-dom";

import { cn } from "@/shared/lib/cn";
import { Icon, type IconName } from "@/shared/ui";

const links: Array<{ to: string; label: string; icon: IconName }> = [
  { to: "/dashboard", label: "Дашборд", icon: "dashboard" },
  { to: "/interview", label: "Интервью", icon: "mic" },
  { to: "/reports", label: "Отчёты", icon: "report" },
  { to: "/profile", label: "Профиль", icon: "user" },
];

/**
 * Mobile bottom-nav. Shown only at <760px (driven by globals.css).
 * Each tap target stacks an icon over its label so the pill stays
 * legible without horizontal scroll.
 */
export function BottomNav() {
  const location = useLocation();

  return (
    <nav className="bottom-nav glass-card" aria-label="Мобильная навигация">
      {links.map((item) => (
        <Link
          aria-current={location.pathname === item.to ? "page" : undefined}
          className={cn(
            "bottom-link",
            location.pathname === item.to && "bottom-link-active",
          )}
          key={item.to}
          to={item.to}
        >
          <Icon name={item.icon} size={20} />
          <span>{item.label}</span>
        </Link>
      ))}
    </nav>
  );
}

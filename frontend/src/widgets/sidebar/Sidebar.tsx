import { Link, useLocation } from "react-router-dom";

import { useUserStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { cn } from "@/shared/lib/cn";
import { Icon, type IconName } from "@/shared/ui";

type SideEntry = {
  to: string;
  label: string;
  hint: string;
  icon: IconName;
  adminOnly?: boolean;
};

export function Sidebar() {
  const location = useLocation();
  const user = useUserStore((s) => s.user);
  const t = useTranslation();

  const entries: SideEntry[] = [
    { to: "/career-center", label: t.careerCenter, hint: "Карьерный профиль", icon: "career" },
    { to: "/profile", label: t.profile, hint: "Личные настройки", icon: "user" },
    { to: "/resume", label: t.resume, hint: "Резюме и инсайты", icon: "resume" },
    {
      to: "/admin",
      label: t.admin,
      hint: "Управление платформой",
      icon: "shield",
      adminOnly: true,
    },
  ];

  const visible = entries.filter((e) => !e.adminOnly || user.role === "admin");

  return (
    <aside className="side-panel">
      <h3 className="side-title">{t.workspace}</h3>
      <ul>
        {visible.map((entry, idx) => (
          <li key={entry.to} style={{ animationDelay: `${idx * 50}ms` }}>
            <Link
              className={cn(
                "side-link",
                location.pathname === entry.to && "side-link-active",
              )}
              to={entry.to}
            >
              <span className="side-link-icon">
                <Icon name={entry.icon} size={18} />
              </span>
              <span className="side-link-body">
                <span className="side-link-label">{entry.label}</span>
                <span className="side-link-hint">{entry.hint}</span>
              </span>
              <Icon className="side-link-arrow" name="chevron-right" size={16} />
            </Link>
          </li>
        ))}
      </ul>
    </aside>
  );
}

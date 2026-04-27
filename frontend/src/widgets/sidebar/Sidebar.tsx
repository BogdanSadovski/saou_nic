import { Link, useLocation } from "react-router-dom";

import { useTranslation } from "@/shared/i18n";
import { cn } from "@/shared/lib/cn";

export function Sidebar() {
  const location = useLocation();
  const t = useTranslation();

  const links = [
    { to: "/career-center", label: t.careerCenter },
    { to: "/profile", label: t.profile },
    { to: "/resume", label: t.resume },
    { to: "/admin", label: t.admin },
  ];

  return (
    <aside className="side-panel glass-card">
      <h3>{t.workspace}</h3>
      <ul>
        {links.map((link) => (
          <li key={link.to}>
            <Link
              className={cn(
                "side-link",
                location.pathname === link.to && "side-link-active",
              )}
              to={link.to}
            >
              {link.label}
            </Link>
          </li>
        ))}
      </ul>
    </aside>
  );
}

import { Link, useLocation } from "react-router-dom";

import { useTranslation } from "@/shared/i18n";
import { cn } from "@/shared/lib/cn";

export function BottomNav() {
  const location = useLocation();
  const t = useTranslation();

  const mobileLinks = [
    { to: "/dashboard", label: "Dash" },
    { to: "/interview", label: "Talk" },
    { to: "/profile", label: t.profile },
  ];

  return (
    <nav className="bottom-nav glass-card">
      {mobileLinks.map((item) => (
        <Link
          className={cn(
            "bottom-link",
            location.pathname === item.to && "bottom-link-active",
          )}
          key={item.to}
          to={item.to}
        >
          {item.label}
        </Link>
      ))}
    </nav>
  );
}

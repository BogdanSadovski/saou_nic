import { Outlet, useLocation } from "react-router-dom";

import { BottomNav } from "@/widgets/navbar/BottomNav";
import { Navbar } from "@/widgets/navbar/Navbar";
import { Sidebar } from "@/widgets/sidebar/Sidebar";

const compactRoutes = ["/", "/auth"];

export function AppShell() {
  const location = useLocation();
  const compact = compactRoutes.includes(location.pathname);
  const interviewImmersive = location.pathname.startsWith("/interview/session/");
  const hasHeader = !interviewImmersive;

  const mainClassName = [
    compact || interviewImmersive ? "layout-compact" : "layout-grid",
    hasHeader ? "with-header-offset" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className="app-root">
      <div className="ambient ambient-a" />
      <div className="ambient ambient-b" />

      {!interviewImmersive && <Navbar />}

      <main className={mainClassName}>
        {!compact && !interviewImmersive && <Sidebar />}
        <div className="route-shell page-transition">
          <Outlet />
        </div>
      </main>

      {!interviewImmersive && <BottomNav />}
    </div>
  );
}

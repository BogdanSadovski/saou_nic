import { Outlet, useLocation } from "react-router-dom";

import { BackendStatusBanner } from "@/shared/ui/BackendStatusBanner";
import { Navbar } from "@/widgets/navbar/Navbar";

export function AppShell() {
  const location = useLocation();
  const interviewImmersive = location.pathname.startsWith("/interview/session/");

  return (
    <div className="app">
      <BackendStatusBanner />
      {!interviewImmersive && <Navbar />}
      <Outlet />
    </div>
  );
}

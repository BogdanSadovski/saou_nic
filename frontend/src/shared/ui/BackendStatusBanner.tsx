import { useEffect, useState } from "react";

import { apiClient } from "@/shared/api/client";

type Status = "checking" | "online" | "offline";

/**
 * Tiny banner that pings the backend health endpoint periodically and
 * shows a clear "backend offline" notice when the user's API gateway is
 * unreachable. Prevents the user from staring at a "404"/"network error"
 * across every page and wondering whether it's their auth or their infra.
 */
export function BackendStatusBanner() {
  const [status, setStatus] = useState<Status>("checking");

  useEffect(() => {
    let cancelled = false;

    const ping = async () => {
      try {
        // Hit the gateway /health (proxied by Vite). Use a short timeout
        // so a stuck connection doesn't keep the banner in "checking".
        await apiClient.get("/health", { timeout: 4_000 });
        if (!cancelled) setStatus("online");
      } catch {
        if (!cancelled) setStatus("offline");
      }
    };

    void ping();
    const interval = window.setInterval(ping, 20_000);

    return () => {
      cancelled = true;
      window.clearInterval(interval);
    };
  }, []);

  if (status !== "offline") {
    return null;
  }

  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        position: "fixed",
        top: 12,
        left: "50%",
        transform: "translateX(-50%)",
        zIndex: 9999,
        background: "rgba(220, 53, 69, 0.95)",
        color: "white",
        padding: "10px 18px",
        borderRadius: 10,
        fontSize: 13,
        fontWeight: 500,
        boxShadow: "0 6px 20px rgba(0,0,0,0.25)",
        maxWidth: "90vw",
        textAlign: "center",
      }}
    >
      Бэкенд недоступен. Запустите api-gateway (порт 8000) — например, <code>make dev-up</code>.
      Часть страниц будет в офлайн-режиме.
    </div>
  );
}

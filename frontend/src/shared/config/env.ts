const toBool = (value: string | undefined, fallback = false): boolean => {
  if (!value) return fallback;
  return value === "true" || value === "1";
};

export const env = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "/api",
  apiWsUrl: import.meta.env.VITE_API_WS_URL ?? "ws://localhost:8000/ws",
  appName: import.meta.env.VITE_APP_NAME ?? "RealSync",
  appVersion: import.meta.env.VITE_APP_VERSION ?? "1.0.0",
  enableAnalytics: toBool(import.meta.env.VITE_ENABLE_ANALYTICS, false),
};

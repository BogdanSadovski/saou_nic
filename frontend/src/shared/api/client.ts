import axios, { AxiosError, AxiosHeaders } from "axios";

import { env } from "@/shared/config/env";

const MAX_RETRIES = 2;
const TOKEN_KEY = "realsync_token";
const REFRESH_TOKEN_KEY = "realsync_refresh_token";

// Statuses where a safe (idempotent) request is worth retrying. We
// intentionally do NOT retry 404 (route doesn't exist) or 401 (handled
// by the refresh flow) or 4xx generally — those are caller errors.
const RETRIABLE_STATUSES = new Set([502, 503, 504]);
// Network-level errors emitted by axios when the backend is unreachable.
const RETRIABLE_NETWORK_CODES = new Set([
  "ECONNABORTED",
  "ECONNRESET",
  "ECONNREFUSED",
  "ENETUNREACH",
  "ETIMEDOUT",
  "ERR_NETWORK",
]);

let refreshPromise: Promise<string | null> | null = null;

export const apiClient = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: 15_000,
});

const RETRIABLE_METHODS = new Set(["get", "head", "options"]);
const IS_DEV = (() => {
  try {
    return Boolean(import.meta.env?.DEV);
  } catch {
    return false;
  }
})();

const clearAuthTokens = () => {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
};

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY);
  const userID = localStorage.getItem("realsync_user_id");

  const headers = new AxiosHeaders(config.headers);
  // Skip Authorization header when token is empty/whitespace — sending
  // "Bearer " with no token causes some upstreams to return 400/404
  // instead of a clean 401, which masks the real auth state.
  if (token && token.trim()) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (userID) {
    headers.set("X-User-ID", userID);
  }

  return {
    ...config,
    headers,
  };
});

apiClient.interceptors.response.use(
  (response) => {
    if (IS_DEV) {
      const method = (response.config.method ?? "get").toUpperCase();
      const url = response.config.url ?? "";
      // Only log in dev to avoid leaking internal endpoint paths in prod.
      console.info("[api]", method, url, "->", response.status);
    }
    return response;
  },
  async (error: AxiosError) => {
    const config = error.config;
    if (!config) {
      return Promise.reject(error);
    }

    const status = error.response?.status;
    const requestUrl = config.url ?? "";
    const isRefreshCall = requestUrl.includes("/auth/refresh");

    // If our dev proxy synthesised a 503 with body {error:"backend_unreachable"}
    // (because api-gateway/upstream is down), normalise it into a synthetic
    // network error so callers can show "backend offline" instead of a
    // confusing 503-from-upstream message.
    const data = error.response?.data as { error?: string } | undefined;
    if (status === 503 && data?.error === "backend_unreachable") {
      // Mark as network error for downstream handlers
      (error as { code?: string }).code = "ERR_NETWORK";
    }

    if (status === 401 && !isRefreshCall && !(config as { __isRetryAfterRefresh?: boolean }).__isRetryAfterRefresh) {
      const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
      if (!refreshToken) {
        clearAuthTokens();
        return Promise.reject(error);
      }

      if (!refreshPromise) {
        refreshPromise = (async () => {
          try {
            const { data } = await apiClient.post<{
              access_token?: string;
              refresh_token?: string;
              accessToken?: string;
              refreshToken?: string;
            }>("/auth/refresh", {
              refresh_token: refreshToken,
            });

            const nextAccess = data.access_token ?? data.accessToken ?? "";
            const nextRefresh = data.refresh_token ?? data.refreshToken ?? "";

            if (!nextAccess) {
              clearAuthTokens();
              return null;
            }

            localStorage.setItem(TOKEN_KEY, nextAccess);
            if (nextRefresh) {
              localStorage.setItem(REFRESH_TOKEN_KEY, nextRefresh);
            }
            return nextAccess;
          } catch {
            clearAuthTokens();
            return null;
          } finally {
            refreshPromise = null;
          }
        })();
      }

      const nextToken = await refreshPromise;
      if (!nextToken) {
        return Promise.reject(error);
      }

      (config as { __isRetryAfterRefresh?: boolean }).__isRetryAfterRefresh = true;
      const headers = new AxiosHeaders(config.headers);
      headers.set("Authorization", `Bearer ${nextToken}`);
      config.headers = headers;
      return apiClient(config);
    }

    const method = (config.method ?? "get").toLowerCase();
    const isSafeMethod = RETRIABLE_METHODS.has(method);
    const isNetworkError =
      status === undefined &&
      (RETRIABLE_NETWORK_CODES.has(error.code ?? "") || error.message === "Network Error");
    // Auth endpoints are POST but their server-side handlers are
    // idempotent enough at the gateway/upstream-restart level: a 502/503/504
    // there is almost always an in-flight rolling restart, not "the user
    // already exists". Allow these specific paths to retry once on 5xx.
    const isAuthRetryable =
      method === "post" &&
      (requestUrl.includes("/auth/login") || requestUrl.includes("/auth/register"));
    const retryableOnStatus = status !== undefined && RETRIABLE_STATUSES.has(status);
    // Auth flows additionally retry 500 — user-service's
    // `Failed to initialize database` window after a postgres rolling
    // restart surfaces as a generic 500 from the gateway, but it heals
    // within a few seconds. Idempotent for register (unique email
    // constraint) and login (read-only).
    const isAuthRetryable500 = isAuthRetryable && status === 500;
    const retriable =
      (isSafeMethod && (isNetworkError || retryableOnStatus)) ||
      (isAuthRetryable && (isNetworkError || retryableOnStatus || isAuthRetryable500));

    const retries = (config as { __retryCount?: number }).__retryCount ?? 0;
    if (!retriable || retries >= MAX_RETRIES) {
      return Promise.reject(error);
    }

    (config as { __retryCount?: number }).__retryCount = retries + 1;
    // Exponential backoff with jitter: 250-500ms, 500-1000ms, ...
    const base = 250 * Math.pow(2, retries);
    const delay = base + Math.floor(Math.random() * base);
    await new Promise((resolve) => setTimeout(resolve, delay));

    return apiClient(config);
  },
);

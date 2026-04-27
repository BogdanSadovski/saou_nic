import axios, { AxiosError, AxiosHeaders } from "axios";

import { env } from "@/shared/config/env";

const MAX_RETRIES = 2;
const TOKEN_KEY = "realsync_token";
const REFRESH_TOKEN_KEY = "realsync_refresh_token";

let refreshPromise: Promise<string | null> | null = null;

export const apiClient = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: 10_000,
});

const RETRIABLE_METHODS = new Set(["get", "head", "options"]);

const clearAuthTokens = () => {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
};

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY);
  const userID = localStorage.getItem("realsync_user_id");

  const headers = new AxiosHeaders(config.headers);
  if (token) {
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
    const method = (response.config.method ?? "get").toUpperCase();
    const url = response.config.url ?? "";
    console.info("[api]", method, url, "->", response.status);
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
    const retriable = isSafeMethod && (status === undefined || status >= 500);

    const retries = (config as { __retryCount?: number }).__retryCount ?? 0;
    if (!retriable || retries >= MAX_RETRIES) {
      return Promise.reject(error);
    }

    (config as { __retryCount?: number }).__retryCount = retries + 1;
    await new Promise((resolve) => setTimeout(resolve, 250 * (retries + 1)));

    return apiClient(config);
  },
);

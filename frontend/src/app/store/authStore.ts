import { create } from "zustand";

import { authApi } from "@/shared/api";

type AuthState = {
  isAuthenticated: boolean;
  isInitialized: boolean;
  token: string | null;
  initialize: () => void;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, fullName: string) => Promise<void>;
  logout: () => void;
};

const TOKEN_KEY = "realsync_token";
const REFRESH_TOKEN_KEY = "realsync_refresh_token";

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isInitialized: false,
  token: null,
  initialize: () => {
    const token = localStorage.getItem(TOKEN_KEY);
    // A non-empty string in storage. Empty/whitespace-only token is treated
    // as unauthenticated and cleaned up to prevent inconsistent UI state.
    const hasToken = Boolean(token && token.trim());
    if (token && !hasToken) {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(REFRESH_TOKEN_KEY);
    }
    set({
      isAuthenticated: hasToken,
      isInitialized: true,
      token: hasToken ? token : null,
    });
  },
  login: async (email, password) => {
    const tokens = await authApi.login(email, password);
    if (!tokens.accessToken) {
      throw new Error("Не удалось войти: сервер вернул пустой токен");
    }
    localStorage.setItem(TOKEN_KEY, tokens.accessToken);
    if (tokens.refreshToken) {
      localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    }
    set({ isAuthenticated: true, token: tokens.accessToken, isInitialized: true });
  },
  register: async (email, password, fullName) => {
    const tokens = await authApi.register(email, password, fullName);
    if (!tokens.accessToken) {
      throw new Error("Не удалось зарегистрироваться: сервер вернул пустой токен");
    }
    localStorage.setItem(TOKEN_KEY, tokens.accessToken);
    if (tokens.refreshToken) {
      localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    }
    set({ isAuthenticated: true, token: tokens.accessToken, isInitialized: true });
  },
  logout: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem("realsync_user_id");
    set({ isAuthenticated: false, token: null, isInitialized: true });
  },
}));

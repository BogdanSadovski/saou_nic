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
    set({
      isAuthenticated: Boolean(token),
      isInitialized: true,
      token,
    });
  },
  login: async (email, password) => {
    const tokens = await authApi.login(email, password);
    localStorage.setItem(TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    set({ isAuthenticated: true, token: tokens.accessToken, isInitialized: true });
  },
  register: async (email, password, fullName) => {
    const tokens = await authApi.register(email, password, fullName);
    localStorage.setItem(TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    set({ isAuthenticated: true, token: tokens.accessToken, isInitialized: true });
  },
  logout: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem("realsync_user_id");
    set({ isAuthenticated: false, token: null, isInitialized: true });
  },
}));

import { create } from "zustand";

import type { User } from "@/entities/user/model/types";
import { userApi } from "@/shared/api";

type UserState = {
  user: User;
  hydrate: () => Promise<void>;
  updateProfile: (payload: Partial<User>) => Promise<void>;
};

const USER_KEY = "realsync_user";

const initialUser: User = {
  id: "u_1",
  fullName: "Кандидат",
  email: "candidate@example.com",
  role: "candidate",
  connectedGithub: false,
};

export const useUserStore = create<UserState>((set) => ({
  user: initialUser,
  hydrate: async () => {
    const token = localStorage.getItem("realsync_token");
    if (!token) return;

    try {
      const user = await userApi.getProfile();
      localStorage.setItem(USER_KEY, JSON.stringify(user));
      localStorage.setItem("realsync_user_id", user.id);
      set({ user });
      return;
    } catch {
      // Fallback to persisted state when backend is temporarily unavailable.
    }

    const raw = localStorage.getItem(USER_KEY);
    if (!raw) {
      return;
    }

    try {
      const persisted = JSON.parse(raw) as User;
      localStorage.setItem("realsync_user_id", persisted.id);
      set({ user: persisted });
    } catch {
      localStorage.removeItem(USER_KEY);
    }
  },
  updateProfile: async (payload) => {
    let nextUser: User | null = null;

    if (payload.fullName) {
      try {
        nextUser = await userApi.updateProfile({ fullName: payload.fullName });
      } catch {
        nextUser = null;
      }
    }

    set((state) => {
      const resolved = nextUser ?? {
        ...state.user,
        ...payload,
      };
      localStorage.setItem(USER_KEY, JSON.stringify(resolved));
      localStorage.setItem("realsync_user_id", resolved.id);
      return { user: resolved };
    });
  },
}));

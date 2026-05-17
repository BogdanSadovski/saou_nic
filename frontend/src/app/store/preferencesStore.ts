import { create } from "zustand";

/**
 * User-level preferences that don't (yet) live on the backend:
 * notifications, accessibility, sound. Persisted to localStorage so
 * they survive reloads.
 *
 * When server-side preferences land, this store should hydrate from
 * `/users/me` and write through with PATCH.
 */

const KEY = "realsync_preferences";

export type NotificationChannel = "interview_reminder" | "result_ready" | "weekly_digest";

export type Preferences = {
  notifications: Record<NotificationChannel, boolean>;
  /** Reduce motion (overrides @prefers-reduced-motion off→on). */
  reduceMotion: boolean;
  /** Sound feedback during interview (e.g. timer ticks, AI ready ping). */
  soundEnabled: boolean;
  /** Compact mode — denser tables and cards. */
  compactDensity: boolean;
};

const DEFAULTS: Preferences = {
  notifications: {
    interview_reminder: true,
    result_ready: true,
    weekly_digest: false,
  },
  reduceMotion: false,
  soundEnabled: true,
  compactDensity: false,
};

const load = (): Preferences => {
  if (typeof window === "undefined") return DEFAULTS;
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return DEFAULTS;
    const parsed = JSON.parse(raw) as Partial<Preferences>;
    return {
      ...DEFAULTS,
      ...parsed,
      notifications: { ...DEFAULTS.notifications, ...(parsed.notifications ?? {}) },
    };
  } catch {
    return DEFAULTS;
  }
};

type State = Preferences & {
  setNotification: (channel: NotificationChannel, value: boolean) => void;
  setReduceMotion: (value: boolean) => void;
  setSoundEnabled: (value: boolean) => void;
  setCompactDensity: (value: boolean) => void;
  reset: () => void;
};

const persist = (next: Preferences) => {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(KEY, JSON.stringify(next));
  }
};

export const usePreferencesStore = create<State>((set, get) => ({
  ...load(),
  setNotification: (channel, value) => {
    const next = { ...get().notifications, [channel]: value };
    persist({ ...get(), notifications: next });
    set({ notifications: next });
  },
  setReduceMotion: (reduceMotion) => {
    persist({ ...get(), reduceMotion });
    set({ reduceMotion });
  },
  setSoundEnabled: (soundEnabled) => {
    persist({ ...get(), soundEnabled });
    set({ soundEnabled });
  },
  setCompactDensity: (compactDensity) => {
    persist({ ...get(), compactDensity });
    set({ compactDensity });
  },
  reset: () => {
    persist(DEFAULTS);
    set(DEFAULTS);
  },
}));

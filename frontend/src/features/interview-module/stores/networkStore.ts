import { create } from "zustand";

type NetworkState = {
  wsConnected: boolean;
  reconnectAttempts: number;
  lastError: string | null;
  setConnected: (connected: boolean) => void;
  registerReconnectAttempt: () => void;
  setLastError: (error: string | null) => void;
  reset: () => void;
};

export const useNetworkStore = create<NetworkState>((set) => ({
  wsConnected: false,
  reconnectAttempts: 0,
  lastError: null,
  setConnected: (wsConnected) => set({ wsConnected }),
  registerReconnectAttempt: () =>
    set((state) => ({ reconnectAttempts: state.reconnectAttempts + 1 })),
  setLastError: (lastError) => set({ lastError }),
  reset: () => set({ wsConnected: false, reconnectAttempts: 0, lastError: null }),
}));

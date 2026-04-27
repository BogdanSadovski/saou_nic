import { create } from "zustand";

type TimerState = {
  elapsedSec: number;
  countdownSec: number;
  autoFinishTriggered: boolean;
  configure: (totalSec: number) => void;
  tick: () => void;
  triggerAutoFinish: () => void;
  reset: () => void;
};

const initialState = {
  elapsedSec: 0,
  countdownSec: 0,
  autoFinishTriggered: false,
};

export const useTimerStore = create<TimerState>((set, get) => ({
  ...initialState,
  configure: (totalSec) => set({ countdownSec: totalSec, elapsedSec: 0, autoFinishTriggered: false }),
  tick: () => {
    const current = get();
    if (current.countdownSec <= 0) {
      if (!current.autoFinishTriggered) {
        set({ autoFinishTriggered: true });
      }
      return;
    }

    const nextCountdown = current.countdownSec - 1;
    set({ elapsedSec: current.elapsedSec + 1, countdownSec: nextCountdown });
    if (nextCountdown <= 0) {
      set({ autoFinishTriggered: true });
    }
  },
  triggerAutoFinish: () => set({ autoFinishTriggered: true }),
  reset: () => set(initialState),
}));

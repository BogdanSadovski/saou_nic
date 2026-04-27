import { useEffect } from "react";

import { useInterviewStore } from "@/app/store";
import { useTranslation } from "@/shared/i18n";
import { GlassCard } from "@/shared/ui";

const formatTime = (total: number): string => {
  const min = Math.floor(total / 60);
  const sec = total % 60;
  return `${min.toString().padStart(2, "0")}:${sec.toString().padStart(2, "0")}`;
};

export function InterviewTimer() {
  const elapsedSec = useInterviewStore((state) => state.elapsedSec);
  const isPaused = useInterviewStore((state) => state.isPaused);
  const tick = useInterviewStore((state) => state.tick);
  const t = useTranslation();

  useEffect(() => {
    const timer = window.setInterval(() => tick(), 1000);
    return () => clearInterval(timer);
  }, [tick]);

  return (
    <GlassCard className="timer-card">
      <p className="muted">{t.liveInterviewTimer}</p>
      <h3>{formatTime(elapsedSec)}</h3>
      {isPaused && <p className="muted">{t.pauseButton}</p>}
    </GlassCard>
  );
}

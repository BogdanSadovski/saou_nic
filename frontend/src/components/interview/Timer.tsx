import React, { useState, useEffect, useCallback } from 'react';

interface TimerProps {
  durationSeconds: number;
  onEnd?: () => void;
  paused?: boolean;
}

const formatTime = (totalSeconds: number): string => {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
};

const Timer: React.FC<TimerProps> = ({ durationSeconds, onEnd, paused = false }) => {
  const [remaining, setRemaining] = useState(durationSeconds);
  const [isPaused, setIsPaused] = useState(paused);

  useEffect(() => {
    setIsPaused(paused);
  }, [paused]);

  const tick = useCallback(() => {
    setRemaining((prev) => {
      if (prev <= 1) {
        onEnd?.();
        return 0;
      }
      return prev - 1;
    });
  }, [onEnd]);

  useEffect(() => {
    if (isPaused || remaining <= 0) return;
    const interval = setInterval(tick, 1000);
    return () => clearInterval(interval);
  }, [isPaused, remaining, tick]);

  const handleTogglePause = () => {
    setIsPaused((prev) => !prev);
  };

  const handleReset = () => {
    setRemaining(durationSeconds);
    setIsPaused(false);
  };

  return (
    <div className="timer">
      <span className={`timer__display ${remaining < 60 ? 'timer__display--warning' : ''}`}>
        {formatTime(remaining)}
      </span>
      <div className="timer__controls">
        <button className="timer__btn" onClick={handleTogglePause}>
          {isPaused ? 'Resume' : 'Pause'}
        </button>
        <button className="timer__btn timer__btn--secondary" onClick={handleReset}>
          Reset
        </button>
      </div>
    </div>
  );
};

export default Timer;

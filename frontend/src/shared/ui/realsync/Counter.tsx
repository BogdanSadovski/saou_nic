import { useEffect, useState } from "react";

type CounterProps = {
  target: number;
  duration?: number;
  suffix?: string;
  decimals?: number;
};

export function Counter({ target, duration = 900, suffix = "", decimals = 0 }: CounterProps) {
  const [v, setV] = useState(0);
  useEffect(() => {
    let raf = 0;
    let start: number | undefined;
    const tick = (t: number) => {
      if (start === undefined) start = t;
      const p = Math.min((t - start) / duration, 1);
      const eased = 1 - Math.pow(1 - p, 3);
      setV(target * eased);
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);
  return (
    <>
      {v.toFixed(decimals)}
      {suffix}
    </>
  );
}

import { useEffect, useState } from "react";

type TrackProps = {
  value: number;
  animated?: boolean;
  lg?: boolean;
};

export function Track({ value, animated = true, lg = false }: TrackProps) {
  const [w, setW] = useState(animated ? 0 : value);
  useEffect(() => {
    if (!animated) return;
    const t = setTimeout(() => setW(value), 80);
    return () => clearTimeout(t);
  }, [value, animated]);
  return (
    <div className={`track ${lg ? "track--lg" : ""}`}>
      <div className="track-fill" style={{ width: `${Math.max(2, w)}%` }} />
    </div>
  );
}

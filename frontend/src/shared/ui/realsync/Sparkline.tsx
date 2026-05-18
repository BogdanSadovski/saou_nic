type SparklineProps = {
  data: number[];
  width?: number;
  height?: number;
  color?: string;
};

export function Sparkline({ data, width = 120, height = 32, color = "var(--ink)" }: SparklineProps) {
  if (!data || !data.length) return null;
  const max = Math.max(...data, 1);
  const min = Math.min(...data, 0);
  const span = max - min || 1;
  const step = width / (data.length - 1 || 1);
  const pts = data
    .map((v, i) => `${i * step},${height - ((v - min) / span) * (height - 4) - 2}`)
    .join(" ");
  return (
    <svg width={width} height={height} style={{ display: "block" }}>
      <polyline
        points={pts}
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle
        cx={(data.length - 1) * step}
        cy={height - ((data[data.length - 1]! - min) / span) * (height - 4) - 2}
        r="3"
        fill="var(--accent)"
        stroke={color}
        strokeWidth="1"
      />
    </svg>
  );
}

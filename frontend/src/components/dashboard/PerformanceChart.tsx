import React from 'react';

interface DataPoint {
  label: string;
  value: number;
}

interface PerformanceChartProps {
  data?: DataPoint[];
  title?: string;
  chartType?: 'bar' | 'line';
}

const defaultData: DataPoint[] = [
  { label: 'Mon', value: 65 },
  { label: 'Tue', value: 78 },
  { label: 'Wed', value: 52 },
  { label: 'Thu', value: 91 },
  { label: 'Fri', value: 84 },
  { label: 'Sat', value: 70 },
  { label: 'Sun', value: 60 },
];

const PerformanceChart: React.FC<PerformanceChartProps> = ({
  data = defaultData,
  title = 'Performance Overview',
  chartType = 'bar',
}) => {
  const maxValue = Math.max(...data.map((d) => d.value));

  return (
    <div className="performance-chart">
      <h3 className="performance-chart__title">{title}</h3>
      <div className={`performance-chart__chart performance-chart__chart--${chartType}`}>
        {chartType === 'bar'
          ? data.map((point) => (
              <div key={point.label} className="performance-chart__bar-wrapper">
                <div
                  className="performance-chart__bar"
                  style={{ height: `${(point.value / maxValue) * 100}%` }}
                />
                <span className="performance-chart__bar-label">{point.label}</span>
              </div>
            ))
          : data.map((point) => (
              <span key={point.label} className="performance-chart__point">
                {point.label}: {point.value}
              </span>
            ))}
      </div>
    </div>
  );
};

export default PerformanceChart;

import React from 'react';

interface StatsCardProps {
  title: string;
  value: string | number;
  icon?: React.ReactNode;
  trend?: { value: number; positive: boolean };
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger';
}

const StatsCard: React.FC<StatsCardProps> = ({
  title,
  value,
  icon,
  trend,
  variant = 'default',
}) => {
  return (
    <div className={`stats-card stats-card--${variant}`}>
      <div className="stats-card__header">
        <span className="stats-card__title">{title}</span>
        {icon && <span className="stats-card__icon">{icon}</span>}
      </div>
      <div className="stats-card__value">{value}</div>
      {trend && (
        <div
          className={`stats-card__trend ${trend.positive ? 'stats-card__trend--positive' : 'stats-card__trend--negative'}`}
        >
          {trend.positive ? '↑' : '↓'} {Math.abs(trend.value)}%
        </div>
      )}
    </div>
  );
};

export default StatsCard;

import React from 'react';

export interface StatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  progress?: {
    value: number;
    color?: string;
  };
  valueColor?: string;
}

export const StatsCard: React.FC<StatsCardProps> = ({
  title,
  value,
  subtitle,
  progress,
  valueColor,
}) => {
  return (
    <div className="card">
      <div className="card-body">
        <div className="d-flex align-items-center">
          <div className="subheader">{title}</div>
        </div>
        <div className={`h1 mb-3 ${valueColor || ''}`}>{value}</div>
        {subtitle && (
          <div className="d-flex mb-2">
            <div>{subtitle}</div>
          </div>
        )}
        {progress && (
          <div className="progress progress-sm">
            <div
              className={`progress-bar ${progress.color || 'bg-primary'}`}
              style={{ width: `${Math.min(100, Math.max(0, progress.value))}%` }}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default StatsCard;

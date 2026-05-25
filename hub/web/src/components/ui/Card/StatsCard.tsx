import React from 'react';
import { Card, Progress, Statistic, Typography } from 'antd';

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

const colorMap: Record<string, string> = {
  'bg-primary': '#2563eb',
  'bg-success': '#16a34a',
  'bg-warning': '#f59e0b',
  'bg-danger': '#dc2626',
  primary: '#2563eb',
  success: '#16a34a',
  warning: '#f59e0b',
  danger: '#dc2626',
};

export const StatsCard: React.FC<StatsCardProps> = ({
  title,
  value,
  subtitle,
  progress,
  valueColor,
}) => {
  return (
    <Card>
      <Statistic
        title={title}
        value={value}
        valueStyle={valueColor ? { color: colorMap[valueColor] || valueColor } : undefined}
      />
      {subtitle && (
        <Typography.Text type="secondary">
          {subtitle}
        </Typography.Text>
      )}
      {progress && (
        <Progress
          percent={Math.min(100, Math.max(0, progress.value))}
          strokeColor={colorMap[progress.color || 'primary'] || progress.color}
          showInfo={false}
          size="small"
          style={{ marginTop: 12 }}
        />
      )}
    </Card>
  );
};

export default StatsCard;

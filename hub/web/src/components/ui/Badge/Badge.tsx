import React from 'react';
import { Tag } from 'antd';

export type BadgeVariant = 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info';

export interface BadgeProps {
  variant?: BadgeVariant;
  children: React.ReactNode;
  icon?: React.ReactNode;
  className?: string;
}

const variantColors: Record<BadgeVariant, string> = {
  primary: 'blue',
  secondary: 'default',
  success: 'green',
  danger: 'red',
  warning: 'gold',
  info: 'cyan',
};

export const Badge: React.FC<BadgeProps> = ({
  variant = 'secondary',
  children,
  icon,
  className = '',
}) => {
  return (
    <Tag color={variantColors[variant]} icon={icon} className={className}>
      {children}
    </Tag>
  );
};

export default Badge;

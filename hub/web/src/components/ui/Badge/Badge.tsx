import React from 'react';

export type BadgeVariant = 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info';

export interface BadgeProps {
  variant?: BadgeVariant;
  children: React.ReactNode;
  icon?: React.ReactNode;
  className?: string;
}

const variantClasses: Record<BadgeVariant, string> = {
  primary: 'bg-primary',
  secondary: 'bg-secondary',
  success: 'bg-success',
  danger: 'bg-danger',
  warning: 'bg-warning',
  info: 'bg-info',
};

export const Badge: React.FC<BadgeProps> = ({
  variant = 'secondary',
  children,
  icon,
  className = '',
}) => {
  return (
    <span className={`badge ${variantClasses[variant]} text-white ${className}`}>
      {icon && <span className="me-1">{icon}</span>}
      {children}
    </span>
  );
};

export default Badge;

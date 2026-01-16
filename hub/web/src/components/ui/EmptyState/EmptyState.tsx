import React from 'react';
import { IconMoodEmpty } from '@tabler/icons-react';

export interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon,
  title,
  description,
  action,
}) => {
  return (
    <div className="card">
      <div className="card-body text-center py-5">
        <div className="mb-3 text-muted">
          {icon || <IconMoodEmpty size={48} />}
        </div>
        <h3 className="mb-2">{title}</h3>
        {description && <p className="text-muted mb-3">{description}</p>}
        {action && (
          <button className="btn btn-primary" onClick={action.onClick}>
            {action.label}
          </button>
        )}
      </div>
    </div>
  );
};

export default EmptyState;

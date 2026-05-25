import React from 'react';
import { Button, Card, Empty } from 'antd';

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
    <Card>
      <Empty
        image={icon || Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <span>
            <strong>{title}</strong>
            {description && <small style={{ display: 'block', marginTop: 4 }}>{description}</small>}
          </span>
        }
      >
        {action && (
          <Button type="primary" onClick={action.onClick}>
            {action.label}
          </Button>
        )}
      </Empty>
    </Card>
  );
};

export default EmptyState;

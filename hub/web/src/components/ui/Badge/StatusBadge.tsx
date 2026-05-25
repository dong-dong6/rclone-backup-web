import React from 'react';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  PlayCircleOutlined,
  WifiOutlined,
  DisconnectOutlined,
} from '@ant-design/icons';
import { Badge, BadgeVariant } from './Badge';

export type StatusType = 'online' | 'offline' | 'running' | 'running_task' | 'pending' | 'success' | 'failed';

export interface StatusBadgeProps {
  status: StatusType;
  showIcon?: boolean;
  size?: 'sm' | 'md';
  label?: string;
}

interface StatusConfig {
  variant: BadgeVariant;
  icon: React.ReactNode;
  defaultLabel: string;
}

const statusConfigs: Record<StatusType, StatusConfig> = {
  online: {
    variant: 'success',
    icon: <WifiOutlined />,
    defaultLabel: 'Online',
  },
  offline: {
    variant: 'secondary',
    icon: <DisconnectOutlined />,
    defaultLabel: 'Offline',
  },
  running: {
    variant: 'primary',
    icon: <LoadingOutlined spin />,
    defaultLabel: 'Running',
  },
  running_task: {
    variant: 'primary',
    icon: <PlayCircleOutlined />,
    defaultLabel: 'Running Task',
  },
  pending: {
    variant: 'warning',
    icon: <ClockCircleOutlined />,
    defaultLabel: 'Pending',
  },
  success: {
    variant: 'success',
    icon: <CheckCircleOutlined />,
    defaultLabel: 'Success',
  },
  failed: {
    variant: 'danger',
    icon: <CloseCircleOutlined />,
    defaultLabel: 'Failed',
  },
};

export const StatusBadge: React.FC<StatusBadgeProps> = ({
  status,
  showIcon = true,
  label,
}) => {
  const config = statusConfigs[status] || statusConfigs.pending;

  return (
    <Badge variant={config.variant} icon={showIcon ? config.icon : undefined}>
      {label || config.defaultLabel}
    </Badge>
  );
};

export default StatusBadge;

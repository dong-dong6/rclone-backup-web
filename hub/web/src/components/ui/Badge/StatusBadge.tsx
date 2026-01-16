import React from 'react';
import {
  IconCheck,
  IconX,
  IconRefresh,
  IconClockHour4,
  IconWifi,
  IconWifiOff,
  IconPlayerPlay,
} from '@tabler/icons-react';
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
    icon: <IconWifi size={14} />,
    defaultLabel: 'Online',
  },
  offline: {
    variant: 'secondary',
    icon: <IconWifiOff size={14} />,
    defaultLabel: 'Offline',
  },
  running: {
    variant: 'primary',
    icon: <IconRefresh size={14} className="spinner" />,
    defaultLabel: 'Running',
  },
  running_task: {
    variant: 'primary',
    icon: <IconPlayerPlay size={14} />,
    defaultLabel: 'Running Task',
  },
  pending: {
    variant: 'warning',
    icon: <IconClockHour4 size={14} />,
    defaultLabel: 'Pending',
  },
  success: {
    variant: 'success',
    icon: <IconCheck size={14} />,
    defaultLabel: 'Success',
  },
  failed: {
    variant: 'danger',
    icon: <IconX size={14} />,
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

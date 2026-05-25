import React from 'react';
import { Button, Card, Descriptions, Space, Typography } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  ApiOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { StatusBadge } from '../../../components/ui';
import type { Agent } from '../../../types';

export interface AgentCardProps {
  agent: Agent;
  onSync: (id: string) => void;
  onViewDetails: (agent: Agent) => void;
  onEdit: (agent: Agent) => void;
  onDelete: (id: string, name: string) => void;
}

export const AgentCard: React.FC<AgentCardProps> = ({
  agent,
  onSync,
  onViewDetails,
  onEdit,
  onDelete,
}) => {
  const { t } = useTranslation();

  const formatLastHeartbeat = (heartbeat: string | null) => {
    if (!heartbeat) return t('common.never');
    return new Date(heartbeat).toLocaleString();
  };

  return (
    <Card
      id={`agent-card-${agent.id}`}
      title={
          <Space>
          <ApiOutlined style={{ color: '#2563eb' }} />
          <span>{agent.name}</span>
        </Space>
      }
      extra={
        <StatusBadge
          status={agent.status}
          label={t(`common.${agent.status === 'running_task' ? 'running' : agent.status}`)}
        />
      }
      actions={[
        <Button
          type="text"
          icon={<ReloadOutlined />}
          onClick={() => onSync(agent.id)}
          disabled={agent.status === 'offline'}
          key="sync"
        >
          {t('agents.actions.sync')}
        </Button>,
        <Button type="text" icon={<EyeOutlined />} onClick={() => onViewDetails(agent)} key="details">
          {t('agents.actions.details')}
        </Button>,
        <Button type="text" icon={<EditOutlined />} onClick={() => onEdit(agent)} key="edit">
          {t('common.edit')}
        </Button>,
        !agent.is_local && (
          <Button
            type="text"
            danger
            icon={<DeleteOutlined />}
            onClick={() => onDelete(agent.id, agent.name)}
            key="delete"
          >
            {t('common.delete')}
          </Button>
        ),
      ].filter(Boolean)}
    >
      <Typography.Text type="secondary">{agent.id.substring(0, 8)}...</Typography.Text>
      <Descriptions column={1} size="small" style={{ marginTop: 16 }}>
        <Descriptions.Item label={t('agents.last_heartbeat')}>
          {formatLastHeartbeat(agent.last_heartbeat)}
        </Descriptions.Item>
        <Descriptions.Item label={t('agents.assigned_tasks')}>
          {agent.task_count || 0}
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
};

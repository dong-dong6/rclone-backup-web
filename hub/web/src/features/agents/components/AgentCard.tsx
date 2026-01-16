import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconServer,
  IconRefresh,
  IconChevronRight,
  IconEdit,
  IconTrash,
} from '@tabler/icons-react';
import classNames from 'classnames';
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
    <div
      id={`agent-card-${agent.id}`}
      className="col-md-6 col-lg-4"
    >
      <div className={classNames('card', (agent as any).className)}>
        <div className="card-body">
          {/* Agent Header */}
          <div className="d-flex align-items-center justify-content-between mb-3">
            <div className="d-flex align-items-center">
              <IconServer className="text-primary me-2" size={24} />
              <div>
                <h3 className="card-title mb-0">{agent.name}</h3>
                <small className="text-muted">{agent.id.substring(0, 8)}...</small>
              </div>
            </div>
            <StatusBadge
              status={agent.status}
              label={t(`common.${agent.status === 'running_task' ? 'running' : agent.status}`)}
            />
          </div>

          {/* Agent Details */}
          <div className="mb-3">
            <div className="row">
              <div className="col-6">
                <div className="text-muted small">{t('agents.last_heartbeat')}</div>
                <div className="fw-bold">{formatLastHeartbeat(agent.last_heartbeat)}</div>
              </div>
              <div className="col-6">
                <div className="text-muted small">{t('agents.assigned_tasks')}</div>
                <div className="fw-bold">{agent.task_count || 0}</div>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="d-flex gap-2 pt-2 border-top">
            <button
              onClick={() => onSync(agent.id)}
              className="btn btn-outline-primary btn-sm d-flex align-items-center gap-1"
              title={t('agents.actions.sync_config')}
              disabled={agent.status === 'offline'}
            >
              <IconRefresh size={14} />
              <span>{t('agents.actions.sync')}</span>
            </button>

            <button
              onClick={() => onViewDetails(agent)}
              className="btn btn-outline-secondary btn-sm d-flex align-items-center gap-1"
              title={t('agents.actions.view_details')}
            >
              <IconChevronRight size={14} />
              <span>{t('agents.actions.details')}</span>
            </button>

            <button
              onClick={() => onEdit(agent)}
              className="btn btn-outline-info btn-sm d-flex align-items-center gap-1"
              title={t('common.edit')}
            >
              <IconEdit size={14} />
              <span>{t('common.edit')}</span>
            </button>

            {!agent.is_local && (
              <button
                onClick={() => onDelete(agent.id, agent.name)}
                className="btn btn-outline-danger btn-sm d-flex align-items-center gap-1"
                title={t('agents.actions.delete')}
              >
                <IconTrash size={14} />
                <span>{t('common.delete')}</span>
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

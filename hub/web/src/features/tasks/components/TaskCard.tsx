import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconEdit,
  IconTrash,
  IconPlayerPlay,
  IconPlayerPause,
  IconCalendar,
  IconClock,
} from '@tabler/icons-react';
import type { BackupTask, Agent } from '../../../types';

export interface TaskCardProps {
  task: BackupTask;
  remoteName: string;
  agents: Agent[];
  onEdit: (task: BackupTask) => void;
  onDelete: (id: string) => void;
  onToggleActive: (task: BackupTask) => void;
  onTrigger: (taskId: string, agentId: string) => void;
  getAgentName: (id: string) => string;
  getAgentStatus: (id: string) => Agent['status'];
}

export const TaskCard: React.FC<TaskCardProps> = ({
  task,
  remoteName,
  agents,
  onEdit,
  onDelete,
  onToggleActive,
  onTrigger,
  getAgentName,
  getAgentStatus,
}) => {
  const { t } = useTranslation();

  const getSourceDisplay = () => {
    if (task.source_type === 'database') {
      const engine = (task.db_engine || '').toLowerCase();
      const dumpMode = (task.db_dump_mode || 'single').toLowerCase();
      if (engine === 'sqlite') {
        return `sqlite:${task.db_path || ''}`;
      }
      const host = task.db_host || '';
      const port = task.db_port ? `:${task.db_port}` : '';
      const name = dumpMode === 'all' ? '*' : task.db_name || '';
      return `${engine || 'db'}:${host}${port}/${name}`;
    }
    return task.source_path;
  };

  const getStatusMeta = (status: Agent['status']) => {
    switch (status) {
      case 'online':
        return { label: t('common.online'), badgeClass: 'bg-success' };
      case 'running_task':
        return { label: t('common.running'), badgeClass: 'bg-primary' };
      default:
        return { label: t('common.offline'), badgeClass: 'bg-secondary' };
    }
  };

  return (
    <div className="col-12 col-md-6 col-xl-4">
      <div className="card">
        <div className="card-header">
          <div>
            <h3 className="card-title mb-1">{task.name}</h3>
            <div className="text-muted small">{remoteName}</div>
          </div>
          <div className="ms-auto d-flex align-items-start flex-wrap gap-2">
            <span className={`badge ${task.is_active ? 'bg-success' : 'bg-secondary'} text-white`}>
              {task.is_active ? t('common.active') : t('common.inactive')}
            </span>
            <button
              onClick={() => onToggleActive(task)}
              className="btn btn-outline-secondary btn-sm"
              title={task.is_active ? t('tasks.deactivate') : t('tasks.activate')}
            >
              {task.is_active ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
            </button>
            <button
              onClick={() => onEdit(task)}
              className="btn btn-outline-primary btn-sm"
              title={t('common.edit')}
            >
              <IconEdit size={16} />
            </button>
            <button
              onClick={() => onDelete(task.id)}
              className="btn btn-outline-danger btn-sm"
              title={t('common.delete')}
            >
              <IconTrash size={16} />
            </button>
          </div>
        </div>

        <div className="card-body">
          <div className="mb-3">
            <div className="text-muted small mb-1">{t('tasks.source')}</div>
            <code className="text-break">{getSourceDisplay()}</code>
          </div>

          <div className="mb-3">
            <div className="text-muted small mb-1">{t('tasks.destination')}</div>
            <code className="text-break">{task.destination_path}</code>
          </div>

          <div className="row g-2 mb-3">
            <div className="col-12 col-sm-6">
              <div className="text-muted small mb-1">
                <IconCalendar size={14} className="me-1" />
                {t('tasks.list.columns.schedule')}
              </div>
              <code className="text-break">{task.schedule}</code>
            </div>
            <div className="col-12 col-sm-6">
              <div className="text-muted small mb-1">
                <IconClock size={14} className="me-1" />
                {t('tasks.next_run')}
              </div>
              <div className="fw-bold text-break">
                {task.next_run || t('tasks.calculating')}
              </div>
            </div>
          </div>

          <div className="mb-3">
            <div className="text-muted small mb-2">{t('tasks.assigned_agents')}</div>
            {task.assigned_agents.length > 0 ? (
              <div className="d-flex flex-column gap-2">
                {task.assigned_agents.map(agentId => {
                  const status = getAgentStatus(agentId);
                  const statusMeta = getStatusMeta(status);

                  return (
                    <div key={agentId} className="d-flex align-items-center justify-content-between gap-2">
                      <div className="text-break">{getAgentName(agentId)}</div>
                      <div className="d-flex align-items-center gap-2 flex-shrink-0">
                        <span className={`badge ${statusMeta.badgeClass} text-white`}>
                          {statusMeta.label}
                        </span>
                        {status === 'online' && (
                          <button
                            onClick={() => onTrigger(task.id, agentId)}
                            className="btn btn-outline-primary btn-sm"
                            title={t('tasks.run_now')}
                          >
                            <IconPlayerPlay size={16} />
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="text-muted">{t('tasks.no_agents_assigned')}</div>
            )}
          </div>

          {task.rclone_args.length > 0 && (
            <div>
              <div className="text-muted small mb-2">{t('tasks.arguments')}</div>
              <div className="d-flex flex-wrap gap-1">
                {task.rclone_args.map((arg, idx) => (
                  <span key={idx} className="badge bg-secondary text-white">
                    <code className="text-white">{arg}</code>
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

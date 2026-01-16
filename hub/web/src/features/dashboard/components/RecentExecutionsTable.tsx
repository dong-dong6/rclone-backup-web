import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconCheck, IconX, IconRefresh, IconClockHour4 } from '@tabler/icons-react';
import type { RecentExecution } from '../hooks';

export interface RecentExecutionsTableProps {
  executions: RecentExecution[];
}

export const RecentExecutionsTable: React.FC<RecentExecutionsTableProps> = ({
  executions,
}) => {
  const { t } = useTranslation();

  const getStatusTag = (status: string) => {
    const config: Record<string, { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <IconCheck size={16} /> },
      failed: { color: 'danger', icon: <IconX size={16} /> },
      running: { color: 'primary', icon: <IconRefresh size={16} className="spinner" /> },
      pending: { color: 'warning', icon: <IconClockHour4 size={16} /> },
    };

    const { color, icon } = config[status] || config.pending;
    return (
      <span className={`badge bg-${color} text-white`}>
        {icon}
        <span className="ms-1">{t(`executions.status.${status}`) || status.toUpperCase()}</span>
      </span>
    );
  };

  return (
    <div className="col-12">
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">{t('dashboard.recent_executions.title')}</h3>
        </div>
        <div className="card-body">
          <div className="table-responsive">
            <table className="table table-vcenter card-table">
              <thead>
                <tr>
                  <th>{t('executions.list.columns.task')}</th>
                  <th>{t('executions.list.columns.agent')}</th>
                  <th>{t('executions.list.columns.status')}</th>
                  <th>{t('executions.list.columns.startedAt')}</th>
                  <th>{t('executions.list.columns.duration')}</th>
                </tr>
              </thead>
              <tbody>
                {executions.length > 0 ? (
                  executions.map((execution) => (
                    <tr key={execution.id}>
                      <td>{execution.taskName}</td>
                      <td>{execution.agentName}</td>
                      <td>{getStatusTag(execution.status)}</td>
                      <td>
                        {execution.startedAt
                          ? new Date(execution.startedAt).toLocaleString()
                          : '-'}
                      </td>
                      <td>{Math.round(execution.duration)}s</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} className="text-center text-muted py-4">
                      {t('dashboard.recent_executions.no_executions')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

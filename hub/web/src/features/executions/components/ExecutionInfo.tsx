import React from 'react';
import { useTranslation } from 'react-i18next';
import { formatDuration, formatDate } from '../../../utils';
import type { TaskExecution, TriggerMode } from '../../../types';

export interface ExecutionInfoProps {
  execution: TaskExecution;
}

const getTriggerModeLabel = (mode: TriggerMode, t: (key: string) => string) => {
  switch (mode) {
    case 'manual':
      return t('executions.triggerMode.manual');
    case 'scheduled':
    case 'central':
      return t('executions.triggerMode.scheduled');
    case 'local_fallback':
      return t('executions.triggerMode.local_fallback');
    default:
      return mode;
  }
};

export const ExecutionInfo: React.FC<ExecutionInfoProps> = ({ execution }) => {
  const { t } = useTranslation();

  return (
    <div className="card">
      <div className="card-header">
        <h3 className="card-title">{t('executions.detail.info')}</h3>
      </div>
      <div className="card-body">
        <div className="row g-3">
          <div className="col-12 col-sm-6 col-lg-4">
            <div className="text-muted small">{t('executions.task')}</div>
            <div className="fw-bold text-break">{execution.task_name}</div>
          </div>
          <div className="col-12 col-sm-6 col-lg-4">
            <div className="text-muted small">{t('executions.agent')}</div>
            <div className="fw-bold text-break">{execution.agent_name}</div>
          </div>
          <div className="col-12 col-sm-6 col-lg-4">
            <div className="text-muted small">
              {t('executions.list.columns.triggerMode')}
            </div>
            <div className="fw-bold text-break">
              {getTriggerModeLabel(execution.trigger_mode, t)}
            </div>
          </div>
          <div className="col-12 col-sm-6 col-lg-4">
            <div className="text-muted small">{t('executions.duration')}</div>
            <div className="fw-bold">
              {execution.duration_seconds
                ? formatDuration(execution.duration_seconds)
                : t('executions.in_progress')}
            </div>
          </div>
          <div className="col-12 col-sm-6 col-lg-4">
            <div className="text-muted small">{t('executions.started_at')}</div>
            <div className="fw-bold">
              {execution.started_at ? formatDate(execution.started_at) : '-'}
            </div>
          </div>
          <div className="col-12 col-sm-6 col-lg-4">
            <div className="text-muted small">{t('executions.ended_at')}</div>
            <div className="fw-bold">
              {execution.ended_at ? formatDate(execution.ended_at) : '-'}
            </div>
          </div>
        </div>

        {execution.error_message && (
          <div className="alert alert-danger mt-3 mb-0" role="alert">
            <div className="fw-bold mb-1">{t('executions.error')}</div>
            <div className="text-break">{execution.error_message}</div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ExecutionInfo;

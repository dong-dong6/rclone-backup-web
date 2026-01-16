import React from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { IconChevronRight } from '@tabler/icons-react';
import { StatusBadge } from '../../../components/ui';
import { formatDuration, formatDate } from '../../../utils';
import type { TaskExecution, TriggerMode } from '../../../types';

export interface ExecutionTableProps {
  executions: TaskExecution[];
}

const getTriggerModeBadge = (mode: TriggerMode, t: (key: string) => string) => {
  const labels: Record<TriggerMode, string> = {
    manual: t('executions.triggerMode.manual'),
    scheduled: t('executions.triggerMode.scheduled'),
    central: t('executions.triggerMode.scheduled'),
    local_fallback: t('executions.triggerMode.local_fallback'),
  };

  const colors: Record<TriggerMode, { bg: string; text: string }> = {
    manual: { bg: 'bg-primary', text: 'text-white' },
    scheduled: { bg: 'bg-secondary', text: 'text-white' },
    central: { bg: 'bg-secondary', text: 'text-white' },
    local_fallback: { bg: 'bg-warning', text: 'text-dark' },
  };

  const { bg, text } = colors[mode] || colors.scheduled;
  return <span className={`badge ${bg} ${text}`}>{labels[mode] || mode}</span>;
};

export const ExecutionTable: React.FC<ExecutionTableProps> = ({ executions }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  if (executions.length === 0) {
    return (
      <tr>
        <td colSpan={7} className="text-center text-muted py-4">
          {t('executions.list.empty')}
        </td>
      </tr>
    );
  }

  return (
    <>
      {executions.map((execution) => (
        <tr
          key={execution.id}
          style={{ cursor: 'pointer' }}
          onClick={() => navigate(`/executions/${execution.id}`)}
        >
          <td>
            <StatusBadge
              status={execution.status}
              label={t(`executions.status.${execution.status}`)}
            />
          </td>
          <td className="text-break">{execution.task_name}</td>
          <td className="text-break">{execution.agent_name}</td>
          <td>{getTriggerModeBadge(execution.trigger_mode, t)}</td>
          <td className="text-muted">
            {execution.started_at ? formatDate(execution.started_at) : '-'}
          </td>
          <td className="text-muted">
            {execution.duration_seconds ? formatDuration(execution.duration_seconds) : '-'}
          </td>
          <td>
            <button
              onClick={(e) => {
                e.stopPropagation();
                navigate(`/executions/${execution.id}`);
              }}
              className="btn btn-outline-primary btn-sm"
              title={t('executions.view_details')}
            >
              <IconChevronRight size={16} />
            </button>
          </td>
        </tr>
      ))}
    </>
  );
};

export default ExecutionTable;

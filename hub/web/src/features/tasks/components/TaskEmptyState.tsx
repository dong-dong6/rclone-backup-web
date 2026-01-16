import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconPlus } from '@tabler/icons-react';

export interface TaskEmptyStateProps {
  onCreate: () => void;
}

export const TaskEmptyState: React.FC<TaskEmptyStateProps> = ({ onCreate }) => {
  const { t } = useTranslation();

  return (
    <div className="col-12">
      <div className="card">
        <div className="card-body text-center py-5">
          <p className="text-muted mb-3">{t('tasks.no_tasks')}</p>
          <button onClick={onCreate} className="btn btn-primary">
            <IconPlus size={16} />
            <span className="ms-1">{t('tasks.create_first')}</span>
          </button>
        </div>
      </div>
    </div>
  );
};

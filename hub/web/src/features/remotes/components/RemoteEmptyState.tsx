import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconPlus } from '@tabler/icons-react';

export interface RemoteEmptyStateProps {
  onCreate: () => void;
}

export const RemoteEmptyState: React.FC<RemoteEmptyStateProps> = ({ onCreate }) => {
  const { t } = useTranslation();

  return (
    <div className="col-12">
      <div className="card">
        <div className="card-body text-center py-5">
          <p className="text-muted mb-3">{t('remotes.list.empty')}</p>
          <button onClick={onCreate} className="btn btn-primary">
            <IconPlus size={16} />
            <span className="ms-1">{t('remotes.create.title')}</span>
          </button>
        </div>
      </div>
    </div>
  );
};

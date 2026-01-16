import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconEdit, IconPlugConnected, IconTrash, IconRefresh } from '@tabler/icons-react';
import type { RcloneRemote } from '../../../types';

export interface RemoteCardProps {
  remote: RcloneRemote;
  testing?: boolean;
  onTest: (id: string) => void;
  onEdit: (remote: RcloneRemote) => void;
  onDelete: (id: string) => void;
}

export const RemoteCard: React.FC<RemoteCardProps> = ({
  remote,
  testing,
  onTest,
  onEdit,
  onDelete,
}) => {
  const { t } = useTranslation();

  const getTestStatusBadge = () => {
    if (remote.last_test_success === true) {
      return <span className="badge bg-success text-white">{t('common.success')}</span>;
    }
    if (remote.last_test_success === false) {
      return <span className="badge bg-danger text-white">{t('common.failed')}</span>;
    }
    return <span className="badge bg-secondary text-white">{t('common.never')}</span>;
  };

  return (
    <div className="col-12 col-md-6 col-xl-4">
      <div className="card">
        <div className="card-header">
          <div>
            <h3 className="card-title mb-1">{remote.name}</h3>
            <div className="d-flex flex-wrap gap-1">
              {remote.type && (
                <span className="badge bg-secondary text-white">{remote.type}</span>
              )}
              <span title={remote.last_test_error || remote.last_test_message || ''}>
                {getTestStatusBadge()}
              </span>
            </div>
          </div>
          <div className="ms-auto d-flex gap-2">
            <button
              onClick={() => onTest(remote.id)}
              className="btn btn-outline-secondary btn-sm"
              title={t('remotes.actions.test')}
              disabled={testing}
            >
              {testing ? (
                <IconRefresh className="spinner" size={16} />
              ) : (
                <IconPlugConnected size={16} />
              )}
            </button>
            <button
              onClick={() => onEdit(remote)}
              className="btn btn-outline-primary btn-sm"
              title={t('common.edit')}
            >
              <IconEdit size={16} />
            </button>
            <button
              onClick={() => onDelete(remote.id)}
              className="btn btn-outline-danger btn-sm"
              title={t('common.delete')}
            >
              <IconTrash size={16} />
            </button>
          </div>
        </div>
        <div className="card-body">
          <div className="text-muted small">
            {t('remotes.list.columns.createdAt')}: {new Date(remote.created_at).toLocaleString()}
          </div>
          <div className="text-muted small">
            {t('remotes.list.columns.lastTest')}:{' '}
            {remote.last_test_at ? new Date(remote.last_test_at).toLocaleString() : t('common.never')}
          </div>
        </div>
      </div>
    </div>
  );
};

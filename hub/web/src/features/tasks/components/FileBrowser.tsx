import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconFolder, IconArrowUp, IconRefresh, IconAlertCircle } from '@tabler/icons-react';
import { Modal } from '../../../components/ui';
import type { FSListEntry } from '../../../types';

export interface FileBrowserProps {
  isOpen: boolean;
  path: string;
  parent: string;
  entries: FSListEntry[];
  loading: boolean;
  error: string | null;
  onPathChange: (path: string) => void;
  onNavigate: (path: string) => void;
  onClose: () => void;
  onSelect: () => void;
}

export const FileBrowser: React.FC<FileBrowserProps> = ({
  isOpen,
  path,
  parent,
  entries,
  loading,
  error,
  onPathChange,
  onNavigate,
  onClose,
  onSelect,
}) => {
  const { t } = useTranslation();

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      onNavigate(path);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t('tasks.browse.title')}
      size="lg"
      footer={
        <>
          <button className="btn btn-secondary" onClick={onClose} disabled={loading}>
            {t('common.cancel')}
          </button>
          <button className="btn btn-primary" onClick={onSelect} disabled={loading || !path.trim()}>
            {t('tasks.browse.select')}
          </button>
        </>
      }
    >
      <div className="mb-3">
        <label className="form-label">{t('tasks.browse.current_path')}</label>
        <div className="input-group">
          <input
            type="text"
            className="form-control font-monospace"
            value={path}
            onChange={(e) => onPathChange(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={loading}
          />
          <button
            className="btn btn-outline-secondary"
            onClick={() => onNavigate(path)}
            disabled={loading}
          >
            {loading ? (
              <IconRefresh className="spinner" size={16} />
            ) : (
              <IconRefresh size={16} />
            )}
          </button>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger d-flex align-items-center gap-2">
          <IconAlertCircle size={18} />
          {error}
        </div>
      )}

      <div className="border rounded" style={{ maxHeight: '400px', overflowY: 'auto' }}>
        {parent && (
          <div
            className="d-flex align-items-center gap-2 p-2 border-bottom cursor-pointer hover-bg-light"
            style={{ cursor: 'pointer' }}
            onClick={() => onNavigate(parent)}
          >
            <IconArrowUp size={16} className="text-muted" />
            <span className="text-muted">..</span>
          </div>
        )}

        {entries.length === 0 && !loading && !error && (
          <div className="p-3 text-center text-muted">
            {t('tasks.browse.empty')}
          </div>
        )}

        {entries.map((entry, idx) => (
          <div
            key={idx}
            className="d-flex align-items-center gap-2 p-2 border-bottom hover-bg-light"
            style={{ cursor: entry.is_dir ? 'pointer' : 'default' }}
            onClick={() => entry.is_dir && onNavigate(entry.path)}
          >
            <IconFolder
              size={16}
              className={entry.is_dir ? 'text-warning' : 'text-muted'}
            />
            <span className={entry.is_dir ? '' : 'text-muted'}>
              {entry.name}
              {entry.is_symlink && <span className="ms-1 text-muted">(link)</span>}
            </span>
          </div>
        ))}
      </div>
    </Modal>
  );
};

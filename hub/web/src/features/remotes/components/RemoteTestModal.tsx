import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconPlugConnected, IconRefresh } from '@tabler/icons-react';
import { Modal } from '../../../components/ui';
import type { RcloneRemote, RemoteTestResponse } from '../../../types';

export interface RemoteTestModalProps {
  isOpen: boolean;
  remote: RcloneRemote | null;
  testPath: string;
  submitting: boolean;
  result: RemoteTestResponse | null;
  onPathChange: (path: string) => void;
  onClose: () => void;
  onTest: () => void;
}

export const RemoteTestModal: React.FC<RemoteTestModalProps> = ({
  isOpen,
  remote,
  testPath,
  submitting,
  result,
  onPathChange,
  onClose,
  onTest,
}) => {
  const { t } = useTranslation();

  if (!remote) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`${t('remotes.actions.test')}: ${remote.name}`}
      footer={
        <>
          <button
            type="button"
            className="btn btn-outline-secondary"
            onClick={onClose}
            disabled={submitting}
          >
            {t('common.cancel')}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={onTest}
            disabled={submitting}
          >
            {submitting ? (
              <>
                <IconRefresh className="spinner" size={16} />
                <span className="ms-1">{t('remotes.test.running')}</span>
              </>
            ) : (
              <>
                <IconPlugConnected size={16} />
                <span className="ms-1">{t('remotes.actions.test')}</span>
              </>
            )}
          </button>
        </>
      }
    >
      {remote.type === 's3' && (
        <div className="mb-3">
          <label className="form-label">{t('remotes.test.pathLabel')}</label>
          <input
            type="text"
            className="form-control"
            value={testPath}
            onChange={(e) => onPathChange(e.target.value)}
            placeholder={t('remotes.test.pathPlaceholder')}
            disabled={submitting}
          />
          <div className="form-text">{t('remotes.test.pathPromptS3')}</div>
        </div>
      )}

      {result && (
        <div className={`alert ${result.success ? 'alert-success' : 'alert-danger'}`} role="alert">
          <div className="fw-semibold mb-1">
            {result.success ? t('common.success') : t('common.failed')}
            {typeof result.duration_ms === 'number' ? ` (${result.duration_ms}ms)` : ''}
          </div>
          {result.message && <div className="mb-2">{result.message}</div>}

          {result.error && (
            <div className="mb-2">
              <div className="fw-semibold">{t('common.error')}</div>
              <div className="font-monospace small">{result.error}</div>
            </div>
          )}

          {result.output && (
            <div>
              <div className="fw-semibold">{t('remotes.test.outputLabel')}</div>
              <textarea
                className="form-control font-monospace mt-1"
                rows={6}
                value={result.output}
                readOnly
              />
            </div>
          )}
        </div>
      )}
    </Modal>
  );
};

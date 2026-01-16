import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconRefresh } from '@tabler/icons-react';
import type { ExecutionStatus } from '../../../types';

export interface LogViewerProps {
  logs: string;
  status: ExecutionStatus;
  autoScroll: boolean;
  onAutoScrollChange: (value: boolean) => void;
  onDownload: () => void;
  containerRef: React.RefObject<HTMLDivElement>;
  endRef: React.RefObject<HTMLDivElement>;
  onScroll: () => void;
  id: string;
}

export const LogViewer: React.FC<LogViewerProps> = ({
  logs,
  status,
  autoScroll,
  onAutoScrollChange,
  onDownload,
  containerRef,
  endRef,
  onScroll,
  id,
}) => {
  const { t } = useTranslation();

  return (
    <div className="card">
      <div className="card-header">
        <h3 className="card-title">{t('executions.logs.title')}</h3>
        <div className="ms-auto d-flex align-items-center gap-3">
          <div className="form-check form-switch m-0">
            <input
              type="checkbox"
              className="form-check-input"
              id={`auto-scroll-${id}`}
              checked={autoScroll}
              onChange={(e) => onAutoScrollChange(e.target.checked)}
            />
            <label className="form-check-label" htmlFor={`auto-scroll-${id}`}>
              {t('executions.auto_scroll')}
            </label>
          </div>
          <button
            onClick={onDownload}
            className="btn btn-outline-primary btn-sm"
            disabled={!logs}
          >
            {t('common.download')}
          </button>
        </div>
      </div>

      <div className="card-body p-0">
        <div
          ref={containerRef}
          onScroll={onScroll}
          className="bg-dark text-white font-monospace p-3"
          style={{ height: '500px', maxHeight: '60vh', overflow: 'auto' }}
        >
          {logs ? (
            <>
              <pre className="mb-0" style={{ whiteSpace: 'pre-wrap' }}>
                {logs}
              </pre>
              <div ref={endRef} />
            </>
          ) : (
            <div className="text-secondary fst-italic">
              {status === 'pending'
                ? t('executions.waiting_to_start')
                : t('executions.no_logs')}
            </div>
          )}

          {status === 'running' && (
            <div className="mt-3 d-flex align-items-center gap-2">
              <IconRefresh className="spinner text-success" size={16} />
              <span className="text-success">{t('executions.running_live')}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LogViewer;

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { IconChevronLeft } from '@tabler/icons-react';
import { Loading, StatusBadge } from '../../components/ui';
import { useExecutionDetail } from './hooks';
import { ExecutionInfo, LogViewer } from './components';

export const ExecutionDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const {
    execution,
    logs,
    loading,
    loadError,
    autoScroll,
    setAutoScroll,
    downloadLogs,
    logsContainerRef,
    logsEndRef,
    handleScroll,
  } = useExecutionDetail(id);

  if (loading) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body">
              <Loading text={t('common.loading')} />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (loadError || !execution) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <p className="text-muted mb-3">{t('errors.notFound')}</p>
              <button
                className="btn btn-outline-secondary"
                onClick={() => navigate('/executions')}
              >
                <IconChevronLeft size={16} />
                <span className="ms-1">{t('common.back')}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="row row-deck row-cards">
      <div className="col-12">
        <div className="card">
          <div className="card-body d-flex align-items-center gap-2">
            <button
              className="btn btn-outline-secondary"
              onClick={() => navigate('/executions')}
            >
              <IconChevronLeft size={16} />
              <span className="ms-1">{t('common.back')}</span>
            </button>
            <div className="ms-auto">
              <StatusBadge
                status={execution.status}
                label={t(`executions.status.${execution.status}`)}
              />
            </div>
          </div>
        </div>
      </div>

      <div className="col-12">
        <ExecutionInfo execution={execution} />
      </div>

      <div className="col-12">
        <LogViewer
          logs={logs}
          status={execution.status}
          autoScroll={autoScroll}
          onAutoScrollChange={setAutoScroll}
          onDownload={downloadLogs}
          containerRef={logsContainerRef}
          endRef={logsEndRef}
          onScroll={handleScroll}
          id={id || ''}
        />
      </div>
    </div>
  );
};

export default ExecutionDetailPage;

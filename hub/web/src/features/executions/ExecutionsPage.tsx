import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconRefresh } from '@tabler/icons-react';
import { Loading } from '../../components/ui';
import { useExecutions } from './hooks';
import {
  ExecutionStatsBar,
  ExecutionTable,
  ExecutionFilter,
} from './components';

export const ExecutionsPage: React.FC = () => {
  const { t } = useTranslation();
  const {
    executions,
    stats,
    loading,
    page,
    totalPages,
    filter,
    setPage,
    setFilter,
    refresh,
  } = useExecutions();

  if (loading && executions.length === 0) {
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

  return (
    <div className="row row-deck row-cards">
      <ExecutionStatsBar stats={stats} />

      <div className="col-12">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">{t('executions.list.title')}</h3>
            <div className="ms-auto d-flex gap-2">
              <button
                onClick={refresh}
                className="btn btn-outline-primary btn-sm"
                disabled={loading}
              >
                <IconRefresh size={16} className={loading ? 'spinner' : undefined} />
                <span className="ms-1">{t('common.refresh')}</span>
              </button>
            </div>
          </div>

          <div className="card-body border-bottom py-3">
            <ExecutionFilter filter={filter} onFilterChange={setFilter} />
          </div>

          <div className="table-responsive">
            <table className="table table-vcenter card-table table-hover">
              <thead>
                <tr>
                  <th>{t('executions.list.columns.status')}</th>
                  <th>{t('executions.list.columns.task')}</th>
                  <th>{t('executions.list.columns.agent')}</th>
                  <th>{t('executions.list.columns.triggerMode')}</th>
                  <th>{t('executions.list.columns.startedAt')}</th>
                  <th>{t('executions.list.columns.duration')}</th>
                  <th className="w-1">{t('executions.list.columns.actions')}</th>
                </tr>
              </thead>
              <tbody>
                <ExecutionTable executions={executions} />
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="card-footer d-flex align-items-center">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="btn btn-outline-secondary btn-sm"
              >
                {t('common.previous')}
              </button>
              <div className="mx-auto text-muted">
                {t('common.page_of', { current: page, total: totalPages })}
              </div>
              <button
                onClick={() => setPage(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
                className="btn btn-outline-secondary btn-sm"
              >
                {t('common.next')}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ExecutionsPage;

import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconRefresh, IconPlus } from '@tabler/icons-react';
import type { AgentStats } from '../../../types';

export interface AgentStatsBarProps {
  stats: AgentStats;
  loading?: boolean;
  onRefresh: () => void;
  onRegister: () => void;
}

export const AgentStatsBar: React.FC<AgentStatsBarProps> = ({
  stats,
  loading,
  onRefresh,
  onRegister,
}) => {
  const { t } = useTranslation();

  return (
    <>
      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('agents.stats.total')}</div>
            </div>
            <div className="h1 mb-3">{stats.total}</div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('agents.stats.online')}</div>
            </div>
            <div className="h1 mb-3 text-success">{stats.online}</div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('agents.stats.running')}</div>
            </div>
            <div className="h1 mb-3 text-primary">{stats.running}</div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('common.actions')}</div>
            </div>
            <div className="d-flex gap-2">
              <button
                onClick={onRefresh}
                className="btn btn-outline-primary btn-sm"
                disabled={loading}
              >
                <IconRefresh size={16} className={loading ? 'spinner' : undefined} />
                {t('common.refresh')}
              </button>

              <button onClick={onRegister} className="btn btn-primary btn-sm">
                <IconPlus size={16} />
                {t('agents.register.new')}
              </button>
            </div>
          </div>
        </div>
      </div>
    </>
  );
};

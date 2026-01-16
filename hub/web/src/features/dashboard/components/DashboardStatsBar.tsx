import React from 'react';
import { useTranslation } from 'react-i18next';
import type { DashboardStats as DashboardStatsType } from '../hooks';

export interface DashboardStatsBarProps {
  stats: DashboardStatsType;
}

export const DashboardStatsBar: React.FC<DashboardStatsBarProps> = ({ stats }) => {
  const { t } = useTranslation();

  return (
    <>
      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('dashboard.stats.totalAgents')}</div>
            </div>
            <div className="h1 mb-3">{stats.totalAgents}</div>
            <div className="d-flex mb-2">
              <div>{t('dashboard.agents.online')}: {stats.onlineAgents}</div>
            </div>
            <div className="progress progress-sm">
              <div
                className="progress-bar bg-primary"
                style={{ width: `${stats.totalAgents > 0 ? (stats.onlineAgents / stats.totalAgents) * 100 : 0}%` }}
              />
            </div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('dashboard.stats.activeTasks')}</div>
            </div>
            <div className="h1 mb-3 text-success">{stats.activeTasks}</div>
            <div className="d-flex mb-2">
              <div>{t('dashboard.stats.totalTasks')}: {stats.totalTasks}</div>
            </div>
            <div className="progress progress-sm">
              <div
                className="progress-bar bg-success"
                style={{ width: `${stats.totalTasks > 0 ? (stats.activeTasks / stats.totalTasks) * 100 : 0}%` }}
              />
            </div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('dashboard.stats.recentExecutions')}</div>
            </div>
            <div className="h1 mb-3">{stats.recentExecutions}</div>
            <div className="d-flex mb-2">
              <div>{t('dashboard.time_range.24h')}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('dashboard.stats.successRate')}</div>
            </div>
            <div className="h1 mb-3">{stats.successRate.toFixed(1)}%</div>
            <div className="d-flex mb-2">
              <div className="progress progress-sm w-100">
                <div
                  className={`progress-bar ${
                    stats.successRate >= 90
                      ? 'bg-success'
                      : stats.successRate >= 70
                        ? 'bg-warning'
                        : 'bg-danger'
                  }`}
                  style={{ width: Math.round(stats.successRate) + '%' }}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
};

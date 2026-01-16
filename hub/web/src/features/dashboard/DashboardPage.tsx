import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { useDashboard } from './hooks';
import { DashboardStatsBar, RecentExecutionsTable } from './components';

export const DashboardPage: React.FC = () => {
  const { t } = useTranslation();
  const { stats, recentExecutions, backupTrend, loading } = useDashboard();

  const runningTasksPercent = stats.totalAgents > 0
    ? Math.round((stats.runningTasks / stats.totalAgents) * 100)
    : 0;
  const failedPercent = stats.recentExecutions > 0
    ? Math.round((stats.failedTasks24h / stats.recentExecutions) * 100)
    : 0;

  return (
    <div className="row row-deck row-cards">
      <DashboardStatsBar stats={stats} />

      {/* Charts Row */}
      <div className="col-12">
        <div className="row row-deck row-cards">
          <div className="col-12 col-lg-8">
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">
                  {t('dashboard.charts.execution_trend')} ({t('dashboard.time_range.24h')})
                </h3>
              </div>
              <div className="card-body">
                <div style={{ height: '300px' }}>
                  {backupTrend.length > 0 ? (
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={backupTrend}>
                        <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                        <XAxis dataKey="time" stroke="#6c757d" />
                        <YAxis stroke="#6c757d" />
                        <Tooltip />
                        <Area
                          type="monotone"
                          dataKey="success"
                          stackId="1"
                          stroke="#28a745"
                          fill="#28a745"
                          fillOpacity={0.6}
                          name={t('dashboard.charts.success')}
                        />
                        <Area
                          type="monotone"
                          dataKey="failed"
                          stackId="1"
                          stroke="#dc3545"
                          fill="#dc3545"
                          fillOpacity={0.6}
                          name={t('dashboard.charts.failed')}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  ) : (
                    <div className="d-flex align-items-center justify-content-center h-100 text-muted">
                      {t('dashboard.recent_executions.no_executions')}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div className="col-12 col-lg-4">
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">{t('dashboard.agents.availability')}</h3>
              </div>
              <div className="card-body">
                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">{t('dashboard.agents.online')}</span>
                    <span className="fw-bold">
                      {stats.onlineAgents}/{stats.totalAgents}
                    </span>
                  </div>
                  <div className="progress progress-sm">
                    <div
                      className="progress-bar bg-success"
                      style={{
                        width: `${
                          stats.totalAgents > 0
                            ? (stats.onlineAgents / stats.totalAgents) * 100
                            : 0
                        }%`,
                      }}
                    />
                  </div>
                </div>

                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">{t('dashboard.agents.running')}</span>
                    <span className="fw-bold">{runningTasksPercent}%</span>
                  </div>
                  <div className="progress progress-sm">
                    <div
                      className="progress-bar bg-primary"
                      style={{ width: `${runningTasksPercent}%` }}
                    />
                  </div>
                </div>

                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">
                      {t('dashboard.executions.failed')} ({t('dashboard.time_range.24h')})
                    </span>
                    <span className="fw-bold">{failedPercent}%</span>
                  </div>
                  <div className="progress progress-sm">
                    <div
                      className="progress-bar bg-danger"
                      style={{ width: `${failedPercent}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <RecentExecutionsTable executions={recentExecutions} />
    </div>
  );
};

export default DashboardPage;

import React from 'react';
import { Card, Empty, Progress, Space, Typography } from 'antd';
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
  const { stats, recentExecutions, backupTrend } = useDashboard();

  const runningTasksPercent = stats.totalAgents > 0
    ? Math.round((stats.runningTasks / stats.totalAgents) * 100)
    : 0;
  const failedPercent = stats.recentExecutions > 0
    ? Math.round((stats.failedTasks24h / stats.recentExecutions) * 100)
    : 0;
  const onlinePercent = stats.totalAgents > 0
    ? Math.round((stats.onlineAgents / stats.totalAgents) * 100)
    : 0;

  return (
    <div className="rbw-page">
      <DashboardStatsBar stats={stats} />

      <div className="rbw-grid-2">
        <Card title={`${t('dashboard.charts.execution_trend')} (${t('dashboard.time_range.24h')})`}>
          <div style={{ height: 300 }}>
            {backupTrend.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={backupTrend}>
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                  <XAxis dataKey="time" stroke="#6b7280" />
                  <YAxis stroke="#6b7280" />
                  <Tooltip />
                  <Area
                    type="monotone"
                    dataKey="success"
                    stackId="1"
                    stroke="#16a34a"
                    fill="#16a34a"
                    fillOpacity={0.55}
                    name={t('dashboard.charts.success')}
                  />
                  <Area
                    type="monotone"
                    dataKey="failed"
                    stackId="1"
                    stroke="#dc2626"
                    fill="#dc2626"
                    fillOpacity={0.55}
                    name={t('dashboard.charts.failed')}
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <Empty description={t('dashboard.recent_executions.no_executions')} />
            )}
          </div>
        </Card>

        <Card title={t('dashboard.agents.availability')}>
          <Space direction="vertical" size={18} style={{ width: '100%' }}>
            <div>
              <Typography.Text type="secondary">{t('dashboard.agents.online')}</Typography.Text>
              <Typography.Text strong style={{ float: 'right' }}>
                {stats.onlineAgents}/{stats.totalAgents}
              </Typography.Text>
              <Progress percent={onlinePercent} strokeColor="#16a34a" />
            </div>

            <div>
              <Typography.Text type="secondary">{t('dashboard.agents.running')}</Typography.Text>
              <Typography.Text strong style={{ float: 'right' }}>{runningTasksPercent}%</Typography.Text>
              <Progress percent={runningTasksPercent} strokeColor="#2563eb" />
            </div>

            <div>
              <Typography.Text type="secondary">
                {t('dashboard.executions.failed')} ({t('dashboard.time_range.24h')})
              </Typography.Text>
              <Typography.Text strong style={{ float: 'right' }}>{failedPercent}%</Typography.Text>
              <Progress percent={failedPercent} strokeColor="#dc2626" />
            </div>
          </Space>
        </Card>
      </div>

      <RecentExecutionsTable executions={recentExecutions} />
    </div>
  );
};

export default DashboardPage;

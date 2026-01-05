import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconServer,
  IconClock,
  IconCheck,
  IconX,
  IconRefresh,
  IconClockHour4,
} from '@tabler/icons-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { useSSE } from '../contexts/SSEContext';
import api from '../services/api';
import './Dashboard.css';

interface DashboardStats {
  totalAgents: number;
  onlineAgents: number;
  totalTasks: number;
  activeTasks: number;
  recentExecutions: number;
  successRate: number;
  runningTasks: number;
  failedTasks24h: number;
}

interface RecentExecution {
  id: string;
  taskName: string;
  agentName: string;
  status: string;
  startedAt: string;
  duration: number;
}

interface BackupTrendItem {
  time: string;
  success: number;
  failed: number;
}

const Dashboard: React.FC = () => {
  const { t } = useTranslation();
  const [stats, setStats] = useState<DashboardStats>({
    totalAgents: 0,
    onlineAgents: 0,
    totalTasks: 0,
    activeTasks: 0,
    recentExecutions: 0,
    successRate: 0,
    runningTasks: 0,
    failedTasks24h: 0,
  });
  const [recentExecutions, setRecentExecutions] = useState<RecentExecution[]>([]);
  const [backupTrend, setBackupTrend] = useState<BackupTrendItem[]>([]);
  const [loading, setLoading] = useState(true);
  const { events } = useSSE();

  useEffect(() => {
    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    // Handle SSE events
    if (events.length > 0) {
      const latestEvent = events[events.length - 1];
      if (latestEvent.type === 'agent.status.update' ||
        latestEvent.type === 'execution.status.update') {
        fetchDashboardData();
      }
    }
  }, [events]);

  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      const [agentsRes, tasksRes, executionsRes] = await Promise.all([
        api.get('/admin/agents'),
        api.get('/admin/tasks'),
        api.get('/admin/executions?limit=100'),
      ]);

      const agents = agentsRes.data || [];
      const tasks = tasksRes.data || [];
      const executions = executionsRes.data.items || [];

      const onlineAgents = agents.filter((a: any) => a.status === 'online').length;
      const runningTasks = agents.filter((a: any) => a.status === 'running_task').length;
      const activeTasks = tasks.filter((t: any) => t.is_active).length;

      // Calculate stats from real data
      const now = new Date();
      const oneDayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);

      const recentExecs = executions.filter((e: any) =>
        new Date(e.created_at || e.started_at) >= oneDayAgo
      );

      const successfulExecutions = recentExecs.filter((e: any) => e.status === 'success').length;
      const failedExecutions = recentExecs.filter((e: any) => e.status === 'failed').length;
      const successRate = recentExecs.length > 0
        ? (successfulExecutions / recentExecs.length) * 100
        : 0;

      setStats({
        totalAgents: agents.length,
        onlineAgents,
        totalTasks: tasks.length,
        activeTasks,
        recentExecutions: recentExecs.length,
        successRate,
        runningTasks,
        failedTasks24h: failedExecutions,
      });

      // Build backup trend from real execution data
      const trendData = buildBackupTrend(executions);
      setBackupTrend(trendData);

      setRecentExecutions(executions.slice(0, 5).map((e: any) => ({
        id: e.id,
        taskName: e.task_name || t('common.unknown'),
        agentName: e.agent_name || t('common.unknown'),
        status: e.status,
        startedAt: e.started_at,
        duration: e.duration_seconds || 0,
      })));
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const buildBackupTrend = (executions: any[]): BackupTrendItem[] => {
    if (!executions || executions.length === 0) {
      return [];
    }

    const now = new Date();
    const hourBuckets: { [key: string]: { success: number; failed: number } } = {};

    // Initialize buckets for last 24 hours (every 4 hours)
    for (let i = 5; i >= 0; i--) {
      const hour = new Date(now.getTime() - i * 4 * 60 * 60 * 1000);
      const key = hour.getHours().toString().padStart(2, '0') + ':00';
      hourBuckets[key] = { success: 0, failed: 0 };
    }

    // Aggregate execution data
    executions.forEach((e: any) => {
      const execTime = new Date(e.created_at || e.started_at);
      const hourKey = execTime.getHours().toString().padStart(2, '0') + ':00';

      // Find the nearest bucket
      const bucketKeys = Object.keys(hourBuckets);
      let nearestBucket = bucketKeys[0];
      for (const key of bucketKeys) {
        if (parseInt(hourKey) >= parseInt(key)) {
          nearestBucket = key;
        }
      }

      if (hourBuckets[nearestBucket]) {
        if (e.status === 'success') {
          hourBuckets[nearestBucket].success++;
        } else if (e.status === 'failed') {
          hourBuckets[nearestBucket].failed++;
        }
      }
    });

    return Object.entries(hourBuckets).map(([time, data]) => ({
      time,
      success: data.success,
      failed: data.failed,
    }));
  };

  const getStatusTag = (status: string) => {
    const config: Record<string, { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <IconCheck size={16} /> },
      failed: { color: 'danger', icon: <IconX size={16} /> },
      running: { color: 'primary', icon: <IconRefresh size={16} className="spinner" /> },
      pending: { color: 'warning', icon: <IconClockHour4 size={16} /> },
    };

    const { color, icon } = config[status] || config.pending;
    return (
      <span className={`badge bg-${color} text-white`}>
        {icon}
        <span className="ms-1">{t(`executions.status.${status}`) || status.toUpperCase()}</span>
      </span>
    );
  };

  // Calculate percentages for display
  const runningTasksPercent = stats.totalAgents > 0
    ? Math.round((stats.runningTasks / stats.totalAgents) * 100)
    : 0;
  const failedPercent = stats.recentExecutions > 0
    ? Math.round((stats.failedTasks24h / stats.recentExecutions) * 100)
    : 0;

  return (
    <div className="row row-deck row-cards">
      {/* Stats Cards */}
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
                style={{ width: stats.totalAgents > 0 ? (stats.onlineAgents / stats.totalAgents) * 100 : 0 + '%' }}
              ></div>
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
                style={{ width: stats.totalTasks > 0 ? (stats.activeTasks / stats.totalTasks) * 100 : 0 + '%' }}
              ></div>
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
                  className={`progress-bar ${stats.successRate >= 90 ? 'bg-success' : stats.successRate >= 70 ? 'bg-warning' : 'bg-danger'}`}
                  style={{ width: Math.round(stats.successRate) + '%' }}
                ></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Charts Row */}
      <div className="col-12">
        <div className="row row-deck row-cards">
          <div className="col-12 col-lg-8">
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">{t('dashboard.charts.execution_trend')} ({t('dashboard.time_range.24h')})</h3>
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
                        <Area type="monotone" dataKey="success" stackId="1" stroke="#28a745" fill="#28a745" fillOpacity={0.6} name={t('dashboard.charts.success')} />
                        <Area type="monotone" dataKey="failed" stackId="1" stroke="#dc3545" fill="#dc3545" fillOpacity={0.6} name={t('dashboard.charts.failed')} />
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
                    <span className="fw-bold">{stats.onlineAgents}/{stats.totalAgents}</span>
                  </div>
                  <div className="progress progress-sm">
                    <div
                      className="progress-bar bg-success"
                      style={{ width: stats.totalAgents > 0 ? (stats.onlineAgents / stats.totalAgents) * 100 : 0 + '%' }}
                    ></div>
                  </div>
                </div>

                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">{t('dashboard.agents.running')}</span>
                    <span className="fw-bold">{runningTasksPercent}%</span>
                  </div>
                  <div className="progress progress-sm">
                    <div className="progress-bar bg-primary" style={{ width: `${runningTasksPercent}%` }}></div>
                  </div>
                </div>

                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">{t('dashboard.executions.failed')} ({t('dashboard.time_range.24h')})</span>
                    <span className="fw-bold">{failedPercent}%</span>
                  </div>
                  <div className="progress progress-sm">
                    <div className="progress-bar bg-danger" style={{ width: `${failedPercent}%` }}></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Executions Table */}
      <div className="col-12">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">{t('dashboard.recent_executions.title')}</h3>
          </div>
          <div className="card-body">
            <div className="table-responsive">
              <table className="table table-vcenter card-table">
                <thead>
                  <tr>
                    <th>{t('executions.list.columns.task')}</th>
                    <th>{t('executions.list.columns.agent')}</th>
                    <th>{t('executions.list.columns.status')}</th>
                    <th>{t('executions.list.columns.startedAt')}</th>
                    <th>{t('executions.list.columns.duration')}</th>
                  </tr>
                </thead>
                <tbody>
                  {recentExecutions.length > 0 ? (
                    recentExecutions.map((execution) => (
                      <tr key={execution.id}>
                        <td>{execution.taskName}</td>
                        <td>{execution.agentName}</td>
                        <td>{getStatusTag(execution.status)}</td>
                        <td>{execution.startedAt ? new Date(execution.startedAt).toLocaleString() : '-'}</td>
                        <td>{Math.round(execution.duration)}s</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={5} className="text-center text-muted py-4">
                        {t('dashboard.recent_executions.no_executions')}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
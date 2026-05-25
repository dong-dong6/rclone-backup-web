import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useSSE } from '../../../contexts/SSEContext';
import { agentsApi, tasksApi, executionsApi } from '../../../services';

export interface DashboardStats {
  totalAgents: number;
  onlineAgents: number;
  totalTasks: number;
  activeTasks: number;
  recentExecutions: number;
  successRate: number;
  runningTasks: number;
  failedTasks24h: number;
}

export interface RecentExecution {
  id: string;
  taskName: string;
  agentName: string;
  status: string;
  startedAt: string;
  duration: number;
}

export interface BackupTrendItem {
  time: string;
  success: number;
  failed: number;
}

const buildBackupTrend = (executions: any[]): BackupTrendItem[] => {
  if (!executions || executions.length === 0) {
    return [];
  }

  const now = new Date();
  const hourBuckets: { [key: string]: { success: number; failed: number } } = {};

  for (let i = 5; i >= 0; i--) {
    const hour = new Date(now.getTime() - i * 4 * 60 * 60 * 1000);
    const key = hour.getHours().toString().padStart(2, '0') + ':00';
    hourBuckets[key] = { success: 0, failed: 0 };
  }

  executions.forEach((e: any) => {
    const execTime = new Date(e.created_at || e.started_at);
    const hourKey = execTime.getHours().toString().padStart(2, '0') + ':00';

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

export function useDashboard() {
  const { t } = useTranslation();
  const { events } = useSSE();
  const [loading, setLoading] = useState(true);
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

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      const [agents, tasks, executionsResponse] = await Promise.all([
        agentsApi.getAll(),
        tasksApi.getAll(),
        executionsApi.getAll({ limit: 100 }),
      ]);

      const executions = executionsResponse.items || [];

      const onlineAgents = agents.filter((a: any) => a.status === 'online').length;
      const runningTasks = agents.filter((a: any) => a.status === 'running_task').length;
      const activeTasks = tasks.filter((t: any) => t.is_active).length;

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

      setBackupTrend(buildBackupTrend(executions));

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
  }, [t]);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  useEffect(() => {
    if (events.length > 0) {
      const latestEvent = events[events.length - 1];
      if (
        latestEvent.type === 'agent.status.update' ||
        latestEvent.type === 'execution.status.update'
      ) {
        fetchData();
      }
    }
  }, [events, fetchData]);

  return {
    stats,
    recentExecutions,
    backupTrend,
    loading,
    refresh: fetchData,
  };
}

import { useState, useEffect, useCallback } from 'react';
import { executionsApi } from '../../../services';
import type { TaskExecution, ExecutionStats } from '../../../types';

export interface ExecutionFilter {
  status: string;
  taskId: string;
  agentId: string;
}

export interface UseExecutionsReturn {
  executions: TaskExecution[];
  stats: ExecutionStats | null;
  loading: boolean;
  page: number;
  totalPages: number;
  filter: ExecutionFilter;
  setPage: (page: number) => void;
  setFilter: (filter: Partial<ExecutionFilter>) => void;
  refresh: () => Promise<void>;
}

export function useExecutions(): UseExecutionsReturn {
  const [executions, setExecutions] = useState<TaskExecution[]>([]);
  const [stats, setStats] = useState<ExecutionStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [filter, setFilterState] = useState<ExecutionFilter>({
    status: '',
    taskId: '',
    agentId: '',
  });

  const fetchExecutions = useCallback(async () => {
    setLoading(true);
    try {
      const response = await executionsApi.getAll({
        page,
        limit: 20,
        ...(filter.status && { status: filter.status }),
        ...(filter.taskId && { task_id: filter.taskId }),
        ...(filter.agentId && { agent_id: filter.agentId }),
      });

      const data = response.data;
      const items = Array.isArray(data?.items)
        ? data.items
        : Array.isArray((data as any)?.executions)
          ? (data as any).executions
          : [];

      setExecutions(items);
      setTotalPages(Number.isFinite(data?.total_pages) ? data.total_pages : 1);
    } catch (error) {
      console.error('Failed to fetch executions:', error);
      setExecutions([]);
      setTotalPages(1);
    } finally {
      setLoading(false);
    }
  }, [page, filter]);

  const fetchStats = useCallback(async () => {
    try {
      const response = await executionsApi.getStats();
      setStats(response.data);
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  }, []);

  const refresh = useCallback(async () => {
    await Promise.all([fetchExecutions(), fetchStats()]);
  }, [fetchExecutions, fetchStats]);

  useEffect(() => {
    fetchExecutions();
    fetchStats();
  }, [fetchExecutions, fetchStats]);

  const setFilter = useCallback((newFilter: Partial<ExecutionFilter>) => {
    setPage(1);
    setFilterState((prev) => ({ ...prev, ...newFilter }));
  }, []);

  return {
    executions,
    stats,
    loading,
    page,
    totalPages,
    filter,
    setPage,
    setFilter,
    refresh,
  };
}

export default useExecutions;

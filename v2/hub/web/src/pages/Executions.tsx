import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Play, CheckCircle, XCircle, Clock, Terminal, 
  Download, RefreshCw, ChevronRight, Filter,
  Calendar, User, Server, AlertCircle
} from 'lucide-react';
import { apiClient } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { useSSE } from '../contexts/SSEContext';
import classNames from 'classnames';

interface TaskExecution {
  id: string;
  task_id: string;
  task_name: string;
  agent_id: string;
  agent_name: string;
  status: 'pending' | 'running' | 'success' | 'failed';
  trigger_mode: 'manual' | 'central' | 'local_fallback';
  log_output?: string;
  error_message?: string;
  started_at?: string;
  ended_at?: string;
  created_at: string;
  duration?: number;
}

interface ExecutionStats {
  total: number;
  success: number;
  failed: number;
  running: number;
  success_rate_24h: number;
  avg_duration_seconds: number;
}

const ExecutionList: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const navigate = useNavigate();
  const [executions, setExecutions] = useState<TaskExecution[]>([]);
  const [stats, setStats] = useState<ExecutionStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [filter, setFilter] = useState({
    status: '',
    taskId: '',
    agentId: '',
  });

  useEffect(() => {
    fetchExecutions();
    fetchStats();
  }, [page, filter]);

  const fetchExecutions = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: '20',
        ...(filter.status && { status: filter.status }),
        ...(filter.taskId && { task_id: filter.taskId }),
        ...(filter.agentId && { agent_id: filter.agentId }),
      });

      const response = await apiClient.get(`/admin/executions?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      setExecutions(response.data.executions);
      setTotalPages(response.data.total_pages);
    } catch (error) {
      console.error('Failed to fetch executions:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const response = await apiClient.get('/admin/executions/stats', {
        headers: { Authorization: `Bearer ${token}` },
      });
      setStats(response.data);
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="text-green-500" size={20} />;
      case 'failed':
        return <XCircle className="text-red-500" size={20} />;
      case 'running':
        return <RefreshCw className="text-blue-500 animate-spin" size={20} />;
      case 'pending':
        return <Clock className="text-yellow-500" size={20} />;
      default:
        return <AlertCircle className="text-gray-500" size={20} />;
    }
  };

  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${Math.round(seconds)}s`;
    if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
    return `${Math.round(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  const getTriggerModeLabel = (mode: string) => {
    switch (mode) {
      case 'manual':
        return t('executions.trigger.manual');
      case 'central':
        return t('executions.trigger.scheduled');
      case 'local_fallback':
        return t('executions.trigger.fallback');
      default:
        return mode;
    }
  };

  if (loading && executions.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="neu-card p-8">
          <div className="animate-spin rounded-full h-12 w-12 border-4 border-primary border-t-transparent"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with Stats */}
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">{t('executions.title')}</h1>
        <button
          onClick={() => {
            fetchExecutions();
            fetchStats();
          }}
          className="neu-button flex items-center space-x-2"
        >
          <RefreshCw size={16} />
          <span>{t('common.refresh')}</span>
        </button>
      </div>

      {/* Statistics Cards */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
          <div className="neu-card p-4">
            <div className="text-2xl font-bold">{stats.total}</div>
            <div className="text-sm text-gray-500">{t('executions.stats.total')}</div>
          </div>
          <div className="neu-card p-4">
            <div className="text-2xl font-bold text-green-500">{stats.success}</div>
            <div className="text-sm text-gray-500">{t('executions.stats.success')}</div>
          </div>
          <div className="neu-card p-4">
            <div className="text-2xl font-bold text-red-500">{stats.failed}</div>
            <div className="text-sm text-gray-500">{t('executions.stats.failed')}</div>
          </div>
          <div className="neu-card p-4">
            <div className="text-2xl font-bold text-blue-500">{stats.running}</div>
            <div className="text-sm text-gray-500">{t('executions.stats.running')}</div>
          </div>
          <div className="neu-card p-4">
            <div className="text-2xl font-bold">{stats.success_rate_24h.toFixed(1)}%</div>
            <div className="text-sm text-gray-500">{t('executions.stats.success_rate')}</div>
          </div>
          <div className="neu-card p-4">
            <div className="text-2xl font-bold">{formatDuration(stats.avg_duration_seconds)}</div>
            <div className="text-sm text-gray-500">{t('executions.stats.avg_duration')}</div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="neu-card p-4">
        <div className="flex items-center space-x-4">
          <Filter size={20} className="text-gray-500" />
          <select
            value={filter.status}
            onChange={(e) => setFilter({ ...filter, status: e.target.value })}
            className="neu-input"
          >
            <option value="">{t('executions.filter.all_status')}</option>
            <option value="pending">{t('executions.status.pending')}</option>
            <option value="running">{t('executions.status.running')}</option>
            <option value="success">{t('executions.status.success')}</option>
            <option value="failed">{t('executions.status.failed')}</option>
          </select>
        </div>
      </div>

      {/* Executions Table */}
      <div className="neu-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('executions.status')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('executions.task')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('executions.agent')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('executions.trigger')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('executions.started')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('executions.duration')}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  {t('common.actions')}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {executions.map((execution) => (
                <tr
                  key={execution.id}
                  className="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
                  onClick={() => navigate(`/executions/${execution.id}`)}
                >
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(execution.status)}
                      <span className="text-sm font-medium capitalize">
                        {execution.status}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="text-sm font-medium">{execution.task_name}</div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="text-sm">{execution.agent_name}</div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={classNames(
                      'px-2 py-1 text-xs rounded',
                      execution.trigger_mode === 'manual' 
                        ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                        : execution.trigger_mode === 'local_fallback'
                        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
                    )}>
                      {getTriggerModeLabel(execution.trigger_mode)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {execution.started_at ? formatDate(execution.started_at) : '-'}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {execution.duration ? formatDuration(execution.duration) : '-'}
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(`/executions/${execution.id}`);
                      }}
                      className="neu-button-icon"
                      title={t('executions.view_details')}
                    >
                      <ChevronRight size={16} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="px-4 py-3 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="neu-button disabled:opacity-50"
              >
                {t('common.previous')}
              </button>
              <span className="text-sm text-gray-500">
                {t('common.page_of', { current: page, total: totalPages })}
              </span>
              <button
                onClick={() => setPage(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
                className="neu-button disabled:opacity-50"
              >
                {t('common.next')}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

// Execution Detail with Real-time Logs
const ExecutionDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const { token } = useAuth();
  const { events } = useSSE();
  const navigate = useNavigate();
  const [execution, setExecution] = useState<TaskExecution | null>(null);
  const [logs, setLogs] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (id) {
      fetchExecutionDetail();
    }
  }, [id]);

  // Listen for real-time log updates
  useEffect(() => {
    const handleLogUpdate = (event: any) => {
      if (event.type === 'execution.log.update' && event.data.execution_id === id) {
        setLogs(prev => prev + event.data.log.message + '\n');
      }
    };

    const handleStatusUpdate = (event: any) => {
      if (event.type === 'execution.status.update' && event.data.execution_id === id) {
        setExecution(prev => prev ? { ...prev, status: event.data.status } : null);
      }
    };

    events.forEach(event => {
      handleLogUpdate(event);
      handleStatusUpdate(event);
    });
  }, [events, id]);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const fetchExecutionDetail = async () => {
    setLoading(true);
    try {
      const response = await apiClient.get(`/admin/executions/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      setExecution(response.data);
      setLogs(response.data.log_output || '');
    } catch (error) {
      console.error('Failed to fetch execution detail:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleScroll = () => {
    if (logsContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
      const isAtBottom = scrollHeight - scrollTop - clientHeight < 10;
      setAutoScroll(isAtBottom);
    }
  };

  const downloadLogs = () => {
    const blob = new Blob([logs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `execution-${id}-logs.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (loading || !execution) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="neu-card p-8">
          <div className="animate-spin rounded-full h-12 w-12 border-4 border-primary border-t-transparent"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <button
            onClick={() => navigate('/executions')}
            className="neu-button-icon"
          >
            ← {t('common.back')}
          </button>
          <h1 className="text-2xl font-bold">{t('executions.execution_detail')}</h1>
        </div>
        <div className="flex items-center space-x-2">
          {getStatusIcon(execution.status)}
          <span className="text-lg font-medium capitalize">{execution.status}</span>
        </div>
      </div>

      {/* Execution Info */}
      <div className="neu-card p-6">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
          <div>
            <div className="text-sm text-gray-500 mb-1">{t('executions.task')}</div>
            <div className="font-medium">{execution.task_name}</div>
          </div>
          <div>
            <div className="text-sm text-gray-500 mb-1">{t('executions.agent')}</div>
            <div className="font-medium">{execution.agent_name}</div>
          </div>
          <div>
            <div className="text-sm text-gray-500 mb-1">{t('executions.trigger')}</div>
            <div className="font-medium">{getTriggerModeLabel(execution.trigger_mode)}</div>
          </div>
          <div>
            <div className="text-sm text-gray-500 mb-1">{t('executions.duration')}</div>
            <div className="font-medium">
              {execution.duration ? formatDuration(execution.duration) : t('executions.in_progress')}
            </div>
          </div>
          <div>
            <div className="text-sm text-gray-500 mb-1">{t('executions.started_at')}</div>
            <div className="font-medium">
              {execution.started_at ? formatDate(execution.started_at) : '-'}
            </div>
          </div>
          <div>
            <div className="text-sm text-gray-500 mb-1">{t('executions.ended_at')}</div>
            <div className="font-medium">
              {execution.ended_at ? formatDate(execution.ended_at) : '-'}
            </div>
          </div>
        </div>

        {execution.error_message && (
          <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
            <div className="text-sm font-medium text-red-800 dark:text-red-200 mb-1">
              {t('executions.error')}
            </div>
            <div className="text-sm text-red-700 dark:text-red-300">
              {execution.error_message}
            </div>
          </div>
        )}
      </div>

      {/* Logs */}
      <div className="neu-card overflow-hidden">
        <div className="p-4 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Terminal size={20} />
              <span className="font-medium">{t('executions.logs')}</span>
            </div>
            <div className="flex items-center space-x-2">
              <label className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                  className="neu-checkbox"
                />
                <span className="text-sm">{t('executions.auto_scroll')}</span>
              </label>
              <button
                onClick={downloadLogs}
                className="neu-button flex items-center space-x-1"
              >
                <Download size={16} />
                <span>{t('common.download')}</span>
              </button>
            </div>
          </div>
        </div>

        <div
          ref={logsContainerRef}
          onScroll={handleScroll}
          className="bg-gray-900 text-gray-100 p-4 font-mono text-sm overflow-auto"
          style={{ height: '500px', maxHeight: '60vh' }}
        >
          {logs ? (
            <>
              <pre className="whitespace-pre-wrap">{logs}</pre>
              <div ref={logsEndRef} />
            </>
          ) : (
            <div className="text-gray-500 italic">
              {execution.status === 'pending' 
                ? t('executions.waiting_to_start')
                : t('executions.no_logs')}
            </div>
          )}
          {execution.status === 'running' && (
            <div className="inline-flex items-center space-x-2 mt-2">
              <RefreshCw className="animate-spin" size={16} />
              <span className="text-green-400">{t('executions.running_live')}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// Export both components
export { ExecutionList, ExecutionDetail };
export default ExecutionList;
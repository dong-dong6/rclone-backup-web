import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import {
  IconAlertCircle,
  IconCheck,
  IconChevronLeft,
  IconChevronRight,
  IconClock,
  IconRefresh,
  IconX,
} from '@tabler/icons-react';
import { apiClient } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { useSSE } from '../contexts/SSEContext';

interface TaskExecution {
  id: string;
  task_id: string;
  task_name: string;
  agent_id: string;
  agent_name: string;
  status: 'pending' | 'running' | 'success' | 'failed';
  trigger_mode: 'manual' | 'scheduled' | 'local_fallback' | 'central';
  log_output?: string;
  error_message?: string;
  started_at?: string;
  ended_at?: string;
  created_at: string;
  duration_seconds?: number;
}

interface ExecutionStats {
  total: number;
  success: number;
  failed: number;
  running: number;
  success_rate_24h: number;
  avg_duration_seconds: number;
}

const formatDuration = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds < 0) return '-';
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
};

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr);
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString();
};

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

      const items = Array.isArray(response.data?.items)
        ? response.data.items
        : Array.isArray(response.data?.executions)
          ? response.data.executions
          : [];

      setExecutions(items);
      setTotalPages(Number.isFinite(response.data?.total_pages) ? response.data.total_pages : 1);
    } catch (error) {
      console.error('Failed to fetch executions:', error);
      setExecutions([]);
      setTotalPages(1);
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

  const getStatusBadge = (status: TaskExecution['status']) => {
    const config: Record<TaskExecution['status'], { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <IconCheck size={16} /> },
      failed: { color: 'danger', icon: <IconX size={16} /> },
      running: { color: 'primary', icon: <IconRefresh size={16} className="spinner" /> },
      pending: { color: 'warning', icon: <IconClock size={16} /> },
    };

    const { color, icon } = config[status] || { color: 'secondary', icon: <IconAlertCircle size={16} /> };
    return (
      <span className={`badge bg-${color} text-white`}>
        {icon}
        <span className="ms-1">{t(`executions.status.${status}`) || status}</span>
      </span>
    );
  };

  const getTriggerModeLabel = (mode: TaskExecution['trigger_mode']) => {
    switch (mode) {
      case 'manual':
        return t('executions.triggerMode.manual');
      case 'scheduled':
      case 'central':
        return t('executions.triggerMode.scheduled');
      case 'local_fallback':
        return t('executions.triggerMode.local_fallback');
      default:
        return mode;
    }
  };

  const getTriggerModeBadge = (mode: TaskExecution['trigger_mode']) => {
    const label = getTriggerModeLabel(mode);
    const { color, textClass } =
      mode === 'manual'
        ? { color: 'primary', textClass: 'text-white' }
        : mode === 'local_fallback'
          ? { color: 'warning', textClass: 'text-dark' }
          : { color: 'secondary', textClass: 'text-white' };

    return (
      <span className={`badge bg-${color} ${textClass}`}>
        {label}
      </span>
    );
  };

  if (loading && executions.length === 0) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <IconRefresh className="spinner text-primary mb-3" size={48} />
              <p className="text-muted">{t('common.loading')}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="row row-deck row-cards">
      {stats && (
        <>
          <div className="col-sm-6 col-lg-2">
            <div className="card">
              <div className="card-body">
                <div className="subheader">{t('executions.stats.total')}</div>
                <div className="h1 mb-0">{stats.total}</div>
              </div>
            </div>
          </div>
          <div className="col-sm-6 col-lg-2">
            <div className="card">
              <div className="card-body">
                <div className="subheader">{t('executions.stats.success')}</div>
                <div className="h1 mb-0 text-success">{stats.success}</div>
              </div>
            </div>
          </div>
          <div className="col-sm-6 col-lg-2">
            <div className="card">
              <div className="card-body">
                <div className="subheader">{t('executions.stats.failed')}</div>
                <div className="h1 mb-0 text-danger">{stats.failed}</div>
              </div>
            </div>
          </div>
          <div className="col-sm-6 col-lg-2">
            <div className="card">
              <div className="card-body">
                <div className="subheader">{t('executions.stats.running')}</div>
                <div className="h1 mb-0 text-primary">{stats.running}</div>
              </div>
            </div>
          </div>
          <div className="col-sm-6 col-lg-2">
            <div className="card">
              <div className="card-body">
                <div className="subheader">{t('executions.stats.success_rate')}</div>
                <div className="h1 mb-0">{stats.success_rate_24h.toFixed(1)}%</div>
              </div>
            </div>
          </div>
          <div className="col-sm-6 col-lg-2">
            <div className="card">
              <div className="card-body">
                <div className="subheader">{t('executions.stats.avg_duration')}</div>
                <div className="h1 mb-0">{formatDuration(stats.avg_duration_seconds)}</div>
              </div>
            </div>
          </div>
        </>
      )}

      <div className="col-12">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">{t('executions.list.title')}</h3>
            <div className="ms-auto d-flex gap-2">
              <button
                onClick={() => {
                  fetchExecutions();
                  fetchStats();
                }}
                className="btn btn-outline-primary btn-sm"
                disabled={loading}
              >
                <IconRefresh size={16} className={loading ? 'spinner' : undefined} />
                <span className="ms-1">{t('common.refresh')}</span>
              </button>
            </div>
          </div>

          <div className="card-body border-bottom py-3">
            <div className="row g-3 align-items-end">
              <div className="col-md-3">
                <label className="form-label">{t('executions.list.columns.status')}</label>
                <select
                  value={filter.status}
                  onChange={(e) => {
                    setPage(1);
                    setFilter({ ...filter, status: e.target.value });
                  }}
                  className="form-select"
                >
                  <option value="">{t('executions.filter.all_status')}</option>
                  <option value="pending">{t('executions.status.pending')}</option>
                  <option value="running">{t('executions.status.running')}</option>
                  <option value="success">{t('executions.status.success')}</option>
                  <option value="failed">{t('executions.status.failed')}</option>
                </select>
              </div>
            </div>
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
                {executions.length > 0 ? (
                  executions.map((execution) => (
                    <tr
                      key={execution.id}
                      style={{ cursor: 'pointer' }}
                      onClick={() => navigate(`/executions/${execution.id}`)}
                    >
                      <td>{getStatusBadge(execution.status)}</td>
                      <td className="text-break">{execution.task_name}</td>
                      <td className="text-break">{execution.agent_name}</td>
                      <td>{getTriggerModeBadge(execution.trigger_mode)}</td>
                      <td className="text-muted">
                        {execution.started_at ? formatDate(execution.started_at) : '-'}
                      </td>
                      <td className="text-muted">
                        {execution.duration_seconds ? formatDuration(execution.duration_seconds) : '-'}
                      </td>
                      <td>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            navigate(`/executions/${execution.id}`);
                          }}
                          className="btn btn-outline-primary btn-sm"
                          title={t('executions.view_details')}
                        >
                          <IconChevronRight size={16} />
                        </button>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={7} className="text-center text-muted py-4">
                      {t('executions.list.empty')}
                    </td>
                  </tr>
                )}
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

const ExecutionDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const { token } = useAuth();
  const { subscribe } = useSSE();
  const navigate = useNavigate();

  const [execution, setExecution] = useState<TaskExecution | null>(null);
  const [logs, setLogs] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!id) return;
    fetchExecutionDetail();
  }, [id]);

  useEffect(() => {
    if (!id) return;

    const unsubscribeLog = subscribe('execution.log.update', (event) => {
      if (event.data?.execution_id !== id) return;
      const message = event.data?.log?.message;
      if (!message) return;
      setLogs(prev => prev + message + '\n');
    });

    const unsubscribeStatus = subscribe('execution.status.update', (event) => {
      if (event.data?.execution_id !== id) return;
      const status = event.data?.status as TaskExecution['status'] | undefined;
      if (!status) return;
      setExecution(prev => (prev ? { ...prev, status } : prev));
    });

    return () => {
      unsubscribeLog();
      unsubscribeStatus();
    };
  }, [id]);

  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const fetchExecutionDetail = async () => {
    setLoading(true);
    setLoadError(false);

    try {
      const response = await apiClient.get(`/admin/executions/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      setExecution(response.data);
      setLogs(response.data?.log_output || '');
    } catch (error) {
      console.error('Failed to fetch execution detail:', error);
      setExecution(null);
      setLogs('');
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  };

  const handleScroll = () => {
    if (!logsContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 10;
    setAutoScroll(isAtBottom);
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

  const getStatusBadge = (status: TaskExecution['status']) => {
    const config: Record<TaskExecution['status'], { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <IconCheck size={16} /> },
      failed: { color: 'danger', icon: <IconX size={16} /> },
      running: { color: 'primary', icon: <IconRefresh size={16} className="spinner" /> },
      pending: { color: 'warning', icon: <IconClock size={16} /> },
    };

    const { color, icon } = config[status] || { color: 'secondary', icon: <IconAlertCircle size={16} /> };
    return (
      <span className={`badge bg-${color} text-white`}>
        {icon}
        <span className="ms-1">{t(`executions.status.${status}`) || status}</span>
      </span>
    );
  };

  const getTriggerModeLabel = (mode: TaskExecution['trigger_mode']) => {
    switch (mode) {
      case 'manual':
        return t('executions.triggerMode.manual');
      case 'scheduled':
      case 'central':
        return t('executions.triggerMode.scheduled');
      case 'local_fallback':
        return t('executions.triggerMode.local_fallback');
      default:
        return mode;
    }
  };

  if (loading) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <IconRefresh className="spinner text-primary mb-3" size={48} />
              <p className="text-muted">{t('common.loading')}</p>
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
              <button className="btn btn-outline-secondary" onClick={() => navigate('/executions')}>
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
            <button className="btn btn-outline-secondary" onClick={() => navigate('/executions')}>
              <IconChevronLeft size={16} />
              <span className="ms-1">{t('common.back')}</span>
            </button>
            <div className="ms-auto">{getStatusBadge(execution.status)}</div>
          </div>
        </div>
      </div>

      <div className="col-12">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">{t('executions.detail.info')}</h3>
          </div>
          <div className="card-body">
            <div className="row g-3">
              <div className="col-12 col-sm-6 col-lg-4">
                <div className="text-muted small">{t('executions.task')}</div>
                <div className="fw-bold text-break">{execution.task_name}</div>
              </div>
              <div className="col-12 col-sm-6 col-lg-4">
                <div className="text-muted small">{t('executions.agent')}</div>
                <div className="fw-bold text-break">{execution.agent_name}</div>
              </div>
              <div className="col-12 col-sm-6 col-lg-4">
                <div className="text-muted small">{t('executions.list.columns.triggerMode')}</div>
                <div className="fw-bold text-break">{getTriggerModeLabel(execution.trigger_mode)}</div>
              </div>
              <div className="col-12 col-sm-6 col-lg-4">
                <div className="text-muted small">{t('executions.duration')}</div>
                <div className="fw-bold">
                  {execution.duration_seconds ? formatDuration(execution.duration_seconds) : t('executions.in_progress')}
                </div>
              </div>
              <div className="col-12 col-sm-6 col-lg-4">
                <div className="text-muted small">{t('executions.started_at')}</div>
                <div className="fw-bold">{execution.started_at ? formatDate(execution.started_at) : '-'}</div>
              </div>
              <div className="col-12 col-sm-6 col-lg-4">
                <div className="text-muted small">{t('executions.ended_at')}</div>
                <div className="fw-bold">{execution.ended_at ? formatDate(execution.ended_at) : '-'}</div>
              </div>
            </div>

            {execution.error_message && (
              <div className="alert alert-danger mt-3 mb-0" role="alert">
                <div className="fw-bold mb-1">{t('executions.error')}</div>
                <div className="text-break">{execution.error_message}</div>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="col-12">
          <div className="card">
          <div className="card-header">
            <h3 className="card-title">{t('executions.logs.title')}</h3>
            <div className="ms-auto d-flex align-items-center gap-3">
              <div className="form-check form-switch m-0">
                <input
                  type="checkbox"
                  className="form-check-input"
                  id={`auto-scroll-${id}`}
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                />
                <label className="form-check-label" htmlFor={`auto-scroll-${id}`}>
                  {t('executions.auto_scroll')}
                </label>
              </div>
              <button
                onClick={downloadLogs}
                className="btn btn-outline-primary btn-sm"
                disabled={!logs}
              >
                {t('common.download')}
              </button>
            </div>
          </div>

          <div className="card-body p-0">
            <div
              ref={logsContainerRef}
              onScroll={handleScroll}
              className="bg-dark text-white font-monospace p-3"
              style={{ height: '500px', maxHeight: '60vh', overflow: 'auto' }}
            >
              {logs ? (
                <>
                  <pre className="mb-0" style={{ whiteSpace: 'pre-wrap' }}>{logs}</pre>
                  <div ref={logsEndRef} />
                </>
              ) : (
                <div className="text-secondary fst-italic">
                  {execution.status === 'pending'
                    ? t('executions.waiting_to_start')
                    : t('executions.no_logs')}
                </div>
              )}

              {execution.status === 'running' && (
                <div className="mt-3 d-flex align-items-center gap-2">
                  <IconRefresh className="spinner text-success" size={16} />
                  <span className="text-success">{t('executions.running_live')}</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export { ExecutionList, ExecutionDetail };
export default ExecutionList;

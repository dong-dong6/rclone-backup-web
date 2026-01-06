import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconWifi, IconWifiOff, IconActivity, IconPlus, IconCopy, IconCheck,
  IconRefresh, IconTrash, IconServer, IconClock, IconCalendar,
  IconChevronRight, IconAlertCircle, IconEdit
} from '@tabler/icons-react';
import { apiClient } from '../services/api';
import { useSSE } from '../contexts/SSEContext';
import classNames from 'classnames';
import { LineChart, Line, CartesianGrid, XAxis, Tooltip, ResponsiveContainer, Legend } from 'recharts';

interface Agent {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'running_task';
  last_heartbeat: string | null;
  is_local: boolean;
  created_at: string;
  task_count?: number;
  current_task?: string;
}

interface AgentMetric {
  cpu_usage: number;
  memory_usage: number;
  memory_total: number;
  memory_used: number;
  disk_total: number;
  disk_used: number;
  disk_usage: number;
  network_rx_rate: number;
  network_tx_rate: number;
  tcp_connections: number;
  udp_connections: number;
  process_count: number;
  recorded_at: string;
}

interface HeartbeatEvent {
  agent_id: string;
  status: string;
  timestamp: string;
  actions?: number;
  metrics?: {
    cpu_usage: number;
    memory_usage: number;
    memory_total: number;
    memory_used: number;
    disk_usage: number;
    disk_total: number;
    disk_used: number;
    network_rx_rate: number;
    network_tx_rate: number;
    tcp_connections: number;
    udp_connections: number;
    process_count: number;
    recorded_at: string;
  };
}

const Agents: React.FC = () => {
  const { t } = useTranslation();
  const { subscribe } = useSSE();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [registrationToken, setRegistrationToken] = useState('');
  const [tokenExpiry, setTokenExpiry] = useState<Date | null>(null);
  const [copied, setCopied] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
  const [metricsLatest, setMetricsLatest] = useState<AgentMetric | null>(null);
  const [metricsHistory, setMetricsHistory] = useState<AgentMetric[]>([]);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [realtimeStats, setRealtimeStats] = useState({
    totalAgents: 0,
    onlineAgents: 0,
    runningTasks: 0
  });
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [editName, setEditName] = useState('');

  // Use ref to track selectedAgent for SSE handler without re-subscribing
  const selectedAgentRef = useRef<Agent | null>(null);
  useEffect(() => {
    selectedAgentRef.current = selectedAgent;
  }, [selectedAgent]);

  // Load agents and setup SSE listeners
  useEffect(() => {
    loadAgents();

    // Subscribe to multiple SSE events (except heartbeat which has its own effect)
    const unsubscribers = [
      // Agent status updates
      subscribe('agent.status.update', (event) => {
        handleAgentStatusUpdate(event.data);
      }),

      // Task dispatch events
      subscribe('task.dispatched', (event) => {
        handleTaskDispatched(event.data);
      }),

      // Agent heartbeat events with real-time metrics update
      subscribe('agent.heartbeat', (event: { data: HeartbeatEvent }) => {
        const { agent_id, timestamp, metrics } = event.data;

        // Update agent list
        setAgents(prev => prev.map(agent => {
          if (agent.id === agent_id) {
            return {
              ...agent,
              last_heartbeat: timestamp,
              className: 'heartbeat-pulse'
            };
          }
          return agent;
        }));

        // Remove pulse animation after 500ms
        setTimeout(() => {
          setAgents(prev => prev.map(agent => {
            if (agent.id === agent_id) {
              const { className, ...rest } = agent as any;
              return rest;
            }
            return agent;
          }));
        }, 500);

        // Update metrics for selected agent in real-time using ref
        const currentSelectedAgent = selectedAgentRef.current;
        if (currentSelectedAgent && agent_id === currentSelectedAgent.id && metrics) {
          const newMetric: AgentMetric = {
            cpu_usage: metrics.cpu_usage,
            memory_usage: metrics.memory_usage,
            memory_total: metrics.memory_total,
            memory_used: metrics.memory_used,
            disk_usage: metrics.disk_usage,
            disk_total: metrics.disk_total,
            disk_used: metrics.disk_used,
            network_rx_rate: metrics.network_rx_rate,
            network_tx_rate: metrics.network_tx_rate,
            tcp_connections: metrics.tcp_connections,
            udp_connections: metrics.udp_connections,
            process_count: metrics.process_count,
            recorded_at: metrics.recorded_at || new Date().toISOString(),
          };

          setMetricsLatest(newMetric);

          // Append to history chart
          setMetricsHistory(prev => {
            const updated = [...prev, newMetric];
            return updated.slice(-360);
          });
        }
      }),

      // Agent registration events
      subscribe('agent.registered', (event) => {
        handleNewAgentRegistered(event.data);
      })
    ];

    // Cleanup subscriptions
    return () => {
      unsubscribers.forEach(unsubscribe => unsubscribe());
    };
  }, []);

  // Update real-time statistics whenever agents change
  useEffect(() => {
    const stats = {
      totalAgents: agents.length,
      onlineAgents: agents.filter(a => a.status === 'online').length,
      runningTasks: agents.filter(a => a.status === 'running_task').length
    };
    setRealtimeStats(stats);
  }, [agents]);

  useEffect(() => {
    if (!selectedAgent) {
      setMetricsLatest(null);
      setMetricsHistory([]);
      return;
    }

    let cancelled = false;
    const loadMetrics = async () => {
      setMetricsLoading(true);
      try {
        const [latestResp, historyResp] = await Promise.all([
          apiClient.get(`/admin/agents/${selectedAgent.id}/metrics/latest`),
          apiClient.get(`/admin/agents/${selectedAgent.id}/metrics/history`, { params: { hours: 6 } })
        ]);

        if (cancelled) return;

        setMetricsLatest(latestResp.data);
        setMetricsHistory(historyResp.data);
      } catch (error) {
        if (!cancelled) {
          console.error('Failed to load metrics:', error);
        }
      } finally {
        if (!cancelled) {
          setMetricsLoading(false);
        }
      }
    };

    loadMetrics();

    return () => {
      cancelled = true;
    };
  }, [selectedAgent]);

  const loadAgents = async () => {
    try {
      setLoading(true);
      const response = await apiClient.get('/admin/agents');
      setAgents(response.data);
    } catch (error) {
      console.error('Failed to load agents:', error);
      // Show notification
      showNotification('error', t('agents.load_failed'));
    } finally {
      setLoading(false);
    }
  };

  // SSE Event Handlers
  const handleAgentStatusUpdate = (data: any) => {
    const { agent_id, status } = data;

    setAgents(prev => prev.map(agent => {
      if (agent.id === agent_id) {
        // Animate status change
        animateStatusChange(agent_id);

        return {
          ...agent,
          status,
          last_heartbeat: new Date().toISOString()
        };
      }
      return agent;
    }));

    // Show notification for important status changes
    const agent = agents.find(a => a.id === agent_id);
    if (agent) {
      if (status === 'offline') {
        showNotification('warning', `${agent.name} ${t('agents.went_offline')}`);
      } else if (status === 'online') {
        showNotification('success', `${agent.name} ${t('agents.came_online')}`);
      }
    }
  };

  const handleTaskDispatched = (data: any) => {
    const { agent_id, task_name } = data;

    setAgents(prev => prev.map(agent => {
      if (agent.id === agent_id) {
        return {
          ...agent,
          status: 'running_task',
          current_task: task_name
        };
      }
      return agent;
    }));

    showNotification('info', `${t('agents.task_dispatched')}: ${task_name}`);
  };

  const handleNewAgentRegistered = (data: any) => {
    // Reload agents list to include new agent
    loadAgents();
    showNotification('success', `${t('agents.new_agent_registered')}: ${data.name}`);
  };

  // UI Helper Functions
  const generateToken = async () => {
    try {
      const response = await apiClient.post('/admin/agents/registration-token');
      setRegistrationToken(response.data.token);

      // Set token expiry (24 hours from now)
      const expiry = new Date();
      expiry.setHours(expiry.getHours() + 24);
      setTokenExpiry(expiry);

      setShowRegisterModal(true);
    } catch (error) {
      console.error('Failed to generate token:', error);
      showNotification('error', t('agents.token_generation_failed'));
    }
  };

  const copyToClipboard = () => {
    navigator.clipboard.writeText(registrationToken);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const deleteAgent = async (id: string, name: string) => {
    if (!confirm(t('agents.actions.deleteConfirm', { name }))) {
      return;
    }

    try {
      await apiClient.delete(`/admin/agents/${id}`);

      // Optimistic UI update with animation
      setAgents(prev => prev.filter(agent => agent.id !== id));

      showNotification('success', t('agents.deleted_successfully'));
    } catch (error) {
      console.error('Failed to delete agent:', error);
      showNotification('error', t('agents.delete_failed'));
      // Reload to restore correct state
      loadAgents();
    }
  };

  const syncConfig = async (agentId: string) => {
    try {
      await apiClient.post(`/admin/agents/${agentId}/sync`);
      showNotification('success', t('agents.config_sync_triggered'));
    } catch (error) {
      console.error('Failed to sync config:', error);
      showNotification('error', t('agents.config_sync_failed'));
    }
  };

  const startEditAgent = (agent: Agent) => {
    setEditingAgent(agent);
    setEditName(agent.name);
  };

  const updateAgent = async () => {
    if (!editingAgent || !editName.trim()) return;

    try {
      await apiClient.put(`/admin/agents/${editingAgent.id}`, { name: editName.trim() });

      // Update local state
      setAgents(prev => prev.map(agent =>
        agent.id === editingAgent.id ? { ...agent, name: editName.trim() } : agent
      ));

      showNotification('success', t('agents.updated_successfully'));
      setEditingAgent(null);
      setEditName('');
    } catch (error) {
      console.error('Failed to update agent:', error);
      showNotification('error', t('agents.update_failed'));
    }
  };

  const isLocalAgent = (agent: Agent) => agent.is_local;

  // Animation helpers
  const animateStatusChange = (agentId: string) => {
    const element = document.getElementById(`agent-card-${agentId}`);
    if (element) {
      element.classList.add('status-change-animation');
      setTimeout(() => {
        element.classList.remove('status-change-animation');
      }, 1000);
    }
  };

  const showNotification = (type: 'success' | 'error' | 'warning' | 'info', message: string) => {
    // This would integrate with a notification system
    console.log(`[${type.toUpperCase()}] ${message}`);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
        return <span className="badge bg-success">{t('common.online')}</span>;
      case 'offline':
        return <span className="badge bg-danger">{t('common.offline')}</span>;
      case 'running_task':
        return <span className="badge bg-primary">{t('common.running')}</span>;
      default:
        return <span className="badge bg-warning">{t('common.offline')}</span>;
    }
  };

  const formatLastHeartbeat = (heartbeat: string | null) => {
    if (!heartbeat) return t('common.never');

    const date = new Date(heartbeat);
    // 显示日期+时间格式 YYYY-MM-DD HH:MM:SS
    return date.toLocaleString();
  };

  const formatBytes = (value: number) => {
    if (!value) return '0 B';
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
    let idx = 0;
    let val = value;
    while (val >= 1024 && idx < units.length - 1) {
      val /= 1024;
      idx += 1;
    }
    return `${val.toFixed(2)} ${units[idx]}`;
  };

  const formatRate = (value: number) => `${formatBytes(value)}/s`;

  const chartData = metricsHistory.map((metric) => ({
    time: new Date(metric.recorded_at).toLocaleTimeString(),
    cpu: Number(metric.cpu_usage.toFixed(2)),
    memory: Number(metric.memory_usage.toFixed(2)),
  }));

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

  return (
    <div className="row row-deck row-cards">
      {/* Stats Cards */}
      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('agents.stats.total')}</div>
            </div>
            <div className="h1 mb-3">{realtimeStats.totalAgents}</div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('agents.stats.online')}</div>
            </div>
            <div className="h1 mb-3 text-success">{realtimeStats.onlineAgents}</div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">{t('agents.stats.running')}</div>
            </div>
            <div className="h1 mb-3 text-primary">{realtimeStats.runningTasks}</div>
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
                onClick={loadAgents}
                className="btn btn-outline-primary btn-sm"
              >
                <IconRefresh size={16} />
                {t('common.refresh')}
              </button>

              <button
                onClick={generateToken}
                className="btn btn-primary btn-sm"
              >
                <IconPlus size={16} />
                {t('agents.register.new')}
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Agents Grid */}
      <div className="col-12">
        <div className="row row-deck row-cards">
          {agents.map((agent) => (
            <div
              key={agent.id}
              id={`agent-card-${agent.id}`}
              className="col-md-6 col-lg-4"
            >
              <div className={classNames(
                'card',
                (agent as any).className
              )}>
                <div className="card-body">
                  {/* Agent Header */}
                  <div className="d-flex align-items-center justify-content-between mb-3">
                    <div className="d-flex align-items-center">
                      <IconServer className="text-primary me-2" size={24} />
                      <div>
                        <h3 className="card-title mb-0">{agent.name}</h3>
                        <small className="text-muted">{agent.id.substring(0, 8)}...</small>
                      </div>
                    </div>
                    {getStatusIcon(agent.status)}
                  </div>

                  {/* Agent Details */}
                  <div className="mb-3">
                    <div className="row">
                      <div className="col-6">
                        <div className="text-muted small">{t('agents.last_heartbeat')}</div>
                        <div className="fw-bold">{formatLastHeartbeat(agent.last_heartbeat)}</div>
                      </div>
                      <div className="col-6">
                        <div className="text-muted small">{t('agents.assigned_tasks')}</div>
                        <div className="fw-bold">{agent.task_count || 0}</div>
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="d-flex gap-2 pt-2 border-top">
                    <button
                      onClick={() => syncConfig(agent.id)}
                      className="btn btn-outline-primary btn-sm d-flex align-items-center gap-1"
                      title={t('agents.actions.sync_config')}
                      disabled={agent.status === 'offline'}
                    >
                      <IconRefresh size={14} />
                      <span>{t('agents.actions.sync')}</span>
                    </button>

                    <button
                      onClick={() => setSelectedAgent(agent)}
                      className="btn btn-outline-secondary btn-sm d-flex align-items-center gap-1"
                      title={t('agents.actions.view_details')}
                    >
                      <IconChevronRight size={14} />
                      <span>{t('agents.actions.details')}</span>
                    </button>

                    <button
                      onClick={() => startEditAgent(agent)}
                      className="btn btn-outline-info btn-sm d-flex align-items-center gap-1"
                      title={t('common.edit')}
                    >
                      <IconEdit size={14} />
                      <span>{t('common.edit')}</span>
                    </button>

                    {!isLocalAgent(agent) && (
                      <button
                        onClick={() => deleteAgent(agent.id, agent.name)}
                        className="btn btn-outline-danger btn-sm d-flex align-items-center gap-1"
                        title={t('agents.actions.delete')}
                      >
                        <IconTrash size={14} />
                        <span>{t('common.delete')}</span>
                      </button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Empty State */}
      {agents.length === 0 && (
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <IconServer className="mx-auto text-muted mb-4" size={64} />
              <h3 className="card-title">{t('agents.no_agents')}</h3>
              <p className="text-muted mb-4">{t('agents.no_agents_description')}</p>
              <button
                onClick={generateToken}
                className="btn btn-primary"
              >
                <IconPlus size={16} className="me-1" />
                {t('agents.register.first_agent')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Registration Modal */}
      {showRegisterModal && (
        <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1}>
          <div className="modal-dialog modal-lg modal-dialog-centered">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">{t('agents.register.title')}</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => {
                    setShowRegisterModal(false);
                    setRegistrationToken('');
                    setTokenExpiry(null);
                  }}
                ></button>
              </div>
              <div className="modal-body">
                <div className="mb-3">
                  <label className="form-label">
                    {t('agents.register.token_label')}
                  </label>
                  <div className="input-group">
                    <input
                      type="text"
                      value={registrationToken}
                      readOnly
                      className="form-control font-monospace"
                    />
                    <button
                      onClick={copyToClipboard}
                      className="btn btn-outline-secondary"
                    >
                      {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                      <span className="ms-1">{copied ? t('common.copied') : t('common.copy')}</span>
                    </button>
                  </div>
                </div>

                {tokenExpiry && (
                  <div className="alert alert-warning">
                    <IconAlertCircle className="me-1" size={16} />
                    {t('agents.register.token_expires', {
                      time: tokenExpiry.toLocaleString()
                    })}
                  </div>
                )}

                <div className="card">
                  <div className="card-header">
                    <h3 className="card-title">{t('agents.register.instructions')}</h3>
                  </div>
                  <div className="card-body">
                    <ol className="list-unstyled">
                      <li className="mb-2">1. {t('agents.register.step1')}</li>
                      <li className="mb-2">2. {t('agents.register.step2')}</li>
                      <li className="mb-2">
                        <code className="d-block bg-dark text-light p-2 rounded">
                          curl -L http://hub:8080/api/v1/agent/download -o rclone-backup-agent<br />
                          chmod +x rclone-backup-agent<br />
                          ./rclone-backup-agent register \\<br />
                          &nbsp;&nbsp;--hub-url http://hub:8080 \\<br />
                          &nbsp;&nbsp;--token {registrationToken} \\<br />
                          &nbsp;&nbsp;--name my-agent \\<br />
                          &nbsp;&nbsp;--daemon
                        </code>
                      </li>
                      <li>3. {t('agents.register.step3')}</li>
                    </ol>
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button
                  onClick={() => {
                    setShowRegisterModal(false);
                    setRegistrationToken('');
                    setTokenExpiry(null);
                  }}
                  className="btn btn-secondary"
                >
                  {t('common.close')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edit Agent Modal */}
      {editingAgent && (
        <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1}>
          <div className="modal-dialog modal-dialog-centered">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">{t('agents.edit.title')}</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => {
                    setEditingAgent(null);
                    setEditName('');
                  }}
                ></button>
              </div>
              <div className="modal-body">
                <div className="mb-3">
                  <label className="form-label">{t('agents.edit.name_label')}</label>
                  <input
                    type="text"
                    className="form-control"
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    placeholder={t('agents.edit.name_placeholder')}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button
                  onClick={() => {
                    setEditingAgent(null);
                    setEditName('');
                  }}
                  className="btn btn-secondary"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={updateAgent}
                  className="btn btn-primary"
                  disabled={!editName.trim() || editName === editingAgent.name}
                >
                  {t('common.save')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Agent Detail Modal */}
      {selectedAgent && (
        <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1}>
          <div className="modal-dialog modal-lg modal-dialog-centered">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">{selectedAgent.name}</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => setSelectedAgent(null)}
                ></button>
              </div>
              <div className="modal-body">
                <div className="mb-4">
                  <div className="d-flex justify-content-between align-items-center mb-3">
                    <h6 className="mb-0">实时监控</h6>
                    {metricsLatest && (
                      <small className="text-muted">
                        更新于 {new Date(metricsLatest.recorded_at).toLocaleTimeString()}
                      </small>
                    )}
                  </div>
                  {metricsLoading ? (
                    <div className="text-center py-5">
                      <IconRefresh className="spinner text-primary" size={32} />
                    </div>
                  ) : (
                    <div className="row g-3">
                      <div className="col-md-6">
                        <div className="border rounded p-3 h-100">
                          <div className="d-flex justify-content-between small text-muted">
                            <span>CPU</span>
                            <span>
                              {metricsLatest ? `${metricsLatest.cpu_usage.toFixed(1)}%` : '--'}
                            </span>
                          </div>
                          <div className="progress progress-sm mt-2">
                            <div
                              className="progress-bar bg-primary"
                              role="progressbar"
                              style={{ width: `${metricsLatest?.cpu_usage ?? 0}%` }}
                            ></div>
                          </div>
                        </div>
                      </div>
                      <div className="col-md-6">
                        <div className="border rounded p-3 h-100">
                          <div className="d-flex justify-content-between small text-muted">
                            <span>内存</span>
                            <span>
                              {metricsLatest
                                ? `${metricsLatest.memory_usage.toFixed(1)}%`
                                : '--'}
                            </span>
                          </div>
                          <p className="mb-1">
                            {metricsLatest
                              ? `${formatBytes(metricsLatest.memory_used)} / ${formatBytes(
                                metricsLatest.memory_total
                              )}`
                              : '--'}
                          </p>
                          <div className="progress progress-sm">
                            <div
                              className="progress-bar bg-info"
                              role="progressbar"
                              style={{ width: `${metricsLatest?.memory_usage ?? 0}%` }}
                            ></div>
                          </div>
                        </div>
                      </div>
                      <div className="col-md-6">
                        <div className="border rounded p-3 h-100">
                          <div className="d-flex justify-content-between small text-muted">
                            <span>磁盘</span>
                            <span>
                              {metricsLatest ? `${metricsLatest.disk_usage.toFixed(1)}%` : '--'}
                            </span>
                          </div>
                          <p className="mb-1">
                            {metricsLatest
                              ? `${formatBytes(metricsLatest.disk_used)} / ${formatBytes(
                                metricsLatest.disk_total
                              )}`
                              : '--'}
                          </p>
                          <div className="progress progress-sm">
                            <div
                              className="progress-bar bg-warning"
                              role="progressbar"
                              style={{ width: `${metricsLatest?.disk_usage ?? 0}%` }}
                            ></div>
                          </div>
                        </div>
                      </div>
                      <div className="col-md-6">
                        <div className="border rounded p-3 h-100">
                          <div className="d-flex justify-content-between small text-muted">
                            <span>网络</span>
                            <span>
                              {metricsLatest
                                ? `${formatRate(metricsLatest.network_rx_rate)} ↓ / ${formatRate(
                                  metricsLatest.network_tx_rate
                                )} ↑`
                                : '--'}
                            </span>
                          </div>
                          <p className="mb-1 text-muted">
                            TCP {metricsLatest?.tcp_connections ?? '--'} · UDP{' '}
                            {metricsLatest?.udp_connections ?? '--'}
                          </p>
                          <p className="mb-0 text-muted">
                            进程数 {metricsLatest?.process_count ?? '--'}
                          </p>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
                <div>
                  <div className="d-flex justify-content-between align-items-center mb-3">
                    <h6 className="mb-0">历史趋势 (6h)</h6>
                  </div>
                  {metricsHistory.length > 0 ? (
                    <ResponsiveContainer width="100%" height={240}>
                      <LineChart data={chartData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="time" />
                        <Tooltip />
                        <Legend />
                        <Line
                          type="monotone"
                          dataKey="cpu"
                          stroke="#0d6efd"
                          strokeWidth={2}
                          dot={false}
                        />
                        <Line
                          type="monotone"
                          dataKey="memory"
                          stroke="#0dcaf0"
                          strokeWidth={2}
                          dot={false}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  ) : (
                    <p className="text-muted mb-0">
                      {metricsLoading ? '正在加载历史数据...' : '暂无历史监控数据。'}
                    </p>
                  )}
                </div>
              </div>
              <div className="modal-footer">
                <button
                  onClick={() => setSelectedAgent(null)}
                  className="btn btn-secondary"
                >
                  {t('common.close')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Agents;

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  IconWifi, IconWifiOff, IconActivity, IconPlus, IconCopy, IconCheck, 
  IconRefresh, IconTrash, IconServer, IconClock, IconCalendar,
  IconChevronRight, IconAlertCircle, IconCheckCircle
} from '@tabler/icons-react';
import { apiClient } from '../services/api';
import { useSSE } from '../contexts/SSEContext';
import classNames from 'classnames';

interface Agent {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'running_task';
  last_heartbeat: string | null;
  created_at: string;
  task_count?: number;
  current_task?: string;
  system_info?: {
    platform: string;
    hostname: string;
    cpu_usage: number;
    memory_usage: number;
  };
}

interface HeartbeatEvent {
  agent_id: string;
  status: string;
  timestamp: string;
  actions?: number;
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
  const [realtimeStats, setRealtimeStats] = useState({
    totalAgents: 0,
    onlineAgents: 0,
    runningTasks: 0
  });

  // Load agents and setup SSE listeners
  useEffect(() => {
    loadAgents();
    
    // Subscribe to multiple SSE events
    const unsubscribers = [
      // Agent status updates
      subscribe('agent.status.update', (event) => {
        handleAgentStatusUpdate(event.data);
      }),
      
      // Agent heartbeat events
      subscribe('agent.heartbeat', (event: { data: HeartbeatEvent }) => {
        handleHeartbeat(event.data);
      }),
      
      // Task dispatch events
      subscribe('task.dispatched', (event) => {
        handleTaskDispatched(event.data);
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

  const handleHeartbeat = (data: HeartbeatEvent) => {
    const { agent_id, timestamp } = data;
    
    setAgents(prev => prev.map(agent => {
      if (agent.id === agent_id) {
        return { 
          ...agent, 
          last_heartbeat: timestamp,
          // Pulse animation for heartbeat
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
        return <IconWifi className="text-success" size={20} />;
      case 'offline':
        return <IconWifiOff className="text-muted" size={20} />;
      case 'running_task':
        return <IconActivity className="text-primary spinner" size={20} />;
      default:
        return <IconAlertCircle className="text-warning" size={20} />;
    }
  };

  const formatLastHeartbeat = (heartbeat: string | null) => {
    if (!heartbeat) return t('common.never');
    
    const date = new Date(heartbeat);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSecs = Math.floor(diffMs / 1000);
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    
    if (diffSecs < 10) return <span className="text-green-500">{t('time.just_now')}</span>;
    if (diffMins < 1) return t('time.seconds_ago', { count: diffSecs });
    if (diffMins < 60) return t('time.minutes_ago', { count: diffMins });
    if (diffHours < 24) return t('time.hours_ago', { count: diffHours });
    
    return date.toLocaleString();
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

                  {/* Status Badge */}
                  <div className="mb-3">
                    <span className={classNames(
                      'badge',
                      agent.status === 'online' 
                        ? 'bg-success'
                        : agent.status === 'running_task'
                        ? 'bg-primary'
                        : 'bg-secondary'
                    )}>
                      {t(`agents.status.${agent.status}`)}
                    </span>
                    
                    {agent.current_task && (
                      <span className="badge bg-info ms-1">
                        {agent.current_task}
                      </span>
                    )}
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

                  {/* System Info (if available) */}
                  {agent.system_info && (
                    <div className="mb-3">
                      <div className="mb-2">
                        <div className="d-flex justify-content-between small text-muted">
                          <span>CPU</span>
                          <span>{agent.system_info.cpu_usage}%</span>
                        </div>
                        <div className="progress progress-sm">
                          <div 
                            className="progress-bar bg-primary"
                            style={{ width: `${agent.system_info.cpu_usage}%` }}
                          ></div>
                        </div>
                      </div>
                      <div>
                        <div className="d-flex justify-content-between small text-muted">
                          <span>Memory</span>
                          <span>{agent.system_info.memory_usage}%</span>
                        </div>
                        <div className="progress progress-sm">
                          <div 
                            className="progress-bar bg-info"
                            style={{ width: `${agent.system_info.memory_usage}%` }}
                          ></div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Actions */}
                  <div className="d-flex gap-2">
                    <button
                      onClick={() => syncConfig(agent.id)}
                      className="btn btn-outline-primary btn-sm"
                      title={t('agents.actions.sync_config')}
                      disabled={agent.status === 'offline'}
                    >
                      <IconRefresh size={16} />
                    </button>
                    
                    <button
                      onClick={() => setSelectedAgent(agent)}
                      className="btn btn-outline-secondary btn-sm"
                      title={t('agents.actions.view_details')}
                    >
                      <IconChevronRight size={16} />
                    </button>
                    
                    <button
                      onClick={() => deleteAgent(agent.id, agent.name)}
                      className="btn btn-outline-danger btn-sm"
                      title={t('agents.actions.delete')}
                    >
                      <IconTrash size={16} />
                    </button>
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
        <div className="modal modal-blur fade show" style={{display: 'block'}} tabIndex={-1}>
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
                          docker run -e REGISTRATION_TOKEN={registrationToken} \<br/>
                          &nbsp;&nbsp;-e HUB_URL=http://hub:8080 \<br/>
                          &nbsp;&nbsp;rclone-backup-agent
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

      {/* Agent Detail Modal */}
      {selectedAgent && (
        <div className="modal modal-blur fade show" style={{display: 'block'}} tabIndex={-1}>
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
                {/* Add detailed agent information here */}
                <p className="text-muted">Agent details will be displayed here...</p>
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

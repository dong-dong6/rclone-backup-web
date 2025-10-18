import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  Wifi, WifiOff, Activity, Plus, Copy, Check, 
  RefreshCw, Trash2, Server, Clock, Calendar,
  ChevronRight, AlertCircle, CheckCircle
} from 'lucide-react';
import { apiService } from '../services/api';
import { useSSE } from '../contexts/SSEContext';
import classNames from 'classnames';
import '../styles/neumorphism.css';

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

const AgentsEnhanced: React.FC = () => {
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
      const response = await apiService.getAgents();
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
      const response = await apiService.createRegistrationToken();
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
      await apiService.deleteAgent(id);
      
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
      await apiService.syncAgentConfig(agentId);
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
        return <Wifi className="text-green-500 animate-pulse" size={20} />;
      case 'offline':
        return <WifiOff className="text-gray-400" size={20} />;
      case 'running_task':
        return <Activity className="text-blue-500 animate-spin" size={20} />;
      default:
        return <AlertCircle className="text-yellow-500" size={20} />;
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
      <div className="flex items-center justify-center h-full">
        <div className="neu-card p-8">
          <RefreshCw className="animate-spin text-primary" size={48} />
          <p className="mt-4 text-gray-600">{t('common.loading')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-fadeIn">
      {/* Header with Real-time Stats */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold gradient-text">{t('agents.title')}</h1>
          <p className="text-gray-600 mt-1">{t('agents.subtitle')}</p>
        </div>
        <div className="flex items-center space-x-4">
          {/* Real-time Statistics */}
          <div className="flex space-x-6 mr-8">
            <div className="text-center">
              <div className="text-2xl font-bold">{realtimeStats.totalAgents}</div>
              <div className="text-xs text-gray-500">{t('agents.stats.total')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-500">{realtimeStats.onlineAgents}</div>
              <div className="text-xs text-gray-500">{t('agents.stats.online')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-500">{realtimeStats.runningTasks}</div>
              <div className="text-xs text-gray-500">{t('agents.stats.running')}</div>
            </div>
          </div>
          
          <button 
            onClick={loadAgents}
            className="neu-button flex items-center space-x-2"
          >
            <RefreshCw size={16} />
            <span>{t('common.refresh')}</span>
          </button>
          
          <button 
            onClick={generateToken}
            className="neu-button-primary flex items-center space-x-2"
          >
            <Plus size={16} />
            <span>{t('agents.register.new')}</span>
          </button>
        </div>
      </div>

      {/* Agents Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {agents.map((agent) => (
          <div
            key={agent.id}
            id={`agent-card-${agent.id}`}
            className={classNames(
              'neu-card p-6 space-y-4 transition-all duration-300 hover:scale-105',
              (agent as any).className
            )}
          >
            {/* Agent Header */}
            <div className="flex justify-between items-start">
              <div className="flex items-center space-x-3">
                <Server className="text-primary" size={24} />
                <div>
                  <h3 className="font-semibold text-lg">{agent.name}</h3>
                  <p className="text-xs text-gray-500">{agent.id.substring(0, 8)}...</p>
                </div>
              </div>
              {getStatusIcon(agent.status)}
            </div>

            {/* Status Badge */}
            <div className="flex items-center justify-between">
              <span className={classNames(
                'px-3 py-1 rounded-full text-sm font-medium',
                agent.status === 'online' 
                  ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                  : agent.status === 'running_task'
                  ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
                  : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
              )}>
                {t(`agents.status.${agent.status}`)}
              </span>
              
              {agent.current_task && (
                <span className="text-xs text-blue-600 dark:text-blue-400">
                  {agent.current_task}
                </span>
              )}
            </div>

            {/* Agent Details */}
            <div className="space-y-2 text-sm">
              <div className="flex items-center justify-between text-gray-600 dark:text-gray-400">
                <span className="flex items-center space-x-1">
                  <Clock size={14} />
                  <span>{t('agents.last_heartbeat')}:</span>
                </span>
                <span>{formatLastHeartbeat(agent.last_heartbeat)}</span>
              </div>
              
              <div className="flex items-center justify-between text-gray-600 dark:text-gray-400">
                <span className="flex items-center space-x-1">
                  <Calendar size={14} />
                  <span>{t('agents.registered')}:</span>
                </span>
                <span>{new Date(agent.created_at).toLocaleDateString()}</span>
              </div>
              
              <div className="flex items-center justify-between text-gray-600 dark:text-gray-400">
                <span>{t('agents.assigned_tasks')}:</span>
                <span className="font-semibold">{agent.task_count || 0}</span>
              </div>
            </div>

            {/* System Info (if available) */}
            {agent.system_info && (
              <div className="pt-3 border-t border-gray-200 dark:border-gray-700">
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <span className="text-gray-500">CPU:</span>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 mt-1">
                      <div 
                        className="bg-blue-500 h-2 rounded-full transition-all duration-500"
                        style={{ width: `${agent.system_info.cpu_usage}%` }}
                      />
                    </div>
                  </div>
                  <div>
                    <span className="text-gray-500">Memory:</span>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 mt-1">
                      <div 
                        className="bg-green-500 h-2 rounded-full transition-all duration-500"
                        style={{ width: `${agent.system_info.memory_usage}%` }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Actions */}
            <div className="flex justify-end space-x-2 pt-3">
              <button
                onClick={() => syncConfig(agent.id)}
                className="neu-button-icon"
                title={t('agents.actions.sync_config')}
                disabled={agent.status === 'offline'}
              >
                <RefreshCw size={16} />
              </button>
              
              <button
                onClick={() => setSelectedAgent(agent)}
                className="neu-button-icon"
                title={t('agents.actions.view_details')}
              >
                <ChevronRight size={16} />
              </button>
              
              <button
                onClick={() => deleteAgent(agent.id, agent.name)}
                className="neu-button-icon text-red-500"
                title={t('agents.actions.delete')}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        ))}
      </div>

      {/* Empty State */}
      {agents.length === 0 && (
        <div className="neu-card p-12 text-center">
          <Server className="mx-auto text-gray-400 mb-4" size={64} />
          <h3 className="text-xl font-semibold mb-2">{t('agents.no_agents')}</h3>
          <p className="text-gray-600 mb-6">{t('agents.no_agents_description')}</p>
          <button 
            onClick={generateToken}
            className="neu-button-primary mx-auto"
          >
            {t('agents.register.first_agent')}
          </button>
        </div>
      )}

      {/* Registration Modal */}
      {showRegisterModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 animate-fadeIn">
          <div className="neu-card p-8 max-w-lg w-full animate-slideUp">
            <h2 className="text-2xl font-bold mb-6">{t('agents.register.title')}</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">
                  {t('agents.register.token_label')}
                </label>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={registrationToken}
                    readOnly
                    className="neu-input flex-1 font-mono text-sm"
                  />
                  <button
                    onClick={copyToClipboard}
                    className="neu-button flex items-center space-x-1"
                  >
                    {copied ? <Check size={16} /> : <Copy size={16} />}
                    <span>{copied ? t('common.copied') : t('common.copy')}</span>
                  </button>
                </div>
              </div>

              {tokenExpiry && (
                <div className="bg-yellow-50 dark:bg-yellow-900/20 p-3 rounded-lg">
                  <p className="text-sm text-yellow-800 dark:text-yellow-200">
                    <AlertCircle className="inline mr-1" size={14} />
                    {t('agents.register.token_expires', { 
                      time: tokenExpiry.toLocaleString() 
                    })}
                  </p>
                </div>
              )}

              <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
                <h3 className="font-semibold mb-2">{t('agents.register.instructions')}</h3>
                <ol className="text-sm space-y-2">
                  <li>1. {t('agents.register.step1')}</li>
                  <li>2. {t('agents.register.step2')}</li>
                  <li className="font-mono bg-black text-green-400 p-2 rounded">
                    docker run -e REGISTRATION_TOKEN={registrationToken} \
                    -e HUB_URL=http://hub:8080 \
                    rclone-backup-agent
                  </li>
                  <li>3. {t('agents.register.step3')}</li>
                </ol>
              </div>
            </div>

            <div className="flex justify-end mt-6 space-x-3">
              <button
                onClick={() => {
                  setShowRegisterModal(false);
                  setRegistrationToken('');
                  setTokenExpiry(null);
                }}
                className="neu-button"
              >
                {t('common.close')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Agent Detail Modal */}
      {selectedAgent && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="neu-card p-8 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <h2 className="text-2xl font-bold mb-6">{selectedAgent.name}</h2>
            {/* Add detailed agent information here */}
            <button
              onClick={() => setSelectedAgent(null)}
              className="neu-button mt-6"
            >
              {t('common.close')}
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default AgentsEnhanced;
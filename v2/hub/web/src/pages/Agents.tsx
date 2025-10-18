import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { apiService } from '../services/api';
import { useSSE } from '../contexts/SSEContext';
import '../styles/neumorphism.css';

interface Agent {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'running_task';
  last_heartbeat: string | null;
  created_at: string;
  task_count?: number;
}

const Agents: React.FC = () => {
  const { t } = useTranslation();
  const { subscribe } = useSSE();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [registrationToken, setRegistrationToken] = useState('');
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);

  useEffect(() => {
    loadAgents();
    
    // Subscribe to agent status updates
    const unsubscribe = subscribe('agent.status.update', (event) => {
      const { agent_id, status } = event.data;
      setAgents(prev => prev.map(agent => 
        agent.id === agent_id ? { ...agent, status } : agent
      ));
    });
    
    return () => {
      unsubscribe();
    };
  }, []);

  const loadAgents = async () => {
    try {
      setLoading(true);
      const response = await apiService.getAgents();
      setAgents(response.data);
    } catch (error) {
      console.error('Failed to load agents:', error);
    } finally {
      setLoading(false);
    }
  };

  const generateToken = async () => {
    try {
      const response = await apiService.createRegistrationToken();
      setRegistrationToken(response.data.token);
      setShowRegisterModal(true);
    } catch (error) {
      console.error('Failed to generate token:', error);
    }
  };

  const deleteAgent = async (id: string, name: string) => {
    if (!confirm(t('agents.actions.deleteConfirm', { name }))) {
      return;
    }
    
    try {
      await apiService.deleteAgent(id);
      await loadAgents();
    } catch (error) {
      console.error('Failed to delete agent:', error);
    }
  };

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      online: { class: 'neu-badge-success', icon: '🟢' },
      offline: { class: 'neu-badge-error', icon: '🔴' },
      running_task: { class: 'neu-badge-info', icon: '🔄' },
    };
    
    const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.offline;
    
    return (
      <span className={`neu-badge ${config.class}`}>
        {config.icon} {t(`agents.status.${status}`)}
      </span>
    );
  };

  const formatLastHeartbeat = (heartbeat: string | null) => {
    if (!heartbeat) return t('app.never');
    
    const date = new Date(heartbeat);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return t('time.just_now');
    if (diffMins < 60) return t('time.minutes_ago', { count: diffMins });
    
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return t('time.hours_ago', { count: diffHours });
    
    const diffDays = Math.floor(diffHours / 24);
    return t('time.days_ago', { count: diffDays });
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    alert(t('app.copied'));
  };

  return (
    <div className="agents-page">
      <div className="page-header">
        <h1>{t('agents.title')}</h1>
        <button className="neu-button neu-button-primary" onClick={generateToken}>
          + {t('agents.register.generateToken')}
        </button>
      </div>

      {loading ? (
        <div className="loading-spinner">
          <div className="neu-card">
            <p>{t('app.loading')}</p>
          </div>
        </div>
      ) : agents.length === 0 ? (
        <div className="empty-state neu-card">
          <p>{t('agents.list.empty')}</p>
          <button className="neu-button" onClick={generateToken}>
            {t('agents.register.title')}
          </button>
        </div>
      ) : (
        <div className="agents-grid">
          {agents.map(agent => (
            <div key={agent.id} className="neu-card agent-card">
              <div className="agent-header">
                <h3>{agent.name}</h3>
                {getStatusBadge(agent.status)}
              </div>
              
              <div className="agent-info">
                <div className="info-row">
                  <span className="label">{t('agents.list.columns.lastHeartbeat')}:</span>
                  <span className="value">{formatLastHeartbeat(agent.last_heartbeat)}</span>
                </div>
                <div className="info-row">
                  <span className="label">{t('agents.list.columns.tasks')}:</span>
                  <span className="value">{agent.task_count || 0}</span>
                </div>
                <div className="info-row">
                  <span className="label">{t('agents.list.columns.createdAt')}:</span>
                  <span className="value">{new Date(agent.created_at).toLocaleDateString()}</span>
                </div>
              </div>
              
              <div className="agent-actions">
                <button 
                  className="neu-button"
                  onClick={() => setSelectedAgent(agent)}
                >
                  {t('app.details')}
                </button>
                <button 
                  className="neu-button neu-button-danger"
                  onClick={() => deleteAgent(agent.id, agent.name)}
                >
                  {t('app.delete')}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Registration Modal */}
      {showRegisterModal && (
        <div className="modal-overlay" onClick={() => setShowRegisterModal(false)}>
          <div className="modal neu-card" onClick={e => e.stopPropagation()}>
            <h2>{t('agents.register.tokenGenerated')}</h2>
            <p>{t('agents.register.tokenHint')}</p>
            
            <div className="token-display neu-card-inset">
              <code>{registrationToken}</code>
            </div>
            
            <div className="modal-actions">
              <button 
                className="neu-button neu-button-primary"
                onClick={() => copyToClipboard(registrationToken)}
              >
                {t('agents.register.copyToken')}
              </button>
              <button 
                className="neu-button"
                onClick={() => setShowRegisterModal(false)}
              >
                {t('app.close')}
              </button>
            </div>
            
            <div className="registration-instructions">
              <h3>{t('agents.register.instructions')}</h3>
              <pre className="neu-card-inset">
{`curl -X POST ${window.location.origin}/api/v1/agent/register \\
  -H "Content-Type: application/json" \\
  -d '{
    "token": "${registrationToken}",
    "name": "your-agent-name"
  }'`}
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* Agent Details Modal */}
      {selectedAgent && (
        <div className="modal-overlay" onClick={() => setSelectedAgent(null)}>
          <div className="modal neu-card large" onClick={e => e.stopPropagation()}>
            <h2>{t('agents.detail.title')}: {selectedAgent.name}</h2>
            
            <div className="detail-sections">
              <section className="neu-card-flat">
                <h3>{t('agents.detail.info')}</h3>
                <div className="info-grid">
                  <div>
                    <span className="label">ID:</span>
                    <span className="value">{selectedAgent.id}</span>
                  </div>
                  <div>
                    <span className="label">{t('agents.list.columns.status')}:</span>
                    <span className="value">{getStatusBadge(selectedAgent.status)}</span>
                  </div>
                  <div>
                    <span className="label">{t('agents.list.columns.lastHeartbeat')}:</span>
                    <span className="value">{formatLastHeartbeat(selectedAgent.last_heartbeat)}</span>
                  </div>
                  <div>
                    <span className="label">{t('agents.list.columns.createdAt')}:</span>
                    <span className="value">{new Date(selectedAgent.created_at).toLocaleString()}</span>
                  </div>
                </div>
              </section>
              
              <section className="neu-card-flat">
                <h3>{t('agents.detail.assignedTasks')}</h3>
                <p className="placeholder">{t('app.loading')}...</p>
              </section>
              
              <section className="neu-card-flat">
                <h3>{t('agents.detail.executionHistory')}</h3>
                <p className="placeholder">{t('app.loading')}...</p>
              </section>
            </div>
            
            <div className="modal-actions">
              <button 
                className="neu-button"
                onClick={() => setSelectedAgent(null)}
              >
                {t('app.close')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Agents;
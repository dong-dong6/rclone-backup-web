import { useState, useEffect, useCallback, useRef } from 'react';
import { useSSE } from '../../../contexts/SSEContext';
import { agentsApi } from '../../../services';
import type { Agent, AgentStats, AgentMetric } from '../../../types';

interface HeartbeatEvent {
  agent_id: string;
  status: string;
  timestamp: string;
  actions?: number;
  metrics?: AgentMetric;
}

export function useAgents() {
  const { subscribe } = useSSE();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<AgentStats>({
    total: 0,
    online: 0,
    running: 0,
  });

  const loadAgents = useCallback(async () => {
    try {
      setLoading(true);
      const data = await agentsApi.getAll();
      setAgents(data);
    } catch (error) {
      console.error('Failed to load agents:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  const deleteAgent = useCallback(async (id: string): Promise<boolean> => {
    try {
      await agentsApi.delete(id);
      setAgents(prev => prev.filter(agent => agent.id !== id));
      return true;
    } catch (error) {
      console.error('Failed to delete agent:', error);
      await loadAgents();
      return false;
    }
  }, [loadAgents]);

  const updateAgent = useCallback(async (id: string, name: string): Promise<boolean> => {
    try {
      await agentsApi.update(id, { name });
      setAgents(prev => prev.map(agent =>
        agent.id === id ? { ...agent, name } : agent
      ));
      return true;
    } catch (error) {
      console.error('Failed to update agent:', error);
      return false;
    }
  }, []);

  const syncConfig = useCallback(async (id: string): Promise<boolean> => {
    try {
      await agentsApi.syncConfig(id);
      return true;
    } catch (error) {
      console.error('Failed to sync config:', error);
      return false;
    }
  }, []);

  // Update stats whenever agents change
  useEffect(() => {
    setStats({
      total: agents.length,
      online: agents.filter(a => a.status === 'online').length,
      running: agents.filter(a => a.status === 'running_task').length,
    });
  }, [agents]);

  // Load agents and setup SSE listeners
  useEffect(() => {
    loadAgents();

    const unsubscribers = [
      subscribe('agent.status.update', (event: { data: { agent_id: string; status: string } }) => {
        const { agent_id, status } = event.data;
        setAgents(prev => prev.map(agent =>
          agent.id === agent_id
            ? { ...agent, status: status as Agent['status'], last_heartbeat: new Date().toISOString() }
            : agent
        ));
      }),

      subscribe('task.dispatched', (event: { data: { agent_id: string; task_name: string } }) => {
        const { agent_id, task_name } = event.data;
        setAgents(prev => prev.map(agent =>
          agent.id === agent_id
            ? { ...agent, status: 'running_task', current_task: task_name }
            : agent
        ));
      }),

      subscribe('agent.heartbeat', (event: { data: HeartbeatEvent }) => {
        const { agent_id, timestamp } = event.data;
        setAgents(prev => prev.map(agent =>
          agent.id === agent_id
            ? { ...agent, last_heartbeat: timestamp }
            : agent
        ));
      }),

      subscribe('agent.registered', () => {
        loadAgents();
      }),
    ];

    return () => {
      unsubscribers.forEach(unsubscribe => unsubscribe());
    };
  }, []);

  return {
    agents,
    stats,
    loading,
    loadAgents,
    deleteAgent,
    updateAgent,
    syncConfig,
  };
}

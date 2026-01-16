import { useState, useEffect, useCallback } from 'react';
import { useSSE } from '../../../contexts/SSEContext';
import { tasksApi, agentsApi, remotesApi } from '../../../services';
import type { BackupTask, Agent, RcloneRemote } from '../../../types';

export function useTasks() {
  const { subscribe } = useSSE();
  const [tasks, setTasks] = useState<BackupTask[]>([]);
  const [remotes, setRemotes] = useState<RcloneRemote[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchTasks = useCallback(async () => {
    try {
      const data = await tasksApi.getAll();
      setTasks(data);
    } catch (error) {
      console.error('Failed to fetch tasks:', error);
    }
  }, []);

  const fetchRemotes = useCallback(async () => {
    try {
      const data = await remotesApi.getAll();
      setRemotes(data);
    } catch (error) {
      console.error('Failed to fetch remotes:', error);
    }
  }, []);

  const fetchAgents = useCallback(async () => {
    try {
      const data = await agentsApi.getAll();
      setAgents(data);
    } catch (error) {
      console.error('Failed to fetch agents:', error);
    }
  }, []);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      await Promise.all([fetchTasks(), fetchRemotes(), fetchAgents()]);
    } finally {
      setLoading(false);
    }
  }, [fetchTasks, fetchRemotes, fetchAgents]);

  const deleteTask = useCallback(async (id: string): Promise<boolean> => {
    try {
      await tasksApi.delete(id);
      setTasks(prev => prev.filter(t => t.id !== id));
      return true;
    } catch (error) {
      console.error('Failed to delete task:', error);
      return false;
    }
  }, []);

  const toggleTaskActive = useCallback(async (task: BackupTask): Promise<boolean> => {
    try {
      await tasksApi.update(task.id, { is_active: !task.is_active });
      setTasks(prev => prev.map(t =>
        t.id === task.id ? { ...t, is_active: !t.is_active } : t
      ));
      return true;
    } catch (error) {
      console.error('Failed to toggle task:', error);
      return false;
    }
  }, []);

  const triggerTask = useCallback(async (taskId: string, agentId: string): Promise<boolean> => {
    try {
      await tasksApi.trigger(taskId, agentId);
      return true;
    } catch (error) {
      console.error('Failed to trigger task:', error);
      return false;
    }
  }, []);

  // Load data and setup SSE listeners
  useEffect(() => {
    fetchAll();

    const unsubscribers = [
      subscribe('task.created', fetchTasks),
      subscribe('task.updated', fetchTasks),
      subscribe('task.deleted', fetchTasks),
    ];

    return () => {
      unsubscribers.forEach(unsubscribe => unsubscribe());
    };
  }, []);

  const getAgentName = useCallback((agentId: string) => {
    const agent = agents.find(a => a.id === agentId);
    return agent ? agent.name : agentId;
  }, [agents]);

  const getAgentStatus = useCallback((agentId: string) => {
    const agent = agents.find(a => a.id === agentId);
    return agent ? agent.status : 'offline';
  }, [agents]);

  const getRemoteName = useCallback((remoteId: string) => {
    const remote = remotes.find(r => r.id === remoteId);
    return remote ? remote.name : remoteId;
  }, [remotes]);

  const getRemoteType = useCallback((remoteId: string) => {
    const remote = remotes.find(r => r.id === remoteId);
    return remote?.type || '';
  }, [remotes]);

  return {
    tasks,
    remotes,
    agents,
    loading,
    fetchAll,
    fetchTasks,
    deleteTask,
    toggleTaskActive,
    triggerTask,
    getAgentName,
    getAgentStatus,
    getRemoteName,
    getRemoteType,
  };
}

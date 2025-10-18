import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Edit, Trash2, Play, Pause, Calendar, Clock } from 'lucide-react';
import { apiClient } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { useSSE } from '../contexts/SSEContext';
import classNames from 'classnames';

interface RcloneRemote {
  id: string;
  name: string;
  config_data: string;
  created_at: string;
}

interface Agent {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'running_task';
  last_heartbeat: string;
}

interface BackupTask {
  id: string;
  name: string;
  rclone_remote_id: string;
  remote_name?: string;
  source_path: string;
  destination_path: string;
  schedule: string;
  rclone_args: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
  assigned_agents: string[];
  next_run?: string;
  last_run?: string;
}

const Tasks: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const { events } = useSSE();
  const [tasks, setTasks] = useState<BackupTask[]>([]);
  const [remotes, setRemotes] = useState<RcloneRemote[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingTask, setEditingTask] = useState<BackupTask | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    rclone_remote_id: '',
    source_path: '',
    destination_path: '',
    schedule: '0 2 * * *', // Default: daily at 2 AM
    rclone_args: [] as string[],
    is_active: true,
    assigned_agent_ids: [] as string[],
  });

  // Cron expression presets
  const cronPresets = [
    { label: t('tasks.schedule.every_hour'), value: '0 * * * *' },
    { label: t('tasks.schedule.daily_2am'), value: '0 2 * * *' },
    { label: t('tasks.schedule.weekly_sunday'), value: '0 2 * * 0' },
    { label: t('tasks.schedule.monthly_first'), value: '0 2 1 * *' },
    { label: t('tasks.schedule.custom'), value: 'custom' },
  ];

  // Common rclone arguments
  const rcloneArgPresets = [
    { label: '--dry-run', description: t('tasks.args.dry_run') },
    { label: '--verbose', description: t('tasks.args.verbose') },
    { label: '--checksum', description: t('tasks.args.checksum') },
    { label: '--delete-after', description: t('tasks.args.delete_after') },
    { label: '--exclude *.tmp', description: t('tasks.args.exclude_tmp') },
  ];

  useEffect(() => {
    fetchData();
  }, []);

  // Listen to SSE events
  useEffect(() => {
    const handleTaskEvent = (event: any) => {
      if (event.type === 'task.created' || event.type === 'task.updated') {
        fetchTasks();
      }
    };

    events.forEach(handleTaskEvent);
  }, [events]);

  const fetchData = async () => {
    setLoading(true);
    try {
      await Promise.all([
        fetchTasks(),
        fetchRemotes(),
        fetchAgents(),
      ]);
    } finally {
      setLoading(false);
    }
  };

  const fetchTasks = async () => {
    try {
      const response = await apiClient.get('/admin/tasks', {
        headers: { Authorization: `Bearer ${token}` },
      });
      setTasks(response.data);
    } catch (error) {
      console.error('Failed to fetch tasks:', error);
    }
  };

  const fetchRemotes = async () => {
    try {
      const response = await apiClient.get('/admin/remotes', {
        headers: { Authorization: `Bearer ${token}` },
      });
      setRemotes(response.data);
    } catch (error) {
      console.error('Failed to fetch remotes:', error);
    }
  };

  const fetchAgents = async () => {
    try {
      const response = await apiClient.get('/admin/agents', {
        headers: { Authorization: `Bearer ${token}` },
      });
      setAgents(response.data);
    } catch (error) {
      console.error('Failed to fetch agents:', error);
    }
  };

  const handleCreateTask = () => {
    setEditingTask(null);
    setFormData({
      name: '',
      rclone_remote_id: '',
      source_path: '',
      destination_path: '',
      schedule: '0 2 * * *',
      rclone_args: [],
      is_active: true,
      assigned_agent_ids: [],
    });
    setShowCreateModal(true);
  };

  const handleEditTask = (task: BackupTask) => {
    setEditingTask(task);
    setFormData({
      name: task.name,
      rclone_remote_id: task.rclone_remote_id,
      source_path: task.source_path,
      destination_path: task.destination_path,
      schedule: task.schedule,
      rclone_args: task.rclone_args,
      is_active: task.is_active,
      assigned_agent_ids: task.assigned_agents,
    });
    setShowCreateModal(true);
  };

  const handleDeleteTask = async (taskId: string) => {
    if (!confirm(t('tasks.confirm_delete'))) return;

    try {
      await apiClient.delete(`/admin/tasks/${taskId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      fetchTasks();
    } catch (error) {
      console.error('Failed to delete task:', error);
      alert(t('tasks.delete_failed'));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      if (editingTask) {
        await apiClient.put(`/admin/tasks/${editingTask.id}`, formData, {
          headers: { Authorization: `Bearer ${token}` },
        });
      } else {
        await apiClient.post('/admin/tasks', formData, {
          headers: { Authorization: `Bearer ${token}` },
        });
      }
      
      setShowCreateModal(false);
      fetchTasks();
    } catch (error) {
      console.error('Failed to save task:', error);
      alert(t('tasks.save_failed'));
    }
  };

  const handleTriggerTask = async (taskId: string, agentId: string) => {
    try {
      await apiClient.post('/admin/executions/trigger', 
        { task_id: taskId, agent_id: agentId },
        { headers: { Authorization: `Bearer ${token}` } }
      );
      alert(t('tasks.triggered_success'));
    } catch (error) {
      console.error('Failed to trigger task:', error);
      alert(t('tasks.trigger_failed'));
    }
  };

  const toggleTaskActive = async (task: BackupTask) => {
    try {
      await apiClient.put(`/admin/tasks/${task.id}`, 
        { ...task, is_active: !task.is_active },
        { headers: { Authorization: `Bearer ${token}` } }
      );
      fetchTasks();
    } catch (error) {
      console.error('Failed to toggle task:', error);
    }
  };

  const formatNextRun = (schedule: string) => {
    // This would be calculated server-side ideally
    return t('tasks.calculating');
  };

  const getAgentName = (agentId: string) => {
    const agent = agents.find(a => a.id === agentId);
    return agent ? agent.name : agentId;
  };

  const getAgentStatus = (agentId: string) => {
    const agent = agents.find(a => a.id === agentId);
    return agent ? agent.status : 'offline';
  };

  if (loading) {
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
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">{t('tasks.title')}</h1>
        <button
          onClick={handleCreateTask}
          className="neu-button-primary flex items-center space-x-2"
        >
          <Plus size={20} />
          <span>{t('tasks.create_new')}</span>
        </button>
      </div>

      {/* Tasks Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        {tasks.map((task) => (
          <div key={task.id} className="neu-card p-6 space-y-4">
            <div className="flex justify-between items-start">
              <div className="flex-1">
                <h3 className="text-xl font-semibold mb-1">{task.name}</h3>
                <span className={classNames(
                  'inline-flex items-center px-2 py-1 rounded text-sm',
                  task.is_active 
                    ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200' 
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                )}>
                  {task.is_active ? t('common.active') : t('common.inactive')}
                </span>
              </div>
              
              <div className="flex space-x-1">
                <button
                  onClick={() => toggleTaskActive(task)}
                  className="neu-button-icon"
                  title={task.is_active ? t('tasks.deactivate') : t('tasks.activate')}
                >
                  {task.is_active ? <Pause size={16} /> : <Play size={16} />}
                </button>
                <button
                  onClick={() => handleEditTask(task)}
                  className="neu-button-icon"
                  title={t('common.edit')}
                >
                  <Edit size={16} />
                </button>
                <button
                  onClick={() => handleDeleteTask(task.id)}
                  className="neu-button-icon text-red-500"
                  title={t('common.delete')}
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>

            <div className="space-y-2 text-sm">
              <div className="flex items-center space-x-2 text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('tasks.remote')}:</span>
                <span>{task.remote_name || task.rclone_remote_id}</span>
              </div>
              
              <div className="text-gray-600 dark:text-gray-400">
                <div className="font-medium">{t('tasks.source')}:</div>
                <div className="text-xs font-mono bg-gray-100 dark:bg-gray-800 p-1 rounded mt-1">
                  {task.source_path}
                </div>
              </div>
              
              <div className="text-gray-600 dark:text-gray-400">
                <div className="font-medium">{t('tasks.destination')}:</div>
                <div className="text-xs font-mono bg-gray-100 dark:bg-gray-800 p-1 rounded mt-1">
                  {task.destination_path}
                </div>
              </div>
              
              <div className="flex items-center space-x-2 text-gray-600 dark:text-gray-400">
                <Calendar size={14} />
                <span className="font-medium">{t('tasks.schedule')}:</span>
                <span className="font-mono text-xs">{task.schedule}</span>
              </div>
              
              <div className="flex items-center space-x-2 text-gray-600 dark:text-gray-400">
                <Clock size={14} />
                <span className="font-medium">{t('tasks.next_run')}:</span>
                <span>{task.next_run || formatNextRun(task.schedule)}</span>
              </div>
            </div>

            {/* Assigned Agents */}
            <div>
              <div className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-2">
                {t('tasks.assigned_agents')}:
              </div>
              {task.assigned_agents.length > 0 ? (
                <div className="space-y-1">
                  {task.assigned_agents.map(agentId => {
                    const status = getAgentStatus(agentId);
                    return (
                      <div key={agentId} className="flex justify-between items-center">
                        <span className="text-sm">{getAgentName(agentId)}</span>
                        <div className="flex items-center space-x-2">
                          <span className={classNames(
                            'w-2 h-2 rounded-full',
                            status === 'online' ? 'bg-green-500' :
                            status === 'running_task' ? 'bg-blue-500' :
                            'bg-gray-400'
                          )} />
                          {status === 'online' && (
                            <button
                              onClick={() => handleTriggerTask(task.id, agentId)}
                              className="text-xs neu-button px-2 py-1"
                              title={t('tasks.run_now')}
                            >
                              <Play size={12} />
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="text-sm text-gray-500 italic">
                  {t('tasks.no_agents_assigned')}
                </div>
              )}
            </div>

            {/* Rclone Args */}
            {task.rclone_args.length > 0 && (
              <div>
                <div className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">
                  {t('tasks.arguments')}:
                </div>
                <div className="flex flex-wrap gap-1">
                  {task.rclone_args.map((arg, idx) => (
                    <span key={idx} className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-1 rounded">
                      {arg}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {tasks.length === 0 && (
        <div className="neu-card p-12 text-center">
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            {t('tasks.no_tasks')}
          </p>
          <button
            onClick={handleCreateTask}
            className="neu-button-primary"
          >
            {t('tasks.create_first')}
          </button>
        </div>
      )}

      {/* Create/Edit Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="neu-card p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <h2 className="text-2xl font-bold mb-6">
              {editingTask ? t('tasks.edit_task') : t('tasks.create_task')}
            </h2>
            
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('tasks.task_name')}
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="neu-input w-full"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('tasks.remote')}
                </label>
                <select
                  value={formData.rclone_remote_id}
                  onChange={(e) => setFormData({ ...formData, rclone_remote_id: e.target.value })}
                  className="neu-input w-full"
                  required
                >
                  <option value="">{t('tasks.select_remote')}</option>
                  {remotes.map((remote) => (
                    <option key={remote.id} value={remote.id}>
                      {remote.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1">
                    {t('tasks.source_path')}
                  </label>
                  <input
                    type="text"
                    value={formData.source_path}
                    onChange={(e) => setFormData({ ...formData, source_path: e.target.value })}
                    placeholder="/path/to/source"
                    className="neu-input w-full"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">
                    {t('tasks.destination_path')}
                  </label>
                  <input
                    type="text"
                    value={formData.destination_path}
                    onChange={(e) => setFormData({ ...formData, destination_path: e.target.value })}
                    placeholder="remote:path/to/dest"
                    className="neu-input w-full"
                    required
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('tasks.schedule')}
                </label>
                <div className="space-y-2">
                  <select
                    value={cronPresets.find(p => p.value === formData.schedule) ? formData.schedule : 'custom'}
                    onChange={(e) => {
                      if (e.target.value !== 'custom') {
                        setFormData({ ...formData, schedule: e.target.value });
                      }
                    }}
                    className="neu-input w-full"
                  >
                    {cronPresets.map((preset) => (
                      <option key={preset.value} value={preset.value}>
                        {preset.label}
                      </option>
                    ))}
                  </select>
                  <input
                    type="text"
                    value={formData.schedule}
                    onChange={(e) => setFormData({ ...formData, schedule: e.target.value })}
                    placeholder="* * * * *"
                    className="neu-input w-full font-mono"
                    required
                  />
                  <div className="text-xs text-gray-500">
                    {t('tasks.cron_help')}
                  </div>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('tasks.assign_agents')}
                </label>
                <div className="space-y-2">
                  {agents.map((agent) => (
                    <label key={agent.id} className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        checked={formData.assigned_agent_ids.includes(agent.id)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setFormData({
                              ...formData,
                              assigned_agent_ids: [...formData.assigned_agent_ids, agent.id],
                            });
                          } else {
                            setFormData({
                              ...formData,
                              assigned_agent_ids: formData.assigned_agent_ids.filter(id => id !== agent.id),
                            });
                          }
                        }}
                        className="neu-checkbox"
                      />
                      <span>{agent.name}</span>
                      <span className={classNames(
                        'text-xs px-2 py-1 rounded',
                        agent.status === 'online' ? 'bg-green-100 text-green-800' :
                        agent.status === 'running_task' ? 'bg-blue-100 text-blue-800' :
                        'bg-gray-100 text-gray-600'
                      )}>
                        {agent.status}
                      </span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('tasks.rclone_arguments')}
                </label>
                <div className="space-y-2">
                  {rcloneArgPresets.map((arg) => (
                    <label key={arg.label} className="flex items-start space-x-2">
                      <input
                        type="checkbox"
                        checked={formData.rclone_args.includes(arg.label)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setFormData({
                              ...formData,
                              rclone_args: [...formData.rclone_args, arg.label],
                            });
                          } else {
                            setFormData({
                              ...formData,
                              rclone_args: formData.rclone_args.filter(a => a !== arg.label),
                            });
                          }
                        }}
                        className="neu-checkbox mt-1"
                      />
                      <div>
                        <span className="font-mono text-sm">{arg.label}</span>
                        <div className="text-xs text-gray-500">{arg.description}</div>
                      </div>
                    </label>
                  ))}
                  <input
                    type="text"
                    placeholder={t('tasks.custom_args')}
                    onKeyPress={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        const input = e.currentTarget;
                        if (input.value && !formData.rclone_args.includes(input.value)) {
                          setFormData({
                            ...formData,
                            rclone_args: [...formData.rclone_args, input.value],
                          });
                          input.value = '';
                        }
                      }
                    }}
                    className="neu-input w-full text-sm"
                  />
                  {formData.rclone_args.length > 0 && (
                    <div className="flex flex-wrap gap-2">
                      {formData.rclone_args.map((arg, idx) => (
                        <span
                          key={idx}
                          className="inline-flex items-center space-x-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-1 rounded text-sm"
                        >
                          <span className="font-mono">{arg}</span>
                          <button
                            type="button"
                            onClick={() => setFormData({
                              ...formData,
                              rclone_args: formData.rclone_args.filter((_, i) => i !== idx),
                            })}
                            className="ml-1 hover:text-red-500"
                          >
                            ×
                          </button>
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="is_active"
                  checked={formData.is_active}
                  onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                  className="neu-checkbox"
                />
                <label htmlFor="is_active" className="text-sm font-medium">
                  {t('tasks.activate_immediately')}
                </label>
              </div>

              <div className="flex justify-end space-x-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="neu-button"
                >
                  {t('common.cancel')}
                </button>
                <button
                  type="submit"
                  className="neu-button-primary"
                >
                  {editingTask ? t('common.save') : t('common.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Tasks;
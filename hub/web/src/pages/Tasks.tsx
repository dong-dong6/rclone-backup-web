import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconPlus,
  IconEdit,
  IconTrash,
  IconPlayerPlay,
  IconPlayerPause,
  IconCalendar,
  IconClock,
  IconFolder,
  IconRefresh,
  IconArrowUp,
} from '@tabler/icons-react';
import { apiClient } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { useSSE } from '../contexts/SSEContext';

interface RcloneRemote {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  type?: string;
}

interface Agent {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'running_task';
  last_heartbeat: string;
  is_local?: boolean;
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
  backup_mode: 'sync' | 'archive';
  archive_format?: 'tar.gz' | 'zip';
  encryption_enabled?: boolean;
  max_retention?: number | null;
  created_at: string;
  updated_at: string;
  assigned_agents: string[];
  next_run?: string;
  last_run?: string;
}

type FSListEntry = {
  name: string;
  path: string;
  is_dir: boolean;
  is_symlink?: boolean;
};

type FSListResponse = {
  path: string;
  parent?: string;
  entries: FSListEntry[];
};

  const Tasks: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const { subscribe } = useSSE();
  const [tasks, setTasks] = useState<BackupTask[]>([]);
  const [remotes, setRemotes] = useState<RcloneRemote[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingTask, setEditingTask] = useState<BackupTask | null>(null);
  const [sourceBrowserOpen, setSourceBrowserOpen] = useState(false);
  const [sourceBrowserAgentId, setSourceBrowserAgentId] = useState<string | null>(null);
  const [sourceBrowserPath, setSourceBrowserPath] = useState('');
  const [sourceBrowserParent, setSourceBrowserParent] = useState<string>('');
  const [sourceBrowserEntries, setSourceBrowserEntries] = useState<FSListEntry[]>([]);
  const [sourceBrowserLoading, setSourceBrowserLoading] = useState(false);
  const [sourceBrowserError, setSourceBrowserError] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    rclone_remote_id: '',
    source_path: '',
    destination_path: '',
    schedule: '0 2 * * *', // Default: daily at 2 AM
    rclone_args: [] as string[],
    is_active: true,
    backup_mode: 'sync' as 'sync' | 'archive',
    archive_format: 'tar.gz' as 'tar.gz' | 'zip',
    encryption_enabled: false,
    encryption_password: '',
    max_retention: '',
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
  const baseRcloneArgPresets = [
    { label: '--dry-run', description: t('tasks.args.dry_run') },
    { label: '--verbose', description: t('tasks.args.verbose') },
    { label: '--checksum', description: t('tasks.args.checksum') },
    { label: '--delete-after', description: t('tasks.args.delete_after') },
    { label: '--exclude *.tmp', description: t('tasks.args.exclude_tmp') },
  ];

  const selectedRemote = remotes.find(r => r.id === formData.rclone_remote_id);
  const isS3Remote = (selectedRemote?.type || '').toLowerCase() === 's3';

  const rcloneArgPresets = [
    ...baseRcloneArgPresets,
    ...(isS3Remote
      ? [
          {
            label: '--s3-no-check-bucket',
            description: t('tasks.args.s3_no_check_bucket'),
          },
        ]
      : []),
  ];

  useEffect(() => {
    if (!isS3Remote) return;

    const flag = '--s3-no-check-bucket';
    setFormData(current => {
      if (current.rclone_args.includes(flag)) return current;
      return { ...current, rclone_args: [...current.rclone_args, flag] };
    });
  }, [isS3Remote]);

  useEffect(() => {
    fetchData();
  }, []);

  // Listen to SSE events
  useEffect(() => {
    const unsubscribers = [
      subscribe('task.created', () => fetchTasks()),
      subscribe('task.updated', () => fetchTasks()),
      subscribe('task.deleted', () => fetchTasks()),
    ];

    return () => {
      unsubscribers.forEach(unsubscribe => unsubscribe());
    };
  }, []);

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
      backup_mode: 'sync',
      archive_format: 'tar.gz',
      encryption_enabled: false,
      encryption_password: '',
      max_retention: '',
      assigned_agent_ids: [],
    });
    setSourceBrowserOpen(false);
    setSourceBrowserAgentId(null);
    setSourceBrowserPath('');
    setSourceBrowserParent('');
    setSourceBrowserEntries([]);
    setSourceBrowserLoading(false);
    setSourceBrowserError(null);
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
      backup_mode: task.backup_mode || 'sync',
      archive_format: task.archive_format || 'tar.gz',
      encryption_enabled: !!task.encryption_enabled,
      encryption_password: '',
      max_retention: task.max_retention ? String(task.max_retention) : '',
      assigned_agent_ids: task.assigned_agents,
    });
    setSourceBrowserOpen(false);
    setSourceBrowserAgentId(null);
    setSourceBrowserPath('');
    setSourceBrowserParent('');
    setSourceBrowserEntries([]);
    setSourceBrowserLoading(false);
    setSourceBrowserError(null);
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

  const fetchAgentDirectory = async (agentId: string, path: string) => {
    setSourceBrowserLoading(true);
    setSourceBrowserError(null);

    try {
      const response = await apiClient.get<FSListResponse>(`/admin/agents/${agentId}/fs/list`, {
        params: { path, limit: 200 },
        headers: { Authorization: `Bearer ${token}` },
      });

      setSourceBrowserPath(response.data.path || path);
      setSourceBrowserParent(response.data.parent || '');
      setSourceBrowserEntries(response.data.entries || []);
    } catch (error: any) {
      const message = error?.response?.data?.message || error?.response?.data?.error || t('errors.server');
      setSourceBrowserError(message);
      setSourceBrowserEntries([]);
      setSourceBrowserParent('');
    } finally {
      setSourceBrowserLoading(false);
    }
  };

  const openSourceBrowser = () => {
    setSourceBrowserOpen(true);
    setSourceBrowserError(null);
    setSourceBrowserEntries([]);
    setSourceBrowserParent('');

    const selectedAgentId = formData.assigned_agent_ids[0];
    if (!selectedAgentId) {
      setSourceBrowserAgentId(null);
      setSourceBrowserPath(formData.source_path || '/');
      setSourceBrowserError(t('tasks.browse.select_agent_first'));
      return;
    }

    setSourceBrowserAgentId(selectedAgentId);
    const initialPath = (formData.source_path || '/').trim() || '/';
    setSourceBrowserPath(initialPath);
    void fetchAgentDirectory(selectedAgentId, initialPath);
  };

  const closeSourceBrowser = () => {
    if (sourceBrowserLoading) return;
    setSourceBrowserOpen(false);
    setSourceBrowserAgentId(null);
    setSourceBrowserPath('');
    setSourceBrowserParent('');
    setSourceBrowserEntries([]);
    setSourceBrowserError(null);
  };

  const navigateSourceBrowser = (path: string) => {
    if (!sourceBrowserAgentId || sourceBrowserLoading) return;
    const next = path.trim();
    if (!next) return;
    setSourceBrowserPath(next);
    void fetchAgentDirectory(sourceBrowserAgentId, next);
  };

  const applySourceBrowserPath = () => {
    const next = sourceBrowserPath.trim();
    if (!next) return;
    setFormData({ ...formData, source_path: next });
    closeSourceBrowser();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      if (!formData.destination_path.trim()) {
        alert(t('tasks.destination_path_required'));
        return;
      }
      const dest = formData.destination_path.trim();
      const looksLikeRemotePrefix =
        /^[^\\/]+:/.test(dest) && !/^[A-Za-z]:[\\/]/.test(dest);
      if (looksLikeRemotePrefix) {
        alert(t('tasks.destination_path_no_remote'));
        return;
      }
      if (formData.encryption_enabled && !editingTask && !formData.encryption_password.trim()) {
        alert(t('tasks.encryption_password_required'));
        return;
      }

      const payload: any = {
        name: formData.name,
        rclone_remote_id: formData.rclone_remote_id,
        source_path: formData.source_path,
        destination_path: formData.destination_path,
        schedule: formData.schedule,
        rclone_args: formData.rclone_args,
        is_active: formData.is_active,
        assigned_agent_ids: formData.assigned_agent_ids,
        backup_mode: formData.backup_mode,
        archive_format: formData.archive_format,
        encryption_enabled: formData.encryption_enabled,
      };
      if (formData.encryption_enabled && formData.encryption_password.trim()) {
        payload.encryption_password = formData.encryption_password.trim();
      }
      if (formData.backup_mode === 'archive') {
        const raw = formData.max_retention.trim();
        if (raw) {
          const parsed = Number.parseInt(raw, 10);
          if (!Number.isFinite(parsed) || parsed <= 0) {
            alert(t('tasks.max_retention_invalid'));
            return;
          }
          payload.max_retention = parsed;
        } else if (editingTask) {
          payload.max_retention = 0;
        }
      }

      if (editingTask) {
        await apiClient.put(`/admin/tasks/${editingTask.id}`, payload, {
          headers: { Authorization: `Bearer ${token}` },
        });
      } else {
        await apiClient.post('/admin/tasks', payload, {
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

  const getRemoteDisplayName = (task: BackupTask) => {
    const remoteNameFromList = remotes.find(r => r.id === task.rclone_remote_id)?.name;
    return task.remote_name || remoteNameFromList || task.rclone_remote_id;
  };

  const getAgentStatusMeta = (status: Agent['status']) => {
    switch (status) {
      case 'online':
        return { label: t('common.online'), badgeClass: 'bg-success' };
      case 'running_task':
        return { label: t('common.running'), badgeClass: 'bg-primary' };
      default:
        return { label: t('common.offline'), badgeClass: 'bg-secondary' };
    }
  };

  const closeModal = () => {
    setShowCreateModal(false);
    setEditingTask(null);
    setSourceBrowserOpen(false);
    setSourceBrowserAgentId(null);
    setSourceBrowserPath('');
    setSourceBrowserParent('');
    setSourceBrowserEntries([]);
    setSourceBrowserLoading(false);
    setSourceBrowserError(null);
  };

  return (
    <div className="row row-deck row-cards">
      <div className="col-12">
        <div className="card">
          <div className="card-body d-flex justify-content-end gap-2">
            <button onClick={fetchData} className="btn btn-outline-primary">
              <IconRefresh size={16} />
              <span className="ms-1">{t('common.refresh')}</span>
            </button>
            <button onClick={handleCreateTask} className="btn btn-primary">
              <IconPlus size={16} />
              <span className="ms-1">{t('tasks.create_new')}</span>
            </button>
          </div>
        </div>
      </div>

      {tasks.length > 0 ? (
        tasks.map((task) => (
          <div key={task.id} className="col-12 col-md-6 col-xl-4">
            <div className="card">
              <div className="card-header">
                <div>
                  <h3 className="card-title mb-1">{task.name}</h3>
                  <div className="text-muted small">{getRemoteDisplayName(task)}</div>
                </div>
                <div className="ms-auto d-flex align-items-start flex-wrap gap-2">
                  <span className={`badge ${task.is_active ? 'bg-success' : 'bg-secondary'} text-white`}>
                    {task.is_active ? t('common.active') : t('common.inactive')}
                  </span>
                  <button
                    onClick={() => toggleTaskActive(task)}
                    className="btn btn-outline-secondary btn-sm"
                    title={task.is_active ? t('tasks.deactivate') : t('tasks.activate')}
                  >
                    {task.is_active ? <IconPlayerPause size={16} /> : <IconPlayerPlay size={16} />}
                  </button>
                  <button
                    onClick={() => handleEditTask(task)}
                    className="btn btn-outline-primary btn-sm"
                    title={t('common.edit')}
                  >
                    <IconEdit size={16} />
                  </button>
                  <button
                    onClick={() => handleDeleteTask(task.id)}
                    className="btn btn-outline-danger btn-sm"
                    title={t('common.delete')}
                  >
                    <IconTrash size={16} />
                  </button>
                </div>
              </div>

              <div className="card-body">
                <div className="mb-3">
                  <div className="text-muted small mb-1">{t('tasks.source')}</div>
                  <code className="text-break">{task.source_path}</code>
                </div>

                <div className="mb-3">
                  <div className="text-muted small mb-1">{t('tasks.destination')}</div>
                  <code className="text-break">{task.destination_path}</code>
                </div>

                <div className="row g-2 mb-3">
                  <div className="col-12 col-sm-6">
                    <div className="text-muted small mb-1">
                      <IconCalendar size={14} className="me-1" />
                      {t('tasks.list.columns.schedule')}
                    </div>
                    <code className="text-break">{task.schedule}</code>
                  </div>
                  <div className="col-12 col-sm-6">
                    <div className="text-muted small mb-1">
                      <IconClock size={14} className="me-1" />
                      {t('tasks.next_run')}
                    </div>
                    <div className="fw-bold text-break">
                      {task.next_run || formatNextRun(task.schedule)}
                    </div>
                  </div>
                </div>

                <div className="mb-3">
                  <div className="text-muted small mb-2">{t('tasks.assigned_agents')}</div>
                  {task.assigned_agents.length > 0 ? (
                    <div className="d-flex flex-column gap-2">
                      {task.assigned_agents.map(agentId => {
                        const status = getAgentStatus(agentId);
                        const statusMeta = getAgentStatusMeta(status);

                        return (
                          <div key={agentId} className="d-flex align-items-center justify-content-between gap-2">
                            <div className="text-break">{getAgentName(agentId)}</div>
                            <div className="d-flex align-items-center gap-2 flex-shrink-0">
                              <span className={`badge ${statusMeta.badgeClass} text-white`}>
                                {statusMeta.label}
                              </span>
                              {status === 'online' && (
                                <button
                                  onClick={() => handleTriggerTask(task.id, agentId)}
                                  className="btn btn-outline-primary btn-sm"
                                  title={t('tasks.run_now')}
                                >
                                  <IconPlayerPlay size={16} />
                                </button>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <div className="text-muted">{t('tasks.no_agents_assigned')}</div>
                  )}
                </div>

                {task.rclone_args.length > 0 && (
                  <div>
                    <div className="text-muted small mb-2">{t('tasks.arguments')}</div>
                    <div className="d-flex flex-wrap gap-1">
                      {task.rclone_args.map((arg, idx) => (
                        <span key={idx} className="badge bg-secondary text-white">
                          <code className="text-white">{arg}</code>
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        ))
      ) : (
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <p className="text-muted mb-3">{t('tasks.no_tasks')}</p>
              <button onClick={handleCreateTask} className="btn btn-primary">
                <IconPlus size={16} />
                <span className="ms-1">{t('tasks.create_first')}</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {showCreateModal && (
        <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1} role="dialog">
          <div className="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable" role="document">
            <div className="modal-content">
              <form onSubmit={handleSubmit}>
                <div className="modal-header">
                  <h5 className="modal-title">
                    {editingTask ? t('tasks.edit_task') : t('tasks.create_task')}
                  </h5>
                  <button type="button" className="btn-close" onClick={closeModal} aria-label={t('common.close')}></button>
                </div>

                <div className="modal-body">
                  <div className="row g-3">
                    <div className="col-12">
                      <label className="form-label">{t('tasks.task_name')}</label>
                      <input
                        type="text"
                        value={formData.name}
                        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        className="form-control"
                        required
                      />
                    </div>

                    <div className="col-12">
                      <label className="form-label">{t('tasks.remote')}</label>
                      <select
                        value={formData.rclone_remote_id}
                        onChange={(e) => setFormData({ ...formData, rclone_remote_id: e.target.value })}
                        className="form-select"
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

	                    <div className="col-12 col-md-6">
	                      <label className="form-label">{t('tasks.source_path')}</label>
	                      <div className="input-group">
	                        <input
	                          type="text"
	                          value={formData.source_path}
	                          onChange={(e) => setFormData({ ...formData, source_path: e.target.value })}
	                          placeholder="/path/to/source"
	                          className="form-control font-monospace"
	                          required
	                        />
	                        <button type="button" className="btn btn-outline-secondary" onClick={openSourceBrowser}>
	                          <IconFolder size={16} />
	                          <span className="ms-1">{t('tasks.browse.button')}</span>
	                        </button>
	                      </div>
	                    </div>

	                    <div className="col-12 col-md-6">
	                      <label className="form-label">{t('tasks.destination_path')}</label>
	                      <input
	                        type="text"
	                        value={formData.destination_path}
	                        onChange={(e) => setFormData({ ...formData, destination_path: e.target.value })}
	                        placeholder={t('tasks.destination_path_placeholder')}
	                        className="form-control font-monospace"
	                        required
	                      />
	                    </div>

	                    {sourceBrowserOpen && (
	                      <div className="col-12">
	                        <div className="card">
	                          <div className="card-header">
	                            <div>
	                              <h3 className="card-title mb-1">{t('tasks.browse.title')}</h3>
	                              {sourceBrowserAgentId && (
	                                <div className="text-muted small">
	                                  {t('tasks.browse.agent')}: {getAgentName(sourceBrowserAgentId)}
	                                </div>
	                              )}
	                            </div>
	                            <div className="ms-auto d-flex gap-2">
	                              <button
	                                type="button"
	                                className="btn btn-outline-secondary btn-sm"
	                                onClick={() => sourceBrowserParent && navigateSourceBrowser(sourceBrowserParent)}
	                                disabled={!sourceBrowserParent || sourceBrowserLoading || !sourceBrowserAgentId}
	                                title={t('tasks.browse.up')}
	                              >
	                                <IconArrowUp size={16} />
	                              </button>
	                              <button
	                                type="button"
	                                className="btn btn-outline-secondary btn-sm"
	                                onClick={() => sourceBrowserAgentId && navigateSourceBrowser(sourceBrowserPath)}
	                                disabled={sourceBrowserLoading || !sourceBrowserAgentId}
	                                title={t('common.refresh')}
	                              >
	                                <IconRefresh size={16} />
	                              </button>
	                              <button type="button" className="btn btn-outline-secondary btn-sm" onClick={closeSourceBrowser} disabled={sourceBrowserLoading}>
	                                {t('common.close')}
	                              </button>
	                            </div>
	                          </div>
	                          <div className="card-body">
	                            <div className="input-group mb-3">
	                              <input
	                                type="text"
	                                className="form-control font-monospace"
	                                value={sourceBrowserPath}
	                                onChange={(e) => setSourceBrowserPath(e.target.value)}
	                                placeholder={t('tasks.browse.pathPlaceholder')}
	                                disabled={sourceBrowserLoading}
	                                onKeyDown={(e) => {
	                                  if (e.key === 'Enter') {
	                                    e.preventDefault();
	                                    navigateSourceBrowser(sourceBrowserPath);
	                                  }
	                                }}
	                              />
	                              <button
	                                type="button"
	                                className="btn btn-outline-secondary"
	                                onClick={() => navigateSourceBrowser(sourceBrowserPath)}
	                                disabled={sourceBrowserLoading || !sourceBrowserAgentId}
	                              >
	                                {t('tasks.browse.go')}
	                              </button>
	                            </div>

	                            {sourceBrowserError && (
	                              <div className="alert alert-danger" role="alert">
	                                {sourceBrowserError}
	                              </div>
	                            )}

	                            {sourceBrowserLoading ? (
	                              <div className="text-center py-4">
	                                <IconRefresh className="spinner text-primary mb-2" size={24} />
	                                <div className="text-muted small">{t('common.loading')}</div>
	                              </div>
	                            ) : (
	                              <div className="list-group">
	                                {sourceBrowserParent && (
	                                  <button
	                                    type="button"
	                                    className="list-group-item list-group-item-action d-flex align-items-center gap-2"
	                                    onClick={() => navigateSourceBrowser(sourceBrowserParent)}
	                                    disabled={!sourceBrowserAgentId}
	                                  >
	                                    <IconArrowUp size={16} />
	                                    <span className="font-monospace">..</span>
	                                  </button>
	                                )}
	                                {sourceBrowserEntries.map((entry) => (
	                                  <button
	                                    key={entry.path}
	                                    type="button"
	                                    className="list-group-item list-group-item-action d-flex align-items-center gap-2"
	                                    onClick={() => navigateSourceBrowser(entry.path)}
	                                    disabled={!sourceBrowserAgentId}
	                                  >
	                                    <IconFolder size={16} />
	                                    <span className="font-monospace text-break">{entry.name}</span>
	                                  </button>
	                                ))}
	                                {!sourceBrowserParent && sourceBrowserEntries.length === 0 && !sourceBrowserError && (
	                                  <div className="text-muted small">{t('tasks.browse.empty')}</div>
	                                )}
	                              </div>
	                            )}

	                            <div className="d-flex justify-content-end gap-2 mt-3">
	                              <button type="button" className="btn btn-outline-secondary" onClick={closeSourceBrowser} disabled={sourceBrowserLoading}>
	                                {t('common.cancel')}
	                              </button>
	                              <button
	                                type="button"
	                                className="btn btn-primary"
	                                onClick={applySourceBrowserPath}
	                                disabled={sourceBrowserLoading || !sourceBrowserPath.trim()}
	                              >
	                                {t('tasks.browse.use')}
	                              </button>
	                            </div>
	                          </div>
	                        </div>
	                      </div>
	                    )}

	                    <div className="col-12">
	                      <label className="form-label">{t('tasks.list.columns.schedule')}</label>
	                      <div className="row g-2">
	                        <div className="col-12 col-md-6">
	                          <select
                            value={cronPresets.find(p => p.value === formData.schedule) ? formData.schedule : 'custom'}
                            onChange={(e) => {
                              if (e.target.value !== 'custom') {
                                setFormData({ ...formData, schedule: e.target.value });
                              }
                            }}
                            className="form-select"
                          >
                            {cronPresets.map((preset) => (
                              <option key={preset.value} value={preset.value}>
                                {preset.label}
                              </option>
                            ))}
                          </select>
                        </div>
                        <div className="col-12 col-md-6">
                          <input
                            type="text"
                            value={formData.schedule}
                            onChange={(e) => setFormData({ ...formData, schedule: e.target.value })}
                            placeholder="* * * * *"
                            className="form-control font-monospace"
                            required
                          />
                        </div>
                      </div>
                      <div className="form-text">{t('tasks.cron_help')}</div>
                    </div>

                    <div className="col-12">
                      <label className="form-label">{t('tasks.assign_agents')}</label>
                      <div className="d-flex flex-column gap-2">
                        {agents.map((agent) => {
                          const statusMeta = getAgentStatusMeta(agent.status);
                          return (
                            <div key={agent.id} className="form-check">
                              <input
                                type="checkbox"
                                className="form-check-input"
                                id={`task-agent-${agent.id}`}
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
                              />
                              <label className="form-check-label" htmlFor={`task-agent-${agent.id}`}>
                                {agent.name}
                                <span className={`badge ms-2 ${statusMeta.badgeClass} text-white`}>
                                  {statusMeta.label}
                                </span>
                              </label>
                            </div>
                          );
                        })}
                      </div>
                    </div>

                    <div className="col-12">
                      <label className="form-label">{t('tasks.backup_mode')}</label>
                      <select
                        value={formData.backup_mode}
                        onChange={(e) => {
                          const mode = e.target.value as 'sync' | 'archive';
                          setFormData({
                            ...formData,
                            backup_mode: mode,
                            archive_format: mode === 'archive' ? formData.archive_format : 'tar.gz',
                          });
                        }}
                        className="form-select"
                      >
                        <option value="sync">{t('tasks.backup_mode_sync')}</option>
                        <option value="archive">{t('tasks.backup_mode_archive')}</option>
                      </select>
                    </div>

                    {formData.backup_mode === 'archive' && (
                      <>
                        <div className="col-12 col-md-6">
                          <label className="form-label">{t('tasks.archive_format')}</label>
                          <select
                            value={formData.archive_format}
                            onChange={(e) => setFormData({ ...formData, archive_format: e.target.value as any })}
                            className="form-select"
                          >
                            <option value="tar.gz">tar.gz</option>
                            <option value="zip">zip</option>
                          </select>
                        </div>

                        <div className="col-12 col-md-6">
                          <label className="form-label">{t('tasks.max_retention')}</label>
                          <input
                            type="number"
                            min="1"
                            step="1"
                            value={formData.max_retention}
                            onChange={(e) => setFormData({ ...formData, max_retention: e.target.value })}
                            className="form-control"
                            placeholder={t('tasks.max_retention_placeholder')}
                          />
                          <div className="form-text">{t('tasks.max_retention_help')}</div>
                        </div>
                      </>
                    )}

                    <div className="col-12">
                      <label className="form-check form-switch">
                        <input
                          className="form-check-input"
                          type="checkbox"
                          checked={formData.encryption_enabled}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              encryption_enabled: e.target.checked,
                              encryption_password: e.target.checked ? formData.encryption_password : '',
                            })
                          }
                        />
                        <span className="form-check-label">{t('tasks.enable_encryption')}</span>
                      </label>
                      <div className="form-text">{t('tasks.encryption_help')}</div>
                    </div>

                    {formData.encryption_enabled && (
                      <div className="col-12 col-md-6">
                        <label className="form-label">{t('tasks.encryption_password')}</label>
                        <input
                          type="password"
                          value={formData.encryption_password}
                          onChange={(e) => setFormData({ ...formData, encryption_password: e.target.value })}
                          className="form-control"
                          placeholder={editingTask ? t('tasks.encryption_password_keep') : ''}
                          required={!editingTask}
                        />
                      </div>
                    )}

                    <div className="col-12">
                      <label className="form-label">{t('tasks.rclone_arguments')}</label>
                      <div className="d-flex flex-column gap-2">
                        {rcloneArgPresets.map((arg, index) => {
                          const inputId = `task-arg-${index}`;
                          return (
                            <div key={arg.label} className="form-check">
                              <input
                                type="checkbox"
                                className="form-check-input"
                                id={inputId}
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
                              />
                              <label className="form-check-label" htmlFor={inputId}>
                                <code>{arg.label}</code>
                                <div className="text-muted small">{arg.description}</div>
                              </label>
                            </div>
                          );
                        })}
                      </div>

                      <div className="mt-2">
                        <input
                          type="text"
                          placeholder={t('tasks.custom_args')}
                          onKeyDown={(e) => {
                            if (e.key !== 'Enter') return;
                            e.preventDefault();

                            const input = e.currentTarget;
                            const value = input.value.trim();
                            if (!value) return;
                            if (formData.rclone_args.includes(value)) {
                              input.value = '';
                              return;
                            }

                            setFormData({
                              ...formData,
                              rclone_args: [...formData.rclone_args, value],
                            });
                            input.value = '';
                          }}
                          className="form-control"
                        />
                      </div>

                      {formData.rclone_args.length > 0 && (
                        <div className="d-flex flex-wrap gap-1 mt-2">
                          {formData.rclone_args.map((arg, idx) => (
                            <span key={idx} className="badge bg-secondary text-white">
                              <code className="text-white">{arg}</code>
                              <button
                                type="button"
                                className="btn-close btn-close-white ms-2"
                                aria-label={t('common.delete')}
                                onClick={() => setFormData({
                                  ...formData,
                                  rclone_args: formData.rclone_args.filter((_, i) => i !== idx),
                                })}
                              ></button>
                            </span>
                          ))}
                        </div>
                      )}
                    </div>

                    <div className="col-12">
                      <div className="form-check form-switch">
                        <input
                          type="checkbox"
                          className="form-check-input"
                          id="task-is-active"
                          checked={formData.is_active}
                          onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                        />
                        <label className="form-check-label" htmlFor="task-is-active">
                          {t('tasks.activate_immediately')}
                        </label>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="modal-footer">
                  <button type="button" className="btn btn-secondary" onClick={closeModal}>
                    {t('common.cancel')}
                  </button>
                  <button type="submit" className="btn btn-primary">
                    {editingTask ? t('common.save') : t('common.create')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Tasks;

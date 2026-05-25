import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { tasksApi, agentsApi } from '../../../services';
import type { BackupTask, RcloneRemote, FSListEntry, TaskFormData } from '../../../types';

const initialFormData: TaskFormData = {
  name: '',
  rclone_remote_id: '',
  source_type: 'path',
  source_path: '',
  db_engine: 'postgres',
  db_dump_mode: 'single',
  db_host: '',
  db_port: '',
  db_user: '',
  db_name: '',
  db_password: '',
  db_path: '',
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
};

interface UseTaskFormOptions {
  remotes: RcloneRemote[];
  onSuccess: () => void;
}

export function useTaskForm({ remotes, onSuccess }: UseTaskFormOptions) {
  const { t } = useTranslation();
  const [editingTask, setEditingTask] = useState<BackupTask | null>(null);
  const [formData, setFormData] = useState<TaskFormData>(initialFormData);
  const [submitting, setSubmitting] = useState(false);

  // File browser state
  const [browserOpen, setBrowserOpen] = useState(false);
  const [browserAgentId, setBrowserAgentId] = useState<string | null>(null);
  const [browserPath, setBrowserPath] = useState('');
  const [browserParent, setBrowserParent] = useState('');
  const [browserEntries, setBrowserEntries] = useState<FSListEntry[]>([]);
  const [browserLoading, setBrowserLoading] = useState(false);
  const [browserError, setBrowserError] = useState<string | null>(null);

  const selectedRemote = remotes.find(r => r.id === formData.rclone_remote_id);
  const isS3Remote = (selectedRemote?.type || '').toLowerCase() === 's3';
  const isDatabaseSource = formData.source_type === 'database';

  // Auto-add S3 flag when S3 remote selected
  useEffect(() => {
    if (!isS3Remote) return;
    const flag = '--s3-no-check-bucket';
    setFormData(current => {
      if (current.rclone_args.includes(flag)) return current;
      return { ...current, rclone_args: [...current.rclone_args, flag] };
    });
  }, [isS3Remote]);

  // Force archive mode for database backups
  useEffect(() => {
    if (!isDatabaseSource) return;
    setFormData(current => {
      if (current.backup_mode === 'archive' && current.archive_format === '7z') return current;
      return { ...current, backup_mode: 'archive', archive_format: '7z' };
    });
  }, [isDatabaseSource]);

  // Force 7z for encrypted archives
  useEffect(() => {
    if (formData.backup_mode !== 'archive') return;
    if (!formData.encryption_enabled) return;
    if (formData.archive_format === '7z') return;
    setFormData(current => ({ ...current, archive_format: '7z' }));
  }, [formData.backup_mode, formData.encryption_enabled, formData.archive_format]);

  const resetForm = useCallback(() => {
    setEditingTask(null);
    setFormData(initialFormData);
    setBrowserOpen(false);
    setBrowserAgentId(null);
    setBrowserPath('');
    setBrowserParent('');
    setBrowserEntries([]);
    setBrowserLoading(false);
    setBrowserError(null);
  }, []);

  const openCreateModal = useCallback(() => {
    resetForm();
  }, [resetForm]);

  const openEditModal = useCallback((task: BackupTask) => {
    setEditingTask(task);
    setFormData({
      name: task.name,
      rclone_remote_id: task.rclone_remote_id,
      source_type: task.source_type || 'path',
      source_path: task.source_path,
      db_engine: (task.db_engine as any) || 'postgres',
      db_dump_mode: (task.db_dump_mode as any) || 'single',
      db_host: task.db_host || '',
      db_port: task.db_port ? String(task.db_port) : '',
      db_user: task.db_user || '',
      db_name: task.db_name || '',
      db_password: '',
      db_path: task.db_path || '',
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
    setBrowserOpen(false);
    setBrowserAgentId(null);
    setBrowserPath('');
    setBrowserParent('');
    setBrowserEntries([]);
    setBrowserLoading(false);
    setBrowserError(null);
  }, []);

  const updateFormData = useCallback(<K extends keyof TaskFormData>(key: K, value: TaskFormData[K]) => {
    setFormData(prev => ({ ...prev, [key]: value }));
  }, []);

  const toggleAgent = useCallback((agentId: string, checked: boolean) => {
    setFormData(prev => ({
      ...prev,
      assigned_agent_ids: checked
        ? [...prev.assigned_agent_ids, agentId]
        : prev.assigned_agent_ids.filter(id => id !== agentId),
    }));
  }, []);

  const toggleRcloneArg = useCallback((arg: string, checked: boolean) => {
    setFormData(prev => ({
      ...prev,
      rclone_args: checked
        ? [...prev.rclone_args, arg]
        : prev.rclone_args.filter(a => a !== arg),
    }));
  }, []);

  // File browser methods
  const fetchDirectory = useCallback(async (agentId: string, path: string) => {
    setBrowserLoading(true);
    setBrowserError(null);

    try {
      const response = await agentsApi.listDirectory(agentId, path, 200);
      setBrowserPath(response.path || path);
      setBrowserParent(response.parent || '');
      setBrowserEntries(response.entries || []);
    } catch (error: any) {
      const message = error?.response?.data?.message || error?.response?.data?.error || t('errors.server');
      setBrowserError(message);
      setBrowserEntries([]);
      setBrowserParent('');
    } finally {
      setBrowserLoading(false);
    }
  }, [t]);

  const openBrowser = useCallback(() => {
    setBrowserOpen(true);
    setBrowserError(null);
    setBrowserEntries([]);
    setBrowserParent('');

    const selectedAgentId = formData.assigned_agent_ids[0];
    if (!selectedAgentId) {
      setBrowserAgentId(null);
      setBrowserPath(formData.source_path || '/');
      setBrowserError(t('tasks.browse.select_agent_first'));
      return;
    }

    setBrowserAgentId(selectedAgentId);
    const initialPath = (formData.source_path || '/').trim() || '/';
    setBrowserPath(initialPath);
    void fetchDirectory(selectedAgentId, initialPath);
  }, [formData.assigned_agent_ids, formData.source_path, fetchDirectory, t]);

  const closeBrowser = useCallback(() => {
    if (browserLoading) return;
    setBrowserOpen(false);
    setBrowserAgentId(null);
    setBrowserPath('');
    setBrowserParent('');
    setBrowserEntries([]);
    setBrowserError(null);
  }, [browserLoading]);

  const navigateBrowser = useCallback((path: string) => {
    if (!browserAgentId || browserLoading) return;
    const next = path.trim();
    if (!next) return;
    setBrowserPath(next);
    void fetchDirectory(browserAgentId, next);
  }, [browserAgentId, browserLoading, fetchDirectory]);

  const applyBrowserPath = useCallback(() => {
    const next = browserPath.trim();
    if (!next) return;
    setFormData(prev => ({ ...prev, source_path: next }));
    closeBrowser();
  }, [browserPath, closeBrowser]);

  // Form validation and submission
  const validateForm = useCallback((): string | null => {
    if (!formData.destination_path.trim()) {
      return t('tasks.destination_path_required');
    }

    const dest = formData.destination_path.trim();
    const looksLikeRemotePrefix = /^[^\\/]+:/.test(dest) && !/^[A-Za-z]:[\\/]/.test(dest);
    if (looksLikeRemotePrefix) {
      return t('tasks.destination_path_no_remote');
    }

    if (formData.encryption_enabled && !editingTask && !formData.encryption_password.trim()) {
      return t('tasks.encryption_password_required');
    }

    if (!isDatabaseSource && !formData.source_path.trim()) {
      return t('tasks.source_path_required');
    }

    if (isDatabaseSource) {
      if (!formData.db_engine) {
        return t('tasks.db_engine_required');
      }
      if (formData.db_engine === 'sqlite') {
        if (!formData.db_path.trim()) {
          return t('tasks.db_path_required');
        }
      } else {
        if (!formData.db_host.trim() || !formData.db_user.trim()) {
          return t('tasks.db_conn_fields_required');
        }
        if (formData.db_dump_mode === 'single' && !formData.db_name.trim()) {
          return t('tasks.db_name_required');
        }
        if (formData.db_port.trim()) {
          const parsed = Number.parseInt(formData.db_port.trim(), 10);
          if (!Number.isFinite(parsed) || parsed <= 0) {
            return t('tasks.db_port_invalid');
          }
        }
      }
    }

    const raw = formData.max_retention.trim();
    if (raw) {
      const parsed = Number.parseInt(raw, 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        return t('tasks.max_retention_invalid');
      }
    }

    return null;
  }, [formData, editingTask, isDatabaseSource, t]);

  const buildPayload = useCallback(() => {
    let dbPort: number | undefined;
    if (isDatabaseSource && formData.db_port.trim()) {
      dbPort = Number.parseInt(formData.db_port.trim(), 10);
    }

    const payload: any = {
      name: formData.name,
      rclone_remote_id: formData.rclone_remote_id,
      source_type: formData.source_type,
      ...(isDatabaseSource ? {} : { source_path: formData.source_path }),
      destination_path: formData.destination_path,
      schedule: formData.schedule,
      rclone_args: formData.rclone_args,
      is_active: formData.is_active,
      assigned_agent_ids: formData.assigned_agent_ids,
      backup_mode: isDatabaseSource ? 'archive' : formData.backup_mode,
      archive_format: isDatabaseSource ? '7z' : formData.archive_format,
      encryption_enabled: formData.encryption_enabled,
    };

    if (isDatabaseSource) {
      payload.db_engine = formData.db_engine;
      payload.db_dump_mode = formData.db_dump_mode;
      if (formData.db_engine === 'sqlite') {
        payload.db_path = formData.db_path.trim();
      } else {
        payload.db_host = formData.db_host.trim();
        payload.db_user = formData.db_user.trim();
        if (formData.db_dump_mode === 'single' && formData.db_name.trim()) {
          payload.db_name = formData.db_name.trim();
        }
        if (formData.db_dump_mode === 'all' && formData.db_engine === 'postgres' && formData.db_name.trim()) {
          payload.db_name = formData.db_name.trim();
        }
        if (dbPort) payload.db_port = dbPort;
      }
      if (formData.db_password.trim()) {
        payload.db_password = formData.db_password.trim();
      }
    }

    if (formData.encryption_enabled && formData.encryption_password.trim()) {
      payload.encryption_password = formData.encryption_password.trim();
    }

    const backupMode = isDatabaseSource ? 'archive' : formData.backup_mode;
    if (backupMode === 'archive') {
      const raw = formData.max_retention.trim();
      if (raw) {
        payload.max_retention = Number.parseInt(raw, 10);
      } else if (editingTask) {
        payload.max_retention = 0;
      }
    }

    return payload;
  }, [formData, editingTask, isDatabaseSource]);

  const handleSubmit = useCallback(async (e: React.FormEvent): Promise<boolean> => {
    e.preventDefault();

    const error = validateForm();
    if (error) {
      alert(error);
      return false;
    }

    setSubmitting(true);
    try {
      const payload = buildPayload();
      if (editingTask) {
        await tasksApi.update(editingTask.id, payload);
      } else {
        await tasksApi.create(payload);
      }
      onSuccess();
      return true;
    } catch (error) {
      console.error('Failed to save task:', error);
      alert(t('tasks.save_failed'));
      return false;
    } finally {
      setSubmitting(false);
    }
  }, [validateForm, buildPayload, editingTask, onSuccess, t]);

  return {
    // Form state
    editingTask,
    formData,
    submitting,

    // Computed values
    isS3Remote,
    isDatabaseSource,
    selectedRemote,

    // Form methods
    resetForm,
    openCreateModal,
    openEditModal,
    updateFormData,
    toggleAgent,
    toggleRcloneArg,
    handleSubmit,

    // File browser state
    browserOpen,
    browserAgentId,
    browserPath,
    browserParent,
    browserEntries,
    browserLoading,
    browserError,

    // File browser methods
    openBrowser,
    closeBrowser,
    navigateBrowser,
    applyBrowserPath,
    setBrowserPath,
  };
}

import { apiClient } from './api';
import type { BackupTask } from '../types';

export interface CreateTaskPayload {
  name: string;
  rclone_remote_id: string;
  source_type?: 'path' | 'database';
  source_path?: string;
  db_engine?: string;
  db_dump_mode?: string;
  db_host?: string;
  db_port?: number;
  db_user?: string;
  db_name?: string;
  db_password?: string;
  db_path?: string;
  destination_path: string;
  schedule: string;
  rclone_args?: string[];
  is_active?: boolean;
  backup_mode?: 'sync' | 'archive';
  archive_format?: 'tar.gz' | 'zip' | '7z';
  encryption_enabled?: boolean;
  encryption_password?: string;
  max_retention?: number;
  assigned_agent_ids?: string[];
}

export type UpdateTaskPayload = Partial<CreateTaskPayload>;

export const tasksApi = {
  getAll: async (): Promise<BackupTask[]> => {
    const response = await apiClient.get<BackupTask[]>('/admin/tasks');
    return response.data;
  },

  getById: async (id: string): Promise<BackupTask> => {
    const response = await apiClient.get<BackupTask>(`/admin/tasks/${id}`);
    return response.data;
  },

  create: (data: CreateTaskPayload) => apiClient.post('/admin/tasks', data),

  update: (id: string, data: UpdateTaskPayload) =>
    apiClient.put(`/admin/tasks/${id}`, data),

  delete: (id: string) => apiClient.delete(`/admin/tasks/${id}`),

  trigger: (taskId: string, agentId: string) =>
    apiClient.post('/admin/executions/trigger', {
      task_id: taskId,
      agent_id: agentId,
    }),
};

export default tasksApi;

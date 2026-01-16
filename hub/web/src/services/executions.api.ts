import { apiClient } from './api';
import type { TaskExecution, ExecutionStats, PaginatedResponse } from '../types';

export interface ExecutionQueryParams {
  page?: number;
  limit?: number;
  status?: string;
  task_id?: string;
  agent_id?: string;
}

export const executionsApi = {
  getAll: async (params?: ExecutionQueryParams): Promise<PaginatedResponse<TaskExecution>> => {
    const response = await apiClient.get<PaginatedResponse<TaskExecution>>('/admin/executions', {
      params,
    });
    return response.data;
  },

  getById: async (id: string): Promise<TaskExecution> => {
    const response = await apiClient.get<TaskExecution>(`/admin/executions/${id}`);
    return response.data;
  },

  getStats: async (): Promise<ExecutionStats> => {
    const response = await apiClient.get<ExecutionStats>('/admin/executions/stats');
    return response.data;
  },

  getLogs: async (id: string): Promise<string[]> => {
    const response = await apiClient.get<string[]>(`/admin/executions/${id}/logs`);
    return response.data;
  },

  cancel: (id: string) => apiClient.post(`/admin/executions/${id}/cancel`),
};

export default executionsApi;

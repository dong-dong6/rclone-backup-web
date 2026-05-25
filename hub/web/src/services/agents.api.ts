import { apiClient } from './api';
import type {
  Agent,
  AgentMetric,
  AgentRegistrationConfig,
  AgentRegistrationResponse,
} from '../types';
import type { FSListResponse } from '../types/task.types';

export const agentsApi = {
  getAll: async (): Promise<Agent[]> => {
    const response = await apiClient.get<Agent[]>('/admin/agents');
    return response.data;
  },

  getById: async (id: string): Promise<Agent> => {
    const response = await apiClient.get<Agent>(`/admin/agents/${id}`);
    return response.data;
  },

  update: (id: string, data: Partial<Agent>) =>
    apiClient.put(`/admin/agents/${id}`, data),

  delete: (id: string) => apiClient.delete(`/admin/agents/${id}`),

  syncConfig: (id: string) => apiClient.post(`/admin/agents/${id}/sync`),

  getMetricsLatest: async (id: string): Promise<AgentMetric> => {
    const response = await apiClient.get<AgentMetric>(`/admin/agents/${id}/metrics/latest`);
    return response.data;
  },

  getMetricsHistory: async (id: string, hours: number): Promise<AgentMetric[]> => {
    const response = await apiClient.get<AgentMetric[]>(`/admin/agents/${id}/metrics/history`, {
      params: { hours },
    });
    return response.data;
  },

  generateRegistrationToken: async (config?: Partial<AgentRegistrationConfig>): Promise<AgentRegistrationResponse> => {
    const response = await apiClient.post<AgentRegistrationResponse>(
      '/admin/agents/registration-token',
      config
    );
    return response.data;
  },

  listDirectory: async (agentId: string, path: string, limit = 200): Promise<FSListResponse> => {
    const response = await apiClient.get<FSListResponse>(`/admin/agents/${agentId}/fs/list`, {
      params: { path, limit },
    });
    return response.data;
  },
};

export default agentsApi;

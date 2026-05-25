import { apiClient } from './api';
import type {
  RcloneRemote,
  RcloneRemoteDetail,
  RemoteTestResponse,
} from '../types';

export interface CreateRemotePayload {
  name: string;
  config_data: string;
  type?: string;
  preset_key?: string;
}

export type UpdateRemotePayload = Partial<CreateRemotePayload>;

export interface OAuthFlowResponse {
  flow_id: string;
  start_url: string;
  expires_at: string;
}

export interface OAuthFlowStatusResponse {
  status: 'pending' | 'success' | 'error';
  token?: Record<string, unknown>;
  error?: string;
}

export const remotesApi = {
  getAll: async (): Promise<RcloneRemote[]> => {
    const response = await apiClient.get<RcloneRemote[]>('/admin/remotes');
    return response.data;
  },

  getById: async (id: string): Promise<RcloneRemoteDetail> => {
    const response = await apiClient.get<RcloneRemoteDetail>(`/admin/remotes/${id}`);
    return response.data;
  },

  create: (data: CreateRemotePayload) => apiClient.post('/admin/remotes', data),

  update: (id: string, data: UpdateRemotePayload) =>
    apiClient.put(`/admin/remotes/${id}`, data),

  delete: (id: string) => apiClient.delete(`/admin/remotes/${id}`),

  test: async (id: string, testPath?: string): Promise<RemoteTestResponse> => {
    const response = await apiClient.post<RemoteTestResponse>(
      `/admin/remotes/${id}/test`,
      testPath ? { test_path: testPath } : null
    );
    return response.data;
  },

  // OAuth methods
  startOAuthFlow: async (
    provider: 'drive' | 'onedrive',
    params: Record<string, string>
  ): Promise<OAuthFlowResponse> => {
    const response = await apiClient.post<OAuthFlowResponse>(
      `/admin/oauth/${provider}/flow`,
      params
    );
    return response.data;
  },

  getOAuthFlowStatus: async (
    provider: 'drive' | 'onedrive',
    flowId: string
  ): Promise<OAuthFlowStatusResponse> => {
    const response = await apiClient.get<OAuthFlowStatusResponse>(
      `/admin/oauth/${provider}/flow/${flowId}`
    );
    return response.data;
  },
};

export default remotesApi;

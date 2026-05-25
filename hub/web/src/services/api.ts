import axios, { AxiosInstance, AxiosError } from 'axios';

// Use relative path for API calls (will use proxy in dev, direct path in production)
const API_BASE_URL = import.meta.env.VITE_API_URL || '';

// Create axios instance
const api: AxiosInstance = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    // Add language header
    const language = localStorage.getItem('language') || 'zh';
    config.headers['Accept-Language'] = language;
    
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error: AxiosError) => {
    // Handle common errors
    if (error.response) {
      switch (error.response.status) {
        case 401:
          // Unauthorized - redirect to login
          localStorage.removeItem('token');
          localStorage.removeItem('user');
          window.location.href = '/login';
          break;
        case 403:
          console.error('Forbidden: You do not have permission to access this resource');
          break;
        case 404:
          console.error('Resource not found');
          break;
        case 500:
          console.error('Server error');
          break;
      }
    } else if (error.request) {
      console.error('Network error: No response from server');
    }
    
    return Promise.reject(error);
  }
);

// API service methods
export const apiService = {
  // Auth
  login: (username: string, password: string) => 
    api.post('/admin/login', { username, password }),
  
  logout: () => 
    api.post('/admin/logout'),
  
  // Agents
  getAgents: () => 
    api.get('/admin/agents'),
  
  deleteAgent: (id: string) => 
    api.delete(`/admin/agents/${id}`),
  
  createRegistrationToken: () => 
    api.post('/admin/agents/registration-token'),

  getAgentMetricsLatest: (id: string) => 
    api.get(`/admin/agents/${id}/metrics/latest`),

  getAgentMetricsHistory: (id: string, hours: number) =>
    api.get(`/admin/agents/${id}/metrics/history`, { params: { hours } }),
  
  // Tasks
  getTasks: () => 
    api.get('/admin/tasks'),
  
  getTask: (id: string) => 
    api.get(`/admin/tasks/${id}`),
  
  createTask: (task: any) => 
    api.post('/admin/tasks', task),
  
  updateTask: (id: string, task: any) => 
    api.put(`/admin/tasks/${id}`, task),
  
  deleteTask: (id: string) => 
    api.delete(`/admin/tasks/${id}`),
  
  // Remotes
  getRemotes: () => 
    api.get('/admin/remotes'),
  
  getRemote: (id: string) => 
    api.get(`/admin/remotes/${id}`),
  
  createRemote: (remote: any) => 
    api.post('/admin/remotes', remote),
  
  updateRemote: (id: string, remote: any) => 
    api.put(`/admin/remotes/${id}`, remote),
  
  deleteRemote: (id: string) => 
    api.delete(`/admin/remotes/${id}`),
  
  testRemote: (id: string) => 
    api.post(`/admin/remotes/${id}/test`),
  
  // Executions
  getExecutions: (params?: { page?: number; limit?: number; status?: string }) => 
    api.get('/admin/executions', { params }),
  
  getExecution: (id: string) => 
    api.get(`/admin/executions/${id}`),
  
  triggerExecution: (taskId: string, agentId: string) => 
    api.post('/admin/executions/trigger', { task_id: taskId, agent_id: agentId }),
  
  cancelExecution: (id: string) => 
    api.post(`/admin/executions/${id}/cancel`),
  
  // Statistics & Dashboard
  getStatistics: () => 
    api.get('/admin/statistics/overview'),
  
  getAgentStatistics: (id: string) => 
    api.get(`/admin/statistics/agents/${id}`),
  
  getTaskStatistics: (id: string) => 
    api.get(`/admin/statistics/tasks/${id}`),
    
  getDashboardStats: () => 
    api.get('/admin/dashboard/stats'),
    
  getRecentActivity: () => 
    api.get('/admin/dashboard/recent'),
    
  getChartData: (timeRange: string) => 
    api.get(`/admin/dashboard/charts?range=${timeRange}`),
  
  // System
  getSystemHealth: () => 
    api.get('/health'),
  
  exportConfig: () => 
    api.get('/admin/export/config', { responseType: 'blob' }),
  
  importConfig: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post('/admin/import/config', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
};

// Export both named and default
export { api as apiClient };
export default api;

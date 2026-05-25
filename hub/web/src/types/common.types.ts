// Common types used across the application

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  total_pages: number;
}

export interface ApiError {
  message: string;
  code?: string;
  details?: Record<string, string>;
}

export type Status = 'online' | 'offline' | 'running' | 'pending' | 'success' | 'failed';

export interface SelectOption {
  label: string;
  value: string;
  description?: string;
}

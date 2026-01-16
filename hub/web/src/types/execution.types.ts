// Execution related types

export type ExecutionStatus = 'pending' | 'running' | 'success' | 'failed';
export type TriggerMode = 'manual' | 'scheduled' | 'local_fallback' | 'central';

export interface TaskExecution {
  id: string;
  task_id: string;
  task_name: string;
  agent_id: string;
  agent_name: string;
  status: ExecutionStatus;
  trigger_mode: TriggerMode;
  log_output?: string;
  error_message?: string;
  started_at?: string;
  ended_at?: string;
  created_at: string;
  duration_seconds?: number;
}

export interface ExecutionStats {
  total: number;
  success: number;
  failed: number;
  running: number;
  success_rate_24h: number;
  avg_duration_seconds: number;
}

export interface ExecutionFilter {
  status: string;
  taskId: string;
  agentId: string;
}

// Agent related types

export type AgentStatus = 'online' | 'offline' | 'running_task';

export interface Agent {
  id: string;
  name: string;
  status: AgentStatus;
  last_heartbeat: string | null;
  is_local: boolean;
  created_at: string;
  task_count?: number;
  current_task?: string;
}

export interface AgentMetric {
  cpu_usage: number;
  memory_usage: number;
  memory_total: number;
  memory_used: number;
  disk_total: number;
  disk_used: number;
  disk_usage: number;
  network_rx_rate: number;
  network_tx_rate: number;
  tcp_connections: number;
  udp_connections: number;
  process_count: number;
  recorded_at: string;
}

export interface AgentStats {
  total: number;
  online: number;
  running: number;
}

export interface AgentRegistrationConfig {
  agent_name: string;
  run_as_root: boolean;
  log_level: 'debug' | 'info' | 'warn' | 'error';
  enable_api: boolean;
  api_port: number;
}

export interface AgentRegistrationResponse {
  token: string;
  expires_at: string;
  install_command: string;
}

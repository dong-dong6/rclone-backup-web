// Task related types

export type SourceType = 'path' | 'database';
export type DbEngine = 'postgres' | 'mysql' | 'sqlite';
export type DbDumpMode = 'single' | 'all';
export type BackupMode = 'sync' | 'archive';
export type ArchiveFormat = 'tar.gz' | 'zip' | '7z';

export interface BackupTask {
  id: string;
  name: string;
  rclone_remote_id: string;
  remote_name?: string;
  source_type?: SourceType;
  source_path: string;
  db_engine?: DbEngine | null;
  db_dump_mode?: DbDumpMode | null;
  db_host?: string | null;
  db_port?: number | null;
  db_user?: string | null;
  db_name?: string | null;
  db_path?: string | null;
  destination_path: string;
  schedule: string;
  rclone_args: string[];
  is_active: boolean;
  backup_mode: BackupMode;
  archive_format?: ArchiveFormat;
  encryption_enabled?: boolean;
  max_retention?: number | null;
  created_at: string;
  updated_at: string;
  assigned_agents: string[];
  next_run?: string;
  last_run?: string;
}

export interface TaskFormData {
  name: string;
  rclone_remote_id: string;
  source_type: SourceType;
  source_path: string;
  db_engine: DbEngine;
  db_dump_mode: DbDumpMode;
  db_host: string;
  db_port: string;
  db_user: string;
  db_name: string;
  db_password: string;
  db_path: string;
  destination_path: string;
  schedule: string;
  rclone_args: string[];
  is_active: boolean;
  backup_mode: BackupMode;
  archive_format: ArchiveFormat;
  encryption_enabled: boolean;
  encryption_password: string;
  max_retention: string;
  assigned_agent_ids: string[];
}

export interface FSListEntry {
  name: string;
  path: string;
  is_dir: boolean;
  is_symlink?: boolean;
}

export interface FSListResponse {
  path: string;
  parent?: string;
  entries: FSListEntry[];
}

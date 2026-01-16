export const createCronPresets = (t: (key: string) => string) => [
  { label: t('tasks.schedule.every_hour'), value: '0 * * * *' },
  { label: t('tasks.schedule.daily_2am'), value: '0 2 * * *' },
  { label: t('tasks.schedule.weekly_sunday'), value: '0 2 * * 0' },
  { label: t('tasks.schedule.monthly_first'), value: '0 2 1 * *' },
  { label: t('tasks.schedule.custom'), value: 'custom' },
];

export const createRcloneArgPresets = (t: (key: string) => string, isS3Remote: boolean) => {
  const base = [
    { label: '--dry-run', description: t('tasks.args.dry_run') },
    { label: '--verbose', description: t('tasks.args.verbose') },
    { label: '--checksum', description: t('tasks.args.checksum') },
    { label: '--delete-after', description: t('tasks.args.delete_after') },
    { label: '--exclude *.tmp', description: t('tasks.args.exclude_tmp') },
  ];

  if (isS3Remote) {
    base.push({
      label: '--s3-no-check-bucket',
      description: t('tasks.args.s3_no_check_bucket'),
    });
  }

  return base;
};

export interface TaskFormData {
  name: string;
  rclone_remote_id: string;
  source_type: 'path' | 'database';
  source_path: string;
  db_engine: 'postgres' | 'mysql' | 'sqlite';
  db_dump_mode: 'single' | 'all';
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
  backup_mode: 'sync' | 'archive';
  archive_format: 'tar.gz' | 'zip' | '7z';
  encryption_enabled: boolean;
  encryption_password: string;
  max_retention: string;
  assigned_agent_ids: string[];
}

export const defaultFormData: TaskFormData = {
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

import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconFolder,
  IconRefresh,
  IconArrowUp,
} from '@tabler/icons-react';
import type { Agent, RcloneRemote, TaskFormData, FSListEntry } from '../../../types';
import { createCronPresets, createRcloneArgPresets } from '../constants';

export interface TaskModalProps {
  isOpen: boolean;
  isEditing: boolean;
  formData: TaskFormData;
  submitting: boolean;
  isS3Remote: boolean;
  isDatabaseSource: boolean;
  agents: Agent[];
  remotes: RcloneRemote[];
  onClose: () => void;
  onSubmit: (e: React.FormEvent) => void;
  updateFormData: <K extends keyof TaskFormData>(key: K, value: TaskFormData[K]) => void;
  toggleAgent: (agentId: string, checked: boolean) => void;
  toggleRcloneArg: (arg: string, checked: boolean) => void;
  // File browser props
  browserOpen: boolean;
  browserPath: string;
  browserParent: string;
  browserEntries: FSListEntry[];
  browserLoading: boolean;
  browserError: string | null;
  onOpenBrowser: () => void;
  onCloseBrowser: () => void;
  onNavigateBrowser: (path: string) => void;
  onApplyBrowserPath: () => void;
  onSetBrowserPath: (path: string) => void;
}

export const TaskModal: React.FC<TaskModalProps> = ({
  isOpen,
  isEditing,
  formData,
  submitting,
  isS3Remote,
  isDatabaseSource,
  agents,
  remotes,
  onClose,
  onSubmit,
  updateFormData,
  toggleAgent,
  toggleRcloneArg,
  browserOpen,
  browserPath,
  browserParent,
  browserEntries,
  browserLoading,
  browserError,
  onOpenBrowser,
  onCloseBrowser,
  onNavigateBrowser,
  onApplyBrowserPath,
  onSetBrowserPath,
}) => {
  const { t } = useTranslation();
  const cronPresets = createCronPresets(t);
  const rcloneArgPresets = createRcloneArgPresets(t, isS3Remote);

  const getAgentStatusMeta = (status: Agent['status']) => {
    switch (status) {
      case 'online':
        return { label: t('common.online'), badgeClass: 'bg-success' };
      case 'running_task':
        return { label: t('common.running'), badgeClass: 'bg-primary' };
      default:
        return { label: t('common.offline'), badgeClass: 'bg-secondary' };
    }
  };

  if (!isOpen) return null;

  return (
    <>
      <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1} role="dialog">
        <div className="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable" role="document">
          <div className="modal-content">
            <form onSubmit={onSubmit}>
              <div className="modal-header">
                <h5 className="modal-title">
                  {isEditing ? t('tasks.edit_task') : t('tasks.create_task')}
                </h5>
                <button type="button" className="btn-close" onClick={onClose} aria-label={t('common.close')} />
              </div>

              <div className="modal-body">
                <div className="row g-3">
                  {/* Basic Info Section */}
                  <div className="col-12">
                    <div className="card">
                      <div className="card-header">
                        <h3 className="card-title">{t('tasks.create.basic')}</h3>
                      </div>
                      <div className="card-body">
                        <div className="row g-3">
                          <div className="col-12">
                            <label className="form-label">{t('tasks.task_name')}</label>
                            <input
                              type="text"
                              value={formData.name}
                              onChange={(e) => updateFormData('name', e.target.value)}
                              className="form-control"
                              required
                            />
                          </div>

                          <div className="col-12">
                            <label className="form-label">{t('tasks.remote')}</label>
                            <select
                              value={formData.rclone_remote_id}
                              onChange={(e) => updateFormData('rclone_remote_id', e.target.value)}
                              className="form-select"
                              required
                            >
                              <option value="">{t('tasks.select_remote')}</option>
                              {remotes.map((remote) => (
                                <option key={remote.id} value={remote.id}>
                                  {remote.name}
                                </option>
                              ))}
                            </select>
                          </div>

                          <div className="col-12">
                            <label className="form-label">{t('tasks.backup_mode')}</label>
                            <select
                              value={formData.backup_mode}
                              onChange={(e) => {
                                const mode = e.target.value as 'sync' | 'archive';
                                updateFormData('backup_mode', mode);
                                if (mode === 'archive') {
                                  updateFormData('archive_format', formData.archive_format || 'tar.gz');
                                }
                              }}
                              className="form-select"
                              disabled={isDatabaseSource}
                            >
                              <option value="sync">{t('tasks.backup_mode_sync')}</option>
                              <option value="archive">{t('tasks.backup_mode_archive')}</option>
                            </select>
                            <div className="form-text">
                              {isDatabaseSource
                                ? t('tasks.backup_mode_help_database')
                                : formData.backup_mode === 'sync'
                                  ? t('tasks.backup_mode_help_sync')
                                  : t('tasks.backup_mode_help_archive')}
                            </div>
                          </div>

                          {formData.backup_mode === 'archive' && (
                            <>
                              <div className="col-12 col-md-6">
                                <label className="form-label">{t('tasks.archive_format')}</label>
                                <select
                                  value={formData.archive_format}
                                  onChange={(e) => updateFormData('archive_format', e.target.value as any)}
                                  className="form-select"
                                  disabled={formData.encryption_enabled || isDatabaseSource}
                                >
                                  <option value="tar.gz">tar.gz</option>
                                  <option value="zip">zip</option>
                                  <option value="7z">7z</option>
                                </select>
                              </div>

                              <div className="col-12 col-md-6">
                                <label className="form-label">{t('tasks.max_retention')}</label>
                                <input
                                  type="number"
                                  min="1"
                                  step="1"
                                  value={formData.max_retention}
                                  onChange={(e) => updateFormData('max_retention', e.target.value)}
                                  className="form-control"
                                  placeholder={t('tasks.max_retention_placeholder')}
                                />
                                <div className="form-text">{t('tasks.max_retention_help')}</div>
                              </div>
                            </>
                          )}

                          <div className="col-12">
                            <div className="form-check form-switch">
                              <input
                                type="checkbox"
                                className="form-check-input"
                                id="task-is-active"
                                checked={formData.is_active}
                                onChange={(e) => updateFormData('is_active', e.target.checked)}
                              />
                              <label className="form-check-label" htmlFor="task-is-active">
                                {t('tasks.activate_immediately')}
                              </label>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Agents and Paths Section */}
                  <div className="col-12">
                    <div className="card">
                      <div className="card-header">
                        <h3 className="card-title">{t('tasks.create.agents_paths')}</h3>
                      </div>
                      <div className="card-body">
                        <div className="row g-3">
                          <div className="col-12">
                            <label className="form-label">{t('tasks.assign_agents')}</label>
                            <div className="d-flex flex-column gap-2">
                              {agents.map((agent) => {
                                const statusMeta = getAgentStatusMeta(agent.status);
                                return (
                                  <div key={agent.id} className="form-check">
                                    <input
                                      type="checkbox"
                                      className="form-check-input"
                                      id={`task-agent-${agent.id}`}
                                      checked={formData.assigned_agent_ids.includes(agent.id)}
                                      onChange={(e) => toggleAgent(agent.id, e.target.checked)}
                                    />
                                    <label className="form-check-label" htmlFor={`task-agent-${agent.id}`}>
                                      {agent.name}
                                      <span className={`badge ms-2 ${statusMeta.badgeClass} text-white`}>
                                        {statusMeta.label}
                                      </span>
                                    </label>
                                  </div>
                                );
                              })}
                            </div>
                            <div className="form-text">{t('tasks.assigned_agents_help')}</div>
                          </div>

                          <div className="col-12">
                            <hr className="my-2" />
                          </div>

                          <div className="col-12">
                            <label className="form-label">{t('tasks.source_type')}</label>
                            <select
                              value={formData.source_type}
                              onChange={(e) => updateFormData('source_type', e.target.value as any)}
                              className="form-select"
                            >
                              <option value="path">{t('tasks.source_type_path')}</option>
                              <option value="database">{t('tasks.source_type_database')}</option>
                            </select>
                            <div className="form-text">{t('tasks.source_type_help')}</div>
                          </div>

                          {formData.source_type === 'path' ? (
                            <>
                              <div className="col-12 col-md-6">
                                <label className="form-label">{t('tasks.source_path')}</label>
                                <div className="input-group">
                                  <input
                                    type="text"
                                    value={formData.source_path}
                                    onChange={(e) => updateFormData('source_path', e.target.value)}
                                    placeholder="/path/to/source"
                                    className="form-control font-monospace"
                                    required
                                  />
                                  <button
                                    type="button"
                                    className="btn btn-outline-secondary"
                                    onClick={onOpenBrowser}
                                    disabled={formData.assigned_agent_ids.length === 0}
                                    title={
                                      formData.assigned_agent_ids.length === 0
                                        ? t('tasks.browse.select_agent_first')
                                        : t('tasks.browse.button')
                                    }
                                  >
                                    <IconFolder size={16} />
                                    <span className="ms-1">{t('tasks.browse.button')}</span>
                                  </button>
                                </div>
                              </div>

                              <div className="col-12 col-md-6">
                                <label className="form-label">{t('tasks.destination_path')}</label>
                                <input
                                  type="text"
                                  value={formData.destination_path}
                                  onChange={(e) => updateFormData('destination_path', e.target.value)}
                                  placeholder={
                                    isS3Remote
                                      ? t('tasks.destination_path_placeholder_s3')
                                      : t('tasks.destination_path_placeholder')
                                  }
                                  className="form-control font-monospace"
                                  required
                                />
                                <div className="form-text">
                                  {isS3Remote ? `${t('tasks.destination_path_help_s3')} ` : ''}
                                  {formData.backup_mode === 'sync'
                                    ? t('tasks.destination_path_help_sync')
                                    : t('tasks.destination_path_help_archive')}
                                </div>
                              </div>
                            </>
                          ) : (
                            <DatabaseConfigSection
                              formData={formData}
                              updateFormData={updateFormData}
                            />
                          )}
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Encryption Section */}
                  {formData.backup_mode === 'archive' && (
                    <div className="col-12">
                      <div className="card">
                        <div className="card-header">
                          <h3 className="card-title">{t('tasks.create.encryption')}</h3>
                        </div>
                        <div className="card-body">
                          <div className="row g-3">
                            <div className="col-12">
                              <div className="form-check form-switch">
                                <input
                                  type="checkbox"
                                  className="form-check-input"
                                  id="task-encryption"
                                  checked={formData.encryption_enabled}
                                  onChange={(e) => updateFormData('encryption_enabled', e.target.checked)}
                                />
                                <label className="form-check-label" htmlFor="task-encryption">
                                  {t('tasks.encryption_enable')}
                                </label>
                              </div>
                              <div className="form-text">{t('tasks.encryption_help')}</div>
                            </div>

                            {formData.encryption_enabled && (
                              <div className="col-12">
                                <label className="form-label">{t('tasks.encryption_password')}</label>
                                <input
                                  type="password"
                                  value={formData.encryption_password}
                                  onChange={(e) => updateFormData('encryption_password', e.target.value)}
                                  className="form-control"
                                  placeholder={isEditing ? t('tasks.encryption_password_placeholder_edit') : ''}
                                />
                                <div className="form-text">
                                  {isEditing ? t('tasks.encryption_password_help_edit') : t('tasks.encryption_password_help')}
                                </div>
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Schedule Section */}
                  <div className="col-12">
                    <div className="card">
                      <div className="card-header">
                        <h3 className="card-title">{t('tasks.create.schedule')}</h3>
                      </div>
                      <div className="card-body">
                        <div className="row g-3">
                          <div className="col-12">
                            <label className="form-label">{t('tasks.schedule_preset')}</label>
                            <select
                              value={cronPresets.some(p => p.value === formData.schedule) ? formData.schedule : 'custom'}
                              onChange={(e) => {
                                if (e.target.value !== 'custom') {
                                  updateFormData('schedule', e.target.value);
                                }
                              }}
                              className="form-select"
                            >
                              {cronPresets.map((preset) => (
                                <option key={preset.value} value={preset.value}>
                                  {preset.label}
                                </option>
                              ))}
                            </select>
                          </div>

                          <div className="col-12">
                            <label className="form-label">{t('tasks.schedule_cron')}</label>
                            <input
                              type="text"
                              value={formData.schedule}
                              onChange={(e) => updateFormData('schedule', e.target.value)}
                              className="form-control font-monospace"
                              placeholder="0 2 * * *"
                              required
                            />
                            <div className="form-text">{t('tasks.schedule_cron_help')}</div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Advanced Section */}
                  <div className="col-12">
                    <div className="card">
                      <div className="card-header">
                        <h3 className="card-title">{t('tasks.create.advanced')}</h3>
                      </div>
                      <div className="card-body">
                        <div className="row g-3">
                          <div className="col-12">
                            <label className="form-label">{t('tasks.rclone_args')}</label>
                            <div className="d-flex flex-column gap-2">
                              {rcloneArgPresets.map((preset) => (
                                <div key={preset.label} className="form-check">
                                  <input
                                    type="checkbox"
                                    className="form-check-input"
                                    id={`rclone-arg-${preset.label}`}
                                    checked={formData.rclone_args.includes(preset.label)}
                                    onChange={(e) => toggleRcloneArg(preset.label, e.target.checked)}
                                  />
                                  <label className="form-check-label" htmlFor={`rclone-arg-${preset.label}`}>
                                    <code>{preset.label}</code>
                                    <span className="text-muted ms-2">- {preset.description}</span>
                                  </label>
                                </div>
                              ))}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={onClose} disabled={submitting}>
                  {t('common.cancel')}
                </button>
                <button type="submit" className="btn btn-primary" disabled={submitting}>
                  {submitting && <IconRefresh className="spinner me-2" size={16} />}
                  {isEditing ? t('common.save') : t('common.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
      <div className="modal-backdrop fade show" />

      {/* File Browser Modal */}
      {browserOpen && (
        <>
          <div className="modal modal-blur fade show" style={{ display: 'block', zIndex: 1060 }} tabIndex={-1}>
            <div className="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable">
              <div className="modal-content">
                <div className="modal-header">
                  <h5 className="modal-title">{t('tasks.browse.title')}</h5>
                  <button type="button" className="btn-close" onClick={onCloseBrowser} aria-label={t('common.close')} />
                </div>

                <div className="modal-body">
                  <div className="mb-3">
                    <div className="input-group">
                      <input
                        type="text"
                        className="form-control font-monospace"
                        value={browserPath}
                        onChange={(e) => onSetBrowserPath(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            onNavigateBrowser(browserPath);
                          }
                        }}
                      />
                      <button
                        type="button"
                        className="btn btn-outline-secondary"
                        onClick={() => onNavigateBrowser(browserPath)}
                        disabled={browserLoading}
                      >
                        <IconRefresh size={16} className={browserLoading ? 'spinner' : ''} />
                      </button>
                    </div>
                  </div>

                  {browserError && (
                    <div className="alert alert-danger">{browserError}</div>
                  )}

                  <div className="list-group list-group-flush" style={{ maxHeight: '400px', overflowY: 'auto' }}>
                    {browserParent && (
                      <button
                        type="button"
                        className="list-group-item list-group-item-action d-flex align-items-center"
                        onClick={() => onNavigateBrowser(browserParent)}
                        disabled={browserLoading}
                      >
                        <IconArrowUp size={16} className="me-2 text-muted" />
                        <span className="text-muted">..</span>
                      </button>
                    )}
                    {browserEntries.map((entry) => (
                      <button
                        key={entry.path}
                        type="button"
                        className="list-group-item list-group-item-action d-flex align-items-center"
                        onClick={() => {
                          if (entry.is_dir) {
                            onNavigateBrowser(entry.path);
                          } else {
                            onSetBrowserPath(entry.path);
                          }
                        }}
                        disabled={browserLoading}
                      >
                        <IconFolder size={16} className={`me-2 ${entry.is_dir ? 'text-warning' : 'text-muted'}`} />
                        <span className={entry.is_dir ? '' : 'text-muted'}>{entry.name}</span>
                        {entry.is_symlink && <span className="badge bg-secondary ms-auto">symlink</span>}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="modal-footer">
                  <button type="button" className="btn btn-secondary" onClick={onCloseBrowser}>
                    {t('common.cancel')}
                  </button>
                  <button type="button" className="btn btn-primary" onClick={onApplyBrowserPath}>
                    {t('tasks.browse.apply')}
                  </button>
                </div>
              </div>
            </div>
          </div>
          <div className="modal-backdrop fade show" style={{ zIndex: 1055 }} />
        </>
      )}
    </>
  );
};

// Database configuration sub-component
interface DatabaseConfigSectionProps {
  formData: TaskFormData;
  updateFormData: <K extends keyof TaskFormData>(key: K, value: TaskFormData[K]) => void;
}

const DatabaseConfigSection: React.FC<DatabaseConfigSectionProps> = ({
  formData,
  updateFormData,
}) => {
  const { t } = useTranslation();

  return (
    <>
      <div className="col-12 col-md-4">
        <label className="form-label">{t('tasks.db_engine')}</label>
        <select
          value={formData.db_engine}
          onChange={(e) => {
            const engine = e.target.value as any;
            updateFormData('db_engine', engine);
            if (engine === 'sqlite') {
              updateFormData('db_dump_mode', 'single');
            }
          }}
          className="form-select"
        >
          <option value="postgres">{t('tasks.db_engine_postgres')}</option>
          <option value="mysql">{t('tasks.db_engine_mysql')}</option>
          <option value="sqlite">{t('tasks.db_engine_sqlite')}</option>
        </select>
      </div>

      {formData.db_engine === 'sqlite' ? (
        <div className="col-12 col-md-8">
          <label className="form-label">{t('tasks.db_path')}</label>
          <input
            type="text"
            value={formData.db_path}
            onChange={(e) => updateFormData('db_path', e.target.value)}
            placeholder="/path/to/app.sqlite3"
            className="form-control font-monospace"
            required
          />
        </div>
      ) : (
        <>
          <div className="col-12 col-md-4">
            <label className="form-label">{t('tasks.db_dump_mode')}</label>
            <select
              value={formData.db_dump_mode}
              onChange={(e) => updateFormData('db_dump_mode', e.target.value as any)}
              className="form-select"
            >
              <option value="single">{t('tasks.db_dump_mode_single')}</option>
              <option value="all">{t('tasks.db_dump_mode_all')}</option>
            </select>
            <div className="form-text">{t('tasks.db_dump_mode_help')}</div>
          </div>

          {formData.db_dump_mode === 'all' && (
            <div className="col-12">
              <div className="alert alert-warning" role="alert">
                <div className="fw-bold mb-1">{t('tasks.db_all_warning_title')}</div>
                <div className="small">{t('tasks.db_all_warning_body')}</div>
              </div>
            </div>
          )}

          <div className="col-12 col-md-6">
            <label className="form-label">{t('tasks.db_host')}</label>
            <input
              type="text"
              value={formData.db_host}
              onChange={(e) => updateFormData('db_host', e.target.value)}
              placeholder="127.0.0.1"
              className="form-control"
              required
            />
          </div>

          <div className="col-12 col-md-3">
            <label className="form-label">{t('tasks.db_port')}</label>
            <input
              type="number"
              min="1"
              step="1"
              value={formData.db_port}
              onChange={(e) => updateFormData('db_port', e.target.value)}
              placeholder={formData.db_engine === 'postgres' ? '5432' : '3306'}
              className="form-control"
            />
          </div>

          <div className="col-12 col-md-3">
            <label className="form-label">{t('tasks.db_user')}</label>
            <input
              type="text"
              value={formData.db_user}
              onChange={(e) => updateFormData('db_user', e.target.value)}
              className="form-control"
              required
            />
          </div>

          {(formData.db_dump_mode === 'single' ||
            (formData.db_dump_mode === 'all' && formData.db_engine === 'postgres')) && (
            <div className="col-12 col-md-6">
              <label className="form-label">{t('tasks.db_name')}</label>
              <input
                type="text"
                value={formData.db_name}
                onChange={(e) => updateFormData('db_name', e.target.value)}
                className="form-control"
                required={formData.db_dump_mode === 'single'}
              />
              {formData.db_dump_mode === 'all' && formData.db_engine === 'postgres' && (
                <div className="form-text">{t('tasks.db_name_postgres_all_help')}</div>
              )}
            </div>
          )}

          <div className="col-12 col-md-6">
            <label className="form-label">{t('tasks.db_password')}</label>
            <input
              type="password"
              value={formData.db_password}
              onChange={(e) => updateFormData('db_password', e.target.value)}
              className="form-control"
            />
          </div>
        </>
      )}

      <div className="col-12">
        <label className="form-label">{t('tasks.destination_path')}</label>
        <input
          type="text"
          value={formData.destination_path}
          onChange={(e) => updateFormData('destination_path', e.target.value)}
          placeholder={t('tasks.destination_path_placeholder')}
          className="form-control font-monospace"
          required
        />
        <div className="form-text">{t('tasks.destination_path_help_archive')}</div>
      </div>
    </>
  );
};

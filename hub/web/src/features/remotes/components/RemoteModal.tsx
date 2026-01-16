import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconRefresh, IconKey } from '@tabler/icons-react';
import { Modal, Loading } from '../../../components/ui';
import type { RemotePreset, RemotePresetKey } from '../constants';
import type { ConfigMode, RemoteFormState } from '../hooks';

export interface RemoteModalProps {
  isOpen: boolean;
  isEditing: boolean;
  modalLoading: boolean;
  configMode: ConfigMode;
  presetKey: RemotePresetKey;
  guidedValues: Record<string, string>;
  formData: RemoteFormState;
  currentPreset: RemotePreset;
  presets: Record<RemotePresetKey, RemotePreset>;
  previewConfig: string;
  oauthPending: boolean;
  onClose: () => void;
  onSubmit: (e: React.FormEvent) => void;
  onSwitchToGuided: () => void;
  onSwitchToRaw: () => void;
  onPresetChange: (preset: RemotePresetKey) => void;
  onGuidedValueChange: (key: string, value: string) => void;
  onFormDataChange: (field: keyof RemoteFormState, value: string) => void;
  onStartOAuth: (provider: 'drive' | 'onedrive') => void;
}

export const RemoteModal: React.FC<RemoteModalProps> = ({
  isOpen,
  isEditing,
  modalLoading,
  configMode,
  presetKey,
  guidedValues,
  formData,
  currentPreset,
  presets,
  previewConfig,
  oauthPending,
  onClose,
  onSubmit,
  onSwitchToGuided,
  onSwitchToRaw,
  onPresetChange,
  onGuidedValueChange,
  onFormDataChange,
  onStartOAuth,
}) => {
  const { t } = useTranslation();

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? t('remotes.edit.title') : t('remotes.create.title')}
      size="lg"
    >
      <form onSubmit={onSubmit}>
        {modalLoading ? (
          <div className="text-center py-5">
            <Loading text={t('common.loading')} />
          </div>
        ) : (
          <>
            <div className="mb-3">
              <label className="form-label">{t('remotes.form.name')}</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => onFormDataChange('name', e.target.value)}
                className="form-control"
                placeholder={t('remotes.form.namePlaceholder')}
                required
              />
            </div>

            <div className="mb-3">
              <label className="form-label">{t('remotes.mode.label')}</label>
              <div className="d-flex gap-2">
                <button
                  type="button"
                  className={`btn btn-sm ${configMode === 'guided' ? 'btn-primary' : 'btn-outline-primary'}`}
                  onClick={onSwitchToGuided}
                >
                  {t('remotes.mode.guided')}
                </button>
                <button
                  type="button"
                  className={`btn btn-sm ${configMode === 'raw' ? 'btn-primary' : 'btn-outline-primary'}`}
                  onClick={onSwitchToRaw}
                >
                  {t('remotes.mode.raw')}
                </button>
              </div>
            </div>

            {configMode === 'guided' ? (
              <>
                <div className="mb-3">
                  <label className="form-label">{t('remotes.preset.label')}</label>
                  <select
                    className="form-select"
                    value={presetKey}
                    onChange={(e) => onPresetChange(e.target.value as RemotePresetKey)}
                  >
                    {Object.values(presets).map((preset) => (
                      <option key={preset.key} value={preset.key}>
                        {preset.label}
                      </option>
                    ))}
                  </select>
                </div>

                {currentPreset.fields.map((field) => {
                  const value = guidedValues[field.key] ?? '';
                  return (
                    <div key={field.key} className="mb-3">
                      <label className="form-label">
                        {field.label}
                        {field.required && <span className="text-danger ms-1">*</span>}
                      </label>

                      {field.kind === 'select' ? (
                        <>
                          <select
                            className="form-select"
                            value={value}
                            onChange={(e) => onGuidedValueChange(field.key, e.target.value)}
                            required={
                              Boolean(field.required) &&
                              !(field.key === 'endpoint' && (presetKey === 's3_alibaba_oss' || presetKey === 's3_tencent_cos'))
                            }
                          >
                            {(() => {
                              const options = field.options || [];
                              const hasValue = value && options.some((opt) => opt.value === value);
                              const normalizedOptions =
                                value && !hasValue
                                  ? [{ value, label: `${t('remotes.values.current')}: ${value}` }, ...options]
                                  : options;

                              return normalizedOptions.map((opt) => (
                                <option key={opt.value || '__empty'} value={opt.value}>
                                  {opt.label}
                                </option>
                              ));
                            })()}
                          </select>

                          {field.key === 'endpoint' &&
                            value === '' &&
                            (presetKey === 's3_alibaba_oss' || presetKey === 's3_tencent_cos') && (
                              <input
                                type="text"
                                className="form-control mt-2"
                                placeholder={t('remotes.placeholders.endpoint_custom')}
                                value={guidedValues.endpoint_custom || ''}
                                onChange={(e) => onGuidedValueChange('endpoint_custom', e.target.value)}
                              />
                            )}
                        </>
                      ) : field.kind === 'textarea' ? (
                        <>
                          <textarea
                            className="form-control font-monospace"
                            rows={6}
                            value={value}
                            placeholder={field.placeholder}
                            onChange={(e) => onGuidedValueChange(field.key, e.target.value)}
                            required={field.required}
                          />

                          {field.key === 'token' && (currentPreset.type === 'drive' || currentPreset.type === 'onedrive') && (
                            <div className="d-flex justify-content-end mt-2">
                              <button
                                type="button"
                                className="btn btn-outline-primary btn-sm"
                                onClick={() => onStartOAuth(currentPreset.type as 'drive' | 'onedrive')}
                                disabled={oauthPending}
                              >
                                <IconKey size={16} />
                                <span className="ms-1">
                                  {oauthPending ? t('remotes.oauth.in_progress') : t('remotes.oauth.button')}
                                </span>
                              </button>
                            </div>
                          )}
                        </>
                      ) : (
                        <input
                          type={field.kind === 'password' ? 'password' : 'text'}
                          className={field.key === 'secret_access_key' ? 'form-control font-monospace' : 'form-control'}
                          value={value}
                          placeholder={field.placeholder}
                          onChange={(e) => onGuidedValueChange(field.key, e.target.value)}
                          required={field.required}
                        />
                      )}

                      {field.help && <div className="form-text">{field.help}</div>}
                    </div>
                  );
                })}

                <div className="mb-3">
                  <label className="form-label">{t('remotes.hints.config_preview')}</label>
                  <textarea className="form-control font-monospace" rows={8} value={previewConfig} readOnly />
                </div>
              </>
            ) : (
              <div className="mb-3">
                <label className="form-label">{t('remotes.form.config')}</label>
                <textarea
                  value={formData.config_data}
                  onChange={(e) => onFormDataChange('config_data', e.target.value)}
                  className="form-control font-monospace"
                  placeholder={t('remotes.hints.raw_help')}
                  rows={10}
                  required
                />
              </div>
            )}
          </>
        )}

        <div className="d-flex justify-content-end gap-2 mt-4">
          <button type="button" className="btn btn-secondary" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn btn-primary" disabled={modalLoading}>
            {isEditing ? t('common.save') : t('common.create')}
          </button>
        </div>
      </form>
    </Modal>
  );
};

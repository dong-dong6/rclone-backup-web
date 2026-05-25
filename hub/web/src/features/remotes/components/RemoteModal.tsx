import React from 'react';
import { Button, Form, Input, Select, Segmented, Space, Typography } from 'antd';
import { KeyOutlined } from '@ant-design/icons';
import { Modal, Loading } from '../../../components/ui';
import { useTranslation } from 'react-i18next';
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

  const handleModeChange = (value: string | number) => {
    if (value === 'guided') {
      onSwitchToGuided();
    } else {
      onSwitchToRaw();
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? t('remotes.edit.title') : t('remotes.create.title')}
      size="lg"
      footer={null}
    >
      <form onSubmit={onSubmit}>
        {modalLoading ? (
          <Loading text={t('common.loading')} />
        ) : (
          <Form layout="vertical">
            <Form.Item label={t('remotes.form.name')} required>
              <Input
                value={formData.name}
                onChange={(e) => onFormDataChange('name', e.target.value)}
                placeholder={t('remotes.form.namePlaceholder')}
                required
              />
            </Form.Item>

            <Form.Item label={t('remotes.mode.label')}>
              <Segmented
                value={configMode}
                onChange={handleModeChange}
                options={[
                  { value: 'guided', label: t('remotes.mode.guided') },
                  { value: 'raw', label: t('remotes.mode.raw') },
                ]}
              />
            </Form.Item>

            {configMode === 'guided' ? (
              <>
                <Form.Item label={t('remotes.preset.label')}>
                  <Select
                    value={presetKey}
                    onChange={(value) => onPresetChange(value as RemotePresetKey)}
                    options={Object.values(presets).map((preset) => ({
                      value: preset.key,
                      label: preset.label,
                    }))}
                  />
                </Form.Item>

                {currentPreset.fields.map((field) => {
                  const value = guidedValues[field.key] ?? '';
                  const required =
                    Boolean(field.required) &&
                    !(field.key === 'endpoint' && (presetKey === 's3_alibaba_oss' || presetKey === 's3_tencent_cos'));

                  return (
                    <Form.Item
                      key={field.key}
                      label={field.label}
                      required={required}
                      extra={field.help}
                    >
                      {field.kind === 'select' ? (
                        <Space direction="vertical" style={{ width: '100%' }}>
                          <Select
                            value={value}
                            onChange={(selected) => onGuidedValueChange(field.key, selected)}
                            options={(() => {
                              const options = field.options || [];
                              const hasValue = value && options.some((opt) => opt.value === value);
                              const normalizedOptions =
                                value && !hasValue
                                  ? [{ value, label: `${t('remotes.values.current')}: ${value}` }, ...options]
                                  : options;
                              return normalizedOptions;
                            })()}
                          />
                          {field.key === 'endpoint' &&
                            value === '' &&
                            (presetKey === 's3_alibaba_oss' || presetKey === 's3_tencent_cos') && (
                              <Input
                                placeholder={t('remotes.placeholders.endpoint_custom')}
                                value={guidedValues.endpoint_custom || ''}
                                onChange={(e) => onGuidedValueChange('endpoint_custom', e.target.value)}
                              />
                            )}
                        </Space>
                      ) : field.kind === 'textarea' ? (
                        <Space direction="vertical" style={{ width: '100%' }}>
                          <Input.TextArea
                            rows={6}
                            value={value}
                            placeholder={field.placeholder}
                            onChange={(e) => onGuidedValueChange(field.key, e.target.value)}
                            required={field.required}
                          />
                          {field.key === 'token' && (currentPreset.type === 'drive' || currentPreset.type === 'onedrive') && (
                            <Button
                              icon={<KeyOutlined />}
                              onClick={() => onStartOAuth(currentPreset.type as 'drive' | 'onedrive')}
                              loading={oauthPending}
                            >
                              {oauthPending ? t('remotes.oauth.in_progress') : t('remotes.oauth.button')}
                            </Button>
                          )}
                        </Space>
                      ) : (
                        <Input
                          type={field.kind === 'password' ? 'password' : 'text'}
                          value={value}
                          placeholder={field.placeholder}
                          onChange={(e) => onGuidedValueChange(field.key, e.target.value)}
                          required={field.required}
                        />
                      )}
                    </Form.Item>
                  );
                })}

                <Form.Item label={t('remotes.hints.config_preview')}>
                  <Input.TextArea rows={8} value={previewConfig} readOnly />
                </Form.Item>
              </>
            ) : (
              <Form.Item label={t('remotes.form.config')} required>
                <Input.TextArea
                  value={formData.config_data}
                  onChange={(e) => onFormDataChange('config_data', e.target.value)}
                  placeholder={t('remotes.hints.raw_help')}
                  rows={10}
                  required
                />
              </Form.Item>
            )}
          </Form>
        )}

        <Space className="rbw-modal-actions">
          <Button onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button type="primary" htmlType="submit" loading={modalLoading}>
            {isEditing ? t('common.save') : t('common.create')}
          </Button>
        </Space>
      </form>
    </Modal>
  );
};

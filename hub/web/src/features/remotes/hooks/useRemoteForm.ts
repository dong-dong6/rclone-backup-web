import { useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { remotesApi } from '../../../services';
import type { RcloneRemote } from '../../../types';
import { createPresets, type RemotePresetKey, type RemotePreset } from '../constants';
import {
  parseRcloneConfig,
  normalizeTokenJson,
  isValidRemoteName,
  buildRcloneConfigSection,
  normalizeRawConfigData,
  detectPreset,
} from '../utils';

export type ConfigMode = 'guided' | 'raw';

export interface RemoteFormState {
  name: string;
  config_data: string;
}

export function useRemoteForm(onSuccess: () => void) {
  const { t } = useTranslation();
  const presets = useMemo(() => createPresets(t), [t]);

  const [editingRemote, setEditingRemote] = useState<RcloneRemote | null>(null);
  const [configMode, setConfigMode] = useState<ConfigMode>('guided');
  const [presetKey, setPresetKey] = useState<RemotePresetKey>('s3_cloudflare_r2');
  const [guidedValues, setGuidedValues] = useState<Record<string, string>>({});
  const [formData, setFormData] = useState<RemoteFormState>({
    name: '',
    config_data: '',
  });
  const [modalLoading, setModalLoading] = useState(false);
  const [isOpen, setIsOpen] = useState(false);

  const currentPreset = presets[presetKey];

  const previewConfig = useMemo(() => {
    if (!(configMode === 'guided' && formData.name.trim())) {
      return formData.config_data;
    }

    const values: Record<string, string> = { ...guidedValues };

    if (presetKey === 's3_alibaba_oss' || presetKey === 's3_tencent_cos') {
      const endpoint = (values.endpoint || '').trim() || (values.endpoint_custom || '').trim();
      values.endpoint = endpoint;
      delete values.endpoint_custom;
    }

    if (currentPreset.type === 'drive' || currentPreset.type === 'onedrive') {
      values.token = normalizeTokenJson(values.token || '');
    }

    return buildRcloneConfigSection(formData.name.trim(), currentPreset.type, {
      ...currentPreset.constantOptions,
      ...values,
    });
  }, [configMode, formData.name, formData.config_data, guidedValues, presetKey, currentPreset]);

  const openCreate = useCallback(() => {
    setEditingRemote(null);
    setConfigMode('guided');
    setPresetKey('s3_cloudflare_r2');
    setGuidedValues(presets['s3_cloudflare_r2'].initialValues);
    setFormData({ name: '', config_data: '' });
    setIsOpen(true);
  }, [presets]);

  const openEdit = useCallback(async (remote: RcloneRemote) => {
    setEditingRemote(remote);
    setModalLoading(true);
    setIsOpen(true);
    setFormData({ name: remote.name, config_data: '' });

    try {
      const detail = await remotesApi.getById(remote.id);
      const parsed = parseRcloneConfig(detail.config_data || '');
      const type = parsed.options.type?.trim();
      const detected = detectPreset(type, parsed.options);

      if (detected) {
        setConfigMode('guided');
        setPresetKey(detected);
        const base = presets[detected].initialValues;
        const merged = { ...base, ...parsed.options };
        setGuidedValues({
          ...merged,
          token: normalizeTokenJson(merged.token || ''),
        });
      } else {
        setConfigMode('raw');
        setPresetKey('s3_cloudflare_r2');
        setGuidedValues({});
      }

      setFormData({ name: detail.name, config_data: detail.config_data || '' });
    } catch (error) {
      console.error('Failed to fetch remote detail:', error);
      alert(t('errors.server'));
      close();
    } finally {
      setModalLoading(false);
    }
  }, [presets, t]);

  const close = useCallback(() => {
    setIsOpen(false);
    setEditingRemote(null);
  }, []);

  const switchToGuided = useCallback(() => {
    setConfigMode('guided');
    setGuidedValues(presets[presetKey].initialValues);
  }, [presets, presetKey]);

  const switchToRaw = useCallback(() => {
    setConfigMode('raw');
  }, []);

  const changePreset = useCallback((newPreset: RemotePresetKey) => {
    setPresetKey(newPreset);
    setGuidedValues(presets[newPreset].initialValues);
  }, [presets]);

  const updateGuidedValue = useCallback((key: string, value: string) => {
    setGuidedValues(prev => ({ ...prev, [key]: value }));
  }, []);

  const updateFormData = useCallback((field: keyof RemoteFormState, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  }, []);

  const submit = useCallback(async (): Promise<boolean> => {
    const remoteName = formData.name.trim();
    if (!remoteName) return false;
    if (!isValidRemoteName(remoteName)) {
      alert(t('remotes.errors.invalid_name'));
      return false;
    }

    let configDataToSave = formData.config_data;
    let remoteTypeToSave: string | undefined;

    if (configMode === 'guided') {
      const preset = presets[presetKey];
      const values: Record<string, string> = { ...guidedValues };

      if (preset.key === 's3_alibaba_oss' || preset.key === 's3_tencent_cos') {
        const endpoint = (values.endpoint || '').trim() || (values.endpoint_custom || '').trim();
        if (!endpoint) {
          alert(t('remotes.errors.endpoint_required'));
          return false;
        }
        values.endpoint = endpoint;
        delete values.endpoint_custom;
      }

      if (preset.type === 'drive' || preset.type === 'onedrive') {
        values.token = normalizeTokenJson(values.token || '');
      }

      configDataToSave = buildRcloneConfigSection(remoteName, preset.type, {
        ...preset.constantOptions,
        ...values,
      });
      remoteTypeToSave = preset.type;
    } else {
      const normalized = normalizeRawConfigData(remoteName, formData.config_data);
      if (!normalized.ok) {
        const message = normalized.error === 'multiple_sections'
          ? t('remotes.errors.multiple_sections')
          : t('remotes.errors.missing_type');
        alert(message);
        return false;
      }
      configDataToSave = normalized.configData;
      remoteTypeToSave = normalized.type;
    }

    const payload = {
      name: remoteName,
      config_data: configDataToSave,
      type: remoteTypeToSave,
      ...(configMode === 'guided' ? { preset_key: presetKey } : {}),
    };

    try {
      if (editingRemote) {
        await remotesApi.update(editingRemote.id, payload);
      } else {
        await remotesApi.create(payload);
      }
      close();
      onSuccess();
      return true;
    } catch (error) {
      console.error('Failed to save remote:', error);
      alert(t('errors.server'));
      return false;
    }
  }, [formData, configMode, presetKey, guidedValues, editingRemote, presets, t, close, onSuccess]);

  return {
    isOpen,
    editingRemote,
    configMode,
    presetKey,
    guidedValues,
    formData,
    modalLoading,
    currentPreset,
    presets,
    previewConfig,
    openCreate,
    openEdit,
    close,
    switchToGuided,
    switchToRaw,
    changePreset,
    updateGuidedValue,
    updateFormData,
    submit,
  };
}

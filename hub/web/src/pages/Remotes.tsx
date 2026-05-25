import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { IconEdit, IconKey, IconPlugConnected, IconPlus, IconRefresh, IconTrash } from '@tabler/icons-react';
import { apiClient } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

interface RcloneRemoteListItem {
  id: string;
  name: string;
  type?: string;
  last_test_at?: string;
  last_test_success?: boolean;
  last_test_message?: string;
  last_test_error?: string;
  last_test_duration_ms?: number;
  created_at: string;
  updated_at: string;
}

interface RcloneRemoteDetail extends RcloneRemoteListItem {
  config_data: string;
}

type ConfigMode = 'guided' | 'raw';

type RemotePresetKey = 'drive' | 'onedrive' | 's3_cloudflare_r2' | 's3_alibaba_oss' | 's3_tencent_cos';

type FieldKind = 'text' | 'password' | 'textarea' | 'select';

type FieldDef = {
  key: string;
  label: string;
  kind: FieldKind;
  placeholder?: string;
  help?: string;
  required?: boolean;
  options?: Array<{ value: string; label: string }>;
};

type RemotePreset = {
  key: RemotePresetKey;
  label: string;
  type: 'drive' | 'onedrive' | 's3';
  constantOptions: Record<string, string>;
  fields: FieldDef[];
  initialValues: Record<string, string>;
};

type ParsedRcloneConfig = {
  sectionNames: string[];
  options: Record<string, string>;
};

type OAuthFlowResponse = {
  flow_id: string;
  start_url: string;
  expires_at: string;
};

type OAuthFlowStatusResponse = {
  status: 'pending' | 'success' | 'error';
  token?: Record<string, unknown>;
  error?: string;
};

type OAuthPopupMessage =
  | {
      type: 'rclone-oauth-result';
      provider: 'drive' | 'onedrive';
      flow_id: string;
      ok: true;
    }
  | {
      type: 'rclone-oauth-result';
      provider: 'drive' | 'onedrive';
      flow_id: string;
      ok: false;
      error: string;
    };

type RemoteTestResponse = {
  success?: boolean;
  message?: string;
  error?: string;
  output?: string;
  duration_ms?: number;
};

const parseRcloneConfig = (configData: string): ParsedRcloneConfig => {
  const sectionNames: string[] = [];
  const options: Record<string, string> = {};

  let currentSectionIndex = -1;
  const lines = configData.split(/\r?\n/);

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line) continue;
    if (line.startsWith('#') || line.startsWith(';')) continue;

    const sectionMatch = line.match(/^\[([^\]]+)\]$/);
    if (sectionMatch) {
      sectionNames.push(sectionMatch[1]);
      currentSectionIndex = sectionNames.length - 1;
      continue;
    }

    const eqIndex = line.indexOf('=');
    if (eqIndex === -1) continue;

    const key = line.slice(0, eqIndex).trim();
    const value = line.slice(eqIndex + 1).trim();
    if (!key) continue;

    // Only parse the first section (or section-less config) for safety.
    if (currentSectionIndex <= 0) {
      options[key] = value;
    }
  }

  return { sectionNames, options };
};

const normalizeTokenJson = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) return '';
  try {
    return JSON.stringify(JSON.parse(trimmed));
  } catch {
    return trimmed;
  }
};

const isValidRemoteName = (name: string) => {
  const trimmed = name.trim();
  if (!trimmed) return false;
  if (/[\r\n\[\]:]/.test(trimmed)) return false;
  if (trimmed.length > 128) return false;
  return true;
};

const buildRcloneConfigSection = (remoteName: string, type: string, options: Record<string, string>) => {
  const lines: string[] = [`[${remoteName}]`, `type = ${type}`];

  for (const [key, value] of Object.entries(options)) {
    const trimmedValue = (value ?? '').toString().trim();
    if (!trimmedValue) continue;
    if (key === 'type') continue;
    lines.push(`${key} = ${trimmedValue}`);
  }

  return lines.join('\n') + '\n';
};

const normalizeRawConfigData = (remoteName: string, rawConfig: string) => {
  const parsed = parseRcloneConfig(rawConfig);
  if (parsed.sectionNames.length > 1) {
    return { ok: false as const, error: 'multiple_sections' as const };
  }

  const type = parsed.options.type?.trim();
  if (!type) {
    return { ok: false as const, error: 'missing_type' as const };
  }

  const options = { ...parsed.options };
  delete options.type;

  return {
    ok: true as const,
    type,
    configData: buildRcloneConfigSection(remoteName, type, options),
  };
};

const detectPreset = (type: string | undefined, options: Record<string, string>): RemotePresetKey | null => {
  if (!type) return null;

  if (type === 'drive') return 'drive';
  if (type === 'onedrive') return 'onedrive';

  if (type === 's3') {
    const provider = options.provider?.trim();
    if (provider === 'Cloudflare') return 's3_cloudflare_r2';
    if (provider === 'Alibaba') return 's3_alibaba_oss';
    if (provider === 'TencentCOS') return 's3_tencent_cos';
  }

  return null;
};

const Remotes: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const [remotes, setRemotes] = useState<RcloneRemoteListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalLoading, setModalLoading] = useState(false);
  const [oauthPending, setOauthPending] = useState(false);
  const [testingRemoteId, setTestingRemoteId] = useState<string | null>(null);
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testModalRemote, setTestModalRemote] = useState<RcloneRemoteListItem | null>(null);
  const [testPath, setTestPath] = useState('');
  const [testSubmitting, setTestSubmitting] = useState(false);
  const [testResult, setTestResult] = useState<RemoteTestResponse | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingRemote, setEditingRemote] = useState<RcloneRemoteListItem | null>(null);
  const [configMode, setConfigMode] = useState<ConfigMode>('guided');
  const [presetKey, setPresetKey] = useState<RemotePresetKey>('s3_cloudflare_r2');
  const [guidedValues, setGuidedValues] = useState<Record<string, string>>({});
  const [formData, setFormData] = useState({
    name: '',
    config_data: '',
  });

  const oauthFlowRef = useRef<{ flowId: string; provider: 'drive' | 'onedrive'; origin: string } | null>(null);
  const oauthPopupRef = useRef<Window | null>(null);
  const oauthPollTimerRef = useRef<number | null>(null);
  const oauthTimeoutTimerRef = useRef<number | null>(null);
  const oauthPollInFlightRef = useRef(false);

  useEffect(() => {
    fetchRemotes();
  }, []);

  useEffect(() => {
    const handler = (event: MessageEvent) => {
      const data = event.data as OAuthPopupMessage;
      if (!data || typeof data !== 'object') return;
      if (data.type !== 'rclone-oauth-result') return;

      const expected = oauthFlowRef.current;
      if (!expected || data.flow_id !== expected.flowId || data.provider !== expected.provider) return;
      if (event.origin !== expected.origin) return;

      if (data.ok) {
        void fetchOAuthFlowResult(expected.provider, expected.flowId);
      } else {
        stopOAuthFlow();
        alert(t('remotes.oauth.failed', { message: data.error || '' }));
      }
    };

    window.addEventListener('message', handler);
    return () => window.removeEventListener('message', handler);
  }, [t, token]);

  const stopOAuthFlow = () => {
    oauthFlowRef.current = null;
    setOauthPending(false);
    oauthPollInFlightRef.current = false;

    if (oauthPollTimerRef.current) {
      window.clearInterval(oauthPollTimerRef.current);
      oauthPollTimerRef.current = null;
    }
    if (oauthTimeoutTimerRef.current) {
      window.clearTimeout(oauthTimeoutTimerRef.current);
      oauthTimeoutTimerRef.current = null;
    }

    const popup = oauthPopupRef.current;
    if (popup && !popup.closed) {
      try {
        popup.close();
      } catch {
        // ignore
      }
    }
    oauthPopupRef.current = null;
  };

  const fetchOAuthFlowResult = async (provider: 'drive' | 'onedrive', flowId: string) => {
    if (oauthPollInFlightRef.current) return;
    oauthPollInFlightRef.current = true;
    try {
      const response = await apiClient.get<OAuthFlowStatusResponse>(`/admin/oauth/${provider}/flow/${flowId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (response.data.status === 'pending') return;

      if (response.data.status === 'success' && response.data.token) {
        setGuidedValues((prev) => ({ ...prev, token: JSON.stringify(response.data.token) }));
        stopOAuthFlow();
        return;
      }

      stopOAuthFlow();
      alert(t('remotes.oauth.failed', { message: response.data.error || '' }));
    } catch (error) {
      console.error('Failed to fetch OAuth result:', error);
    } finally {
      oauthPollInFlightRef.current = false;
    }
  };

  const fetchRemotes = async () => {
    setLoading(true);
    try {
      const response = await apiClient.get('/admin/remotes', {
        headers: { Authorization: `Bearer ${token}` },
      });
      setRemotes(response.data);
    } catch (error) {
      console.error('Failed to fetch remotes:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRemote = () => {
    stopOAuthFlow();
    setEditingRemote(null);
    setConfigMode('guided');
    setPresetKey('s3_cloudflare_r2');
    setGuidedValues({});
    setFormData({
      name: '',
      config_data: '',
    });
    setShowCreateModal(true);
  };

  const handleEditRemote = async (remote: RcloneRemoteListItem) => {
    stopOAuthFlow();
    setEditingRemote(remote);
    setModalLoading(true);
    setShowCreateModal(true);
    setFormData({
      name: remote.name,
      config_data: '',
    });

    try {
      const response = await apiClient.get(`/admin/remotes/${remote.id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const detail: RcloneRemoteDetail = response.data;

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

      setFormData({
        name: detail.name,
        config_data: detail.config_data || '',
      });
    } catch (error) {
      console.error('Failed to fetch remote detail:', error);
      alert(t('errors.server'));
      setShowCreateModal(false);
      setEditingRemote(null);
    } finally {
      setModalLoading(false);
    }
  };

  const handleDeleteRemote = async (remoteId: string) => {
    const remote = remotes.find(r => r.id === remoteId);
    if (!confirm(t('remotes.actions.deleteConfirm', { name: remote?.name || '' }))) return;

    try {
      await apiClient.delete(`/admin/remotes/${remoteId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      fetchRemotes();
    } catch (error) {
      console.error('Failed to delete remote:', error);
      alert(t('errors.server'));
    }
  };

  const openTestModal = (remoteId: string) => {
    const remote = remotes.find((item) => item.id === remoteId);
    if (!remote) return;

    setTestModalRemote(remote);
    setTestPath('');
    setTestResult(null);
    setTestModalOpen(true);
  };

  const closeTestModal = () => {
    if (testSubmitting) return;
    setTestModalOpen(false);
    setTestModalRemote(null);
    setTestResult(null);
    setTestPath('');
  };

  const runRemoteTest = async () => {
    if (!testModalRemote) return;
    if (testSubmitting) return;

    const remoteId = testModalRemote.id;
    const normalizedPath = testPath.trim();

    setTestSubmitting(true);
    setTestingRemoteId(remoteId);
    setTestResult(null);

    try {
      const response = await apiClient.post(
        `/admin/remotes/${remoteId}/test`,
        normalizedPath ? { test_path: normalizedPath } : null,
        {
          headers: { Authorization: `Bearer ${token}` },
        }
      );
      setTestResult(response.data as RemoteTestResponse);
    } catch (error: any) {
      const data = error?.response?.data;
      if (data && typeof data === 'object') {
        setTestResult(data as RemoteTestResponse);
      } else {
        setTestResult({ success: false, message: t('errors.server'), error: String(error) });
      }
    } finally {
      setTestSubmitting(false);
      setTestingRemoteId(null);
      fetchRemotes();
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      const remoteName = formData.name.trim();
      if (!remoteName) return;
      if (!isValidRemoteName(remoteName)) {
        alert(t('remotes.errors.invalid_name'));
        return;
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
            return;
          }
          values.endpoint = endpoint;
          delete values.endpoint_custom;
        }

        if (preset.type === 'drive' || preset.type === 'onedrive') {
          values.token = normalizeTokenJson(values.token || '');
        }

        configDataToSave = buildRcloneConfigSection(remoteName, preset.type, { ...preset.constantOptions, ...values });
        remoteTypeToSave = preset.type;
      } else {
        const normalized = normalizeRawConfigData(remoteName, formData.config_data);
        if (!normalized.ok) {
          const message =
            normalized.error === 'multiple_sections'
              ? t('remotes.errors.multiple_sections')
              : t('remotes.errors.missing_type');
          alert(message);
          return;
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

      if (editingRemote) {
        await apiClient.put(`/admin/remotes/${editingRemote.id}`, payload, {
          headers: { Authorization: `Bearer ${token}` },
        });
      } else {
        await apiClient.post('/admin/remotes', payload, {
          headers: { Authorization: `Bearer ${token}` },
        });
      }

      setShowCreateModal(false);
      fetchRemotes();
    } catch (error) {
      console.error('Failed to save remote:', error);
      alert(t('errors.server'));
    }
  };

  const closeModal = () => {
    stopOAuthFlow();
    setShowCreateModal(false);
    setEditingRemote(null);
  };

  const alibabaEndpoints = [
    { value: 'oss-accelerate.aliyuncs.com', label: 'oss-accelerate.aliyuncs.com' },
    { value: 'oss-accelerate-overseas.aliyuncs.com', label: 'oss-accelerate-overseas.aliyuncs.com' },
    { value: 'oss-cn-hangzhou.aliyuncs.com', label: 'oss-cn-hangzhou.aliyuncs.com' },
    { value: 'oss-cn-shanghai.aliyuncs.com', label: 'oss-cn-shanghai.aliyuncs.com' },
    { value: 'oss-cn-qingdao.aliyuncs.com', label: 'oss-cn-qingdao.aliyuncs.com' },
    { value: 'oss-cn-beijing.aliyuncs.com', label: 'oss-cn-beijing.aliyuncs.com' },
    { value: 'oss-cn-zhangjiakou.aliyuncs.com', label: 'oss-cn-zhangjiakou.aliyuncs.com' },
    { value: 'oss-cn-huhehaote.aliyuncs.com', label: 'oss-cn-huhehaote.aliyuncs.com' },
    { value: 'oss-cn-wulanchabu.aliyuncs.com', label: 'oss-cn-wulanchabu.aliyuncs.com' },
    { value: 'oss-cn-shenzhen.aliyuncs.com', label: 'oss-cn-shenzhen.aliyuncs.com' },
    { value: 'oss-cn-heyuan.aliyuncs.com', label: 'oss-cn-heyuan.aliyuncs.com' },
    { value: 'oss-cn-guangzhou.aliyuncs.com', label: 'oss-cn-guangzhou.aliyuncs.com' },
    { value: 'oss-cn-chengdu.aliyuncs.com', label: 'oss-cn-chengdu.aliyuncs.com' },
    { value: 'oss-cn-hongkong.aliyuncs.com', label: 'oss-cn-hongkong.aliyuncs.com' },
    { value: 'oss-us-west-1.aliyuncs.com', label: 'oss-us-west-1.aliyuncs.com' },
    { value: 'oss-us-east-1.aliyuncs.com', label: 'oss-us-east-1.aliyuncs.com' },
    { value: 'oss-ap-southeast-1.aliyuncs.com', label: 'oss-ap-southeast-1.aliyuncs.com' },
    { value: 'oss-ap-southeast-2.aliyuncs.com', label: 'oss-ap-southeast-2.aliyuncs.com' },
    { value: 'oss-ap-southeast-3.aliyuncs.com', label: 'oss-ap-southeast-3.aliyuncs.com' },
    { value: 'oss-ap-southeast-5.aliyuncs.com', label: 'oss-ap-southeast-5.aliyuncs.com' },
    { value: 'oss-ap-northeast-1.aliyuncs.com', label: 'oss-ap-northeast-1.aliyuncs.com' },
    { value: 'oss-ap-south-1.aliyuncs.com', label: 'oss-ap-south-1.aliyuncs.com' },
    { value: 'oss-eu-central-1.aliyuncs.com', label: 'oss-eu-central-1.aliyuncs.com' },
    { value: 'oss-eu-west-1.aliyuncs.com', label: 'oss-eu-west-1.aliyuncs.com' },
    { value: 'oss-me-east-1.aliyuncs.com', label: 'oss-me-east-1.aliyuncs.com' },
  ];

  const tencentEndpoints = [
    { value: 'cos.ap-beijing.myqcloud.com', label: 'cos.ap-beijing.myqcloud.com' },
    { value: 'cos.ap-nanjing.myqcloud.com', label: 'cos.ap-nanjing.myqcloud.com' },
    { value: 'cos.ap-shanghai.myqcloud.com', label: 'cos.ap-shanghai.myqcloud.com' },
    { value: 'cos.ap-guangzhou.myqcloud.com', label: 'cos.ap-guangzhou.myqcloud.com' },
    { value: 'cos.ap-chengdu.myqcloud.com', label: 'cos.ap-chengdu.myqcloud.com' },
    { value: 'cos.ap-chongqing.myqcloud.com', label: 'cos.ap-chongqing.myqcloud.com' },
    { value: 'cos.ap-hongkong.myqcloud.com', label: 'cos.ap-hongkong.myqcloud.com' },
    { value: 'cos.ap-singapore.myqcloud.com', label: 'cos.ap-singapore.myqcloud.com' },
    { value: 'cos.ap-mumbai.myqcloud.com', label: 'cos.ap-mumbai.myqcloud.com' },
    { value: 'cos.ap-seoul.myqcloud.com', label: 'cos.ap-seoul.myqcloud.com' },
    { value: 'cos.ap-bangkok.myqcloud.com', label: 'cos.ap-bangkok.myqcloud.com' },
    { value: 'cos.ap-tokyo.myqcloud.com', label: 'cos.ap-tokyo.myqcloud.com' },
    { value: 'cos.na-siliconvalley.myqcloud.com', label: 'cos.na-siliconvalley.myqcloud.com' },
    { value: 'cos.na-ashburn.myqcloud.com', label: 'cos.na-ashburn.myqcloud.com' },
    { value: 'cos.na-toronto.myqcloud.com', label: 'cos.na-toronto.myqcloud.com' },
    { value: 'cos.eu-frankfurt.myqcloud.com', label: 'cos.eu-frankfurt.myqcloud.com' },
    { value: 'cos.eu-moscow.myqcloud.com', label: 'cos.eu-moscow.myqcloud.com' },
    { value: 'cos.accelerate.myqcloud.com', label: 'cos.accelerate.myqcloud.com' },
  ];

  const presets: Record<RemotePresetKey, RemotePreset> = {
    drive: {
      key: 'drive',
      label: t('remotes.preset.drive'),
      type: 'drive',
      constantOptions: {},
      initialValues: {
        client_id: '',
        client_secret: '',
        scope: '',
        token: '',
        team_drive: '',
        root_folder_id: '',
      },
      fields: [
        { key: 'client_id', label: t('remotes.fields.client_id'), kind: 'text', placeholder: t('remotes.placeholders.client_id') },
        { key: 'client_secret', label: t('remotes.fields.client_secret'), kind: 'password', placeholder: t('remotes.placeholders.client_secret') },
        {
          key: 'scope',
          label: t('remotes.fields.scope'),
          kind: 'select',
          options: [
            { value: '', label: t('remotes.values.default') },
            { value: 'drive', label: 'drive' },
            { value: 'drive.readonly', label: 'drive.readonly' },
            { value: 'drive.file', label: 'drive.file' },
            { value: 'drive.appfolder', label: 'drive.appfolder' },
            { value: 'drive.metadata.readonly', label: 'drive.metadata.readonly' },
          ],
        },
        {
          key: 'token',
          label: t('remotes.fields.token'),
          kind: 'textarea',
          required: true,
          help: t('remotes.hints.token_drive'),
          placeholder: '{"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}',
        },
        { key: 'team_drive', label: t('remotes.fields.team_drive'), kind: 'text', placeholder: t('remotes.placeholders.team_drive') },
        { key: 'root_folder_id', label: t('remotes.fields.root_folder_id'), kind: 'text', placeholder: t('remotes.placeholders.root_folder_id') },
      ],
    },
    onedrive: {
      key: 'onedrive',
      label: t('remotes.preset.onedrive'),
      type: 'onedrive',
      constantOptions: {},
      initialValues: {
        client_id: '',
        client_secret: '',
        region: 'global',
        token: '',
        drive_type: '',
        drive_id: '',
        root_folder_id: '',
      },
      fields: [
        { key: 'client_id', label: t('remotes.fields.client_id'), kind: 'text', placeholder: t('remotes.placeholders.client_id') },
        { key: 'client_secret', label: t('remotes.fields.client_secret'), kind: 'password', placeholder: t('remotes.placeholders.client_secret') },
        {
          key: 'region',
          label: t('remotes.fields.region'),
          kind: 'select',
          options: [
            { value: 'global', label: 'global' },
            { value: 'us', label: 'us' },
            { value: 'de', label: 'de' },
            { value: 'cn', label: 'cn' },
          ],
        },
        {
          key: 'token',
          label: t('remotes.fields.token'),
          kind: 'textarea',
          required: true,
          help: t('remotes.hints.token_onedrive'),
          placeholder: '{"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}',
        },
        { key: 'drive_type', label: t('remotes.fields.drive_type'), kind: 'text', placeholder: 'personal | business | documentLibrary' },
        { key: 'drive_id', label: t('remotes.fields.drive_id'), kind: 'text' },
        { key: 'root_folder_id', label: t('remotes.fields.root_folder_id'), kind: 'text', placeholder: t('remotes.placeholders.root_folder_id') },
      ],
    },
    s3_cloudflare_r2: {
      key: 's3_cloudflare_r2',
      label: t('remotes.preset.cloudflare_r2'),
      type: 's3',
      constantOptions: { provider: 'Cloudflare', region: 'auto', no_check_bucket: 'true' },
      initialValues: { access_key_id: '', secret_access_key: '', endpoint: '' },
      fields: [
        { key: 'access_key_id', label: t('remotes.fields.access_key_id'), kind: 'text', required: true },
        { key: 'secret_access_key', label: t('remotes.fields.secret_access_key'), kind: 'password', required: true },
        { key: 'endpoint', label: t('remotes.fields.endpoint'), kind: 'text', required: true, placeholder: 'https://<ACCOUNT_ID>.r2.cloudflarestorage.com' },
      ],
    },
    s3_alibaba_oss: {
      key: 's3_alibaba_oss',
      label: t('remotes.preset.aliyun_oss'),
      type: 's3',
      constantOptions: { provider: 'Alibaba' },
      initialValues: { access_key_id: '', secret_access_key: '', endpoint: '' },
      fields: [
        { key: 'access_key_id', label: t('remotes.fields.access_key_id'), kind: 'text', required: true },
        { key: 'secret_access_key', label: t('remotes.fields.secret_access_key'), kind: 'password', required: true },
        { key: 'endpoint', label: t('remotes.fields.endpoint'), kind: 'select', required: true, options: [{ value: '', label: t('remotes.values.custom') }, ...alibabaEndpoints] },
      ],
    },
    s3_tencent_cos: {
      key: 's3_tencent_cos',
      label: t('remotes.preset.tencent_cos'),
      type: 's3',
      constantOptions: { provider: 'TencentCOS' },
      initialValues: { access_key_id: '', secret_access_key: '', endpoint: '' },
      fields: [
        { key: 'access_key_id', label: t('remotes.fields.access_key_id'), kind: 'text', required: true },
        { key: 'secret_access_key', label: t('remotes.fields.secret_access_key'), kind: 'password', required: true },
        { key: 'endpoint', label: t('remotes.fields.endpoint'), kind: 'select', required: true, options: [{ value: '', label: t('remotes.values.custom') }, ...tencentEndpoints] },
      ],
    },
  };

  const currentPreset = presets[presetKey];
  const previewConfig = (() => {
    if (!(configMode === 'guided' && formData.name.trim())) return formData.config_data;

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
  })();

  const startOAuth = async (provider: 'drive' | 'onedrive') => {
    if (oauthPending) return;

    const clientId = (guidedValues.client_id || '').trim();
    if (!clientId) {
      alert(t('remotes.oauth.missing_client_id'));
      return;
    }

    setOauthPending(true);

    try {
      const payload: Record<string, string> = {
        client_id: clientId,
        client_secret: (guidedValues.client_secret || '').trim(),
      };

      if (provider === 'drive') {
        payload.scope = (guidedValues.scope || '').trim();
      } else {
        payload.region = (guidedValues.region || '').trim() || 'global';
      }

      const response = await apiClient.post<OAuthFlowResponse>(`/admin/oauth/${provider}/flow`, payload, {
        headers: { Authorization: `Bearer ${token}` },
      });

      const origin = new URL(response.data.start_url, window.location.href).origin;
      oauthFlowRef.current = { flowId: response.data.flow_id, provider, origin };

      const popup = window.open(response.data.start_url, 'rclone-oauth', 'width=520,height=720');
      if (!popup) {
        stopOAuthFlow();
        alert(t('remotes.oauth.popup_blocked'));
        return;
      }

      oauthPopupRef.current = popup;
      popup.focus();

      if (oauthPollTimerRef.current) {
        window.clearInterval(oauthPollTimerRef.current);
      }
      oauthPollTimerRef.current = window.setInterval(() => {
        const current = oauthFlowRef.current;
        if (!current) return;

        void fetchOAuthFlowResult(current.provider, current.flowId);

        const w = oauthPopupRef.current;
        if (w && w.closed) {
          oauthPopupRef.current = null;
        }
      }, 1000);

      if (oauthTimeoutTimerRef.current) {
        window.clearTimeout(oauthTimeoutTimerRef.current);
      }
      oauthTimeoutTimerRef.current = window.setTimeout(() => {
        if (!oauthFlowRef.current) return;
        stopOAuthFlow();
        alert(t('remotes.oauth.timeout'));
      }, 2 * 60 * 1000);
    } catch (error) {
      stopOAuthFlow();
      console.error('Failed to start OAuth:', error);
      alert(t('remotes.oauth.start_failed'));
    }
  };

  useEffect(() => {
    if (!showCreateModal) return;
    if (configMode !== 'guided') return;
    const preset = presets[presetKey];
    if (Object.keys(guidedValues).length === 0) {
      setGuidedValues(preset.initialValues);
    }
  }, [showCreateModal, configMode, presetKey]);

  if (loading) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body text-center py-5">
              <IconRefresh className="spinner text-primary mb-3" size={48} />
              <p className="text-muted">{t('common.loading')}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="row row-deck row-cards">
      <div className="col-12">
        <div className="card">
          <div className="card-body d-flex justify-content-end gap-2">
            <button onClick={fetchRemotes} className="btn btn-outline-primary">
              <IconRefresh size={16} />
              <span className="ms-1">{t('common.refresh')}</span>
            </button>
            <button onClick={handleCreateRemote} className="btn btn-primary">
              <IconPlus size={16} />
              <span className="ms-1">{t('remotes.create.title')}</span>
            </button>
          </div>
        </div>
      </div>

      {remotes.length > 0 ? (
        remotes.map((remote) => (
          <div key={remote.id} className="col-12 col-md-6 col-xl-4">
            <div className="card">
              <div className="card-header">
                <div>
                  <h3 className="card-title mb-1">{remote.name}</h3>
                  <div className="d-flex flex-wrap gap-1">
                    {remote.type && <span className="badge bg-secondary text-white">{remote.type}</span>}
                    <span
                      className={`badge ${
                        remote.last_test_success === true
                          ? 'bg-success text-white'
                          : remote.last_test_success === false
                            ? 'bg-danger text-white'
                            : 'bg-secondary text-white'
                      }`}
                      title={remote.last_test_error || remote.last_test_message || t('remotes.actions.test')}
                    >
                      {remote.last_test_success === true
                        ? t('common.success')
                        : remote.last_test_success === false
                          ? t('common.failed')
                          : t('common.never')}
                    </span>
                  </div>
                </div>
                <div className="ms-auto d-flex gap-2">
	                  <button
	                    onClick={() => openTestModal(remote.id)}
	                    className="btn btn-outline-secondary btn-sm"
	                    title={t('remotes.actions.test')}
	                    disabled={testingRemoteId === remote.id}
	                  >
                    {testingRemoteId === remote.id ? (
                      <IconRefresh className="spinner" size={16} />
                    ) : (
                      <IconPlugConnected size={16} />
                    )}
                  </button>
                  <button
                    onClick={() => handleEditRemote(remote)}
                    className="btn btn-outline-primary btn-sm"
                    title={t('common.edit')}
                  >
                    <IconEdit size={16} />
                  </button>
                  <button
                    onClick={() => handleDeleteRemote(remote.id)}
                    className="btn btn-outline-danger btn-sm"
                    title={t('common.delete')}
                  >
                    <IconTrash size={16} />
                  </button>
                </div>
              </div>
              <div className="card-body">
                <div className="text-muted small">
                  {t('remotes.list.columns.createdAt')}: {new Date(remote.created_at).toLocaleString()}
                </div>
                <div className="text-muted small">
                  {t('remotes.list.columns.lastTest')}:{' '}
                  {remote.last_test_at ? new Date(remote.last_test_at).toLocaleString() : t('common.never')}
                </div>
              </div>
            </div>
          </div>
        ))
	      ) : (
	        <div className="col-12">
	          <div className="card">
	            <div className="card-body text-center py-5">
	              <p className="text-muted mb-3">{t('remotes.list.empty')}</p>
	              <button onClick={handleCreateRemote} className="btn btn-primary">
	                <IconPlus size={16} />
	                <span className="ms-1">{t('remotes.create.title')}</span>
	              </button>
	            </div>
	          </div>
	        </div>
	      )}

	      {testModalOpen && testModalRemote && (
	        <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1} role="dialog">
	          <div className="modal-dialog modal-md modal-dialog-centered modal-dialog-scrollable" role="document">
	            <div className="modal-content">
	              <div className="modal-header">
	                <h5 className="modal-title">
	                  {t('remotes.actions.test')}: {testModalRemote.name}
	                </h5>
	                <button type="button" className="btn-close" onClick={closeTestModal} aria-label={t('common.close')}></button>
	              </div>

	              <div className="modal-body">
	                {testModalRemote.type === 's3' && (
	                  <div className="mb-3">
	                    <label className="form-label">{t('remotes.test.pathLabel')}</label>
	                    <input
	                      type="text"
	                      className="form-control"
	                      value={testPath}
	                      onChange={(e) => setTestPath(e.target.value)}
	                      placeholder={t('remotes.test.pathPlaceholder')}
	                      disabled={testSubmitting}
	                    />
	                    <div className="form-text">{t('remotes.test.pathPromptS3')}</div>
	                  </div>
	                )}

	                {testResult && (
	                  <div className={`alert ${testResult.success ? 'alert-success' : 'alert-danger'}`} role="alert">
	                    <div className="fw-semibold mb-1">
	                      {testResult.success ? t('common.success') : t('common.failed')}
	                      {typeof testResult.duration_ms === 'number' ? ` (${testResult.duration_ms}ms)` : ''}
	                    </div>
	                    {testResult.message && <div className="mb-2">{testResult.message}</div>}

	                    {testResult.error && (
	                      <div className="mb-2">
	                        <div className="fw-semibold">{t('common.error')}</div>
	                        <div className="font-monospace small">{testResult.error}</div>
	                      </div>
	                    )}

	                    {testResult.output && (
	                      <div>
	                        <div className="fw-semibold">{t('remotes.test.outputLabel')}</div>
	                        <textarea className="form-control font-monospace mt-1" rows={6} value={testResult.output} readOnly />
	                      </div>
	                    )}
	                  </div>
	                )}
	              </div>

	              <div className="modal-footer">
	                <button type="button" className="btn btn-outline-secondary" onClick={closeTestModal} disabled={testSubmitting}>
	                  {t('common.cancel')}
	                </button>
	                <button type="button" className="btn btn-primary" onClick={runRemoteTest} disabled={testSubmitting}>
	                  {testSubmitting ? (
	                    <>
	                      <IconRefresh className="spinner" size={16} />
	                      <span className="ms-1">{t('remotes.test.running')}</span>
	                    </>
	                  ) : (
	                    <>
	                      <IconPlugConnected size={16} />
	                      <span className="ms-1">{t('remotes.actions.test')}</span>
	                    </>
	                  )}
	                </button>
	              </div>
	            </div>
	          </div>
	        </div>
	      )}

	      {showCreateModal && (
	        <div className="modal modal-blur fade show" style={{ display: 'block' }} tabIndex={-1} role="dialog">
	          <div className="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable" role="document">
	            <div className="modal-content">
	              <form onSubmit={handleSubmit}>
                <div className="modal-header">
                  <h5 className="modal-title">
                    {editingRemote ? t('remotes.edit.title') : t('remotes.create.title')}
                  </h5>
                  <button type="button" className="btn-close" onClick={closeModal} aria-label={t('common.close')}></button>
                </div>

                <div className="modal-body">
                  {modalLoading ? (
                    <div className="text-center py-5">
                      <IconRefresh className="spinner text-primary mb-3" size={48} />
                      <p className="text-muted">{t('common.loading')}</p>
                    </div>
                  ) : (
                    <>
                      <div className="mb-3">
                        <label className="form-label">{t('remotes.form.name')}</label>
                        <input
                          type="text"
                          value={formData.name}
                          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
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
                            onClick={() => {
                              setConfigMode('guided');
                              setGuidedValues(presets[presetKey].initialValues);
                            }}
                          >
                            {t('remotes.mode.guided')}
                          </button>
                          <button
                            type="button"
                            className={`btn btn-sm ${configMode === 'raw' ? 'btn-primary' : 'btn-outline-primary'}`}
                            onClick={() => setConfigMode('raw')}
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
                              onChange={(e) => {
                                const next = e.target.value as RemotePresetKey;
                                setPresetKey(next);
                                setGuidedValues(presets[next].initialValues);
                              }}
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
                                      onChange={(e) => setGuidedValues({ ...guidedValues, [field.key]: e.target.value })}
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
                                          onChange={(e) => setGuidedValues({ ...guidedValues, endpoint_custom: e.target.value })}
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
                                      onChange={(e) => setGuidedValues({ ...guidedValues, [field.key]: e.target.value })}
                                      required={field.required}
                                    />

                                    {field.key === 'token' && (currentPreset.type === 'drive' || currentPreset.type === 'onedrive') && (
                                      <div className="d-flex justify-content-end mt-2">
                                        <button
                                          type="button"
                                          className="btn btn-outline-primary btn-sm"
                                          onClick={() => startOAuth(currentPreset.type as 'drive' | 'onedrive')}
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
                                    onChange={(e) => setGuidedValues({ ...guidedValues, [field.key]: e.target.value })}
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
                            onChange={(e) => setFormData({ ...formData, config_data: e.target.value })}
                            className="form-control font-monospace"
                            placeholder={t('remotes.hints.raw_help')}
                            rows={10}
                            required
                          />
                        </div>
                      )}
                    </>
                  )}
                </div>

                <div className="modal-footer">
                  <button type="button" className="btn btn-secondary" onClick={closeModal}>
                    {t('common.cancel')}
                  </button>
                  <button type="submit" className="btn btn-primary">
                    {editingRemote ? t('common.save') : t('common.create')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Remotes;
